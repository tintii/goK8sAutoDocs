const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { spawnSync } = require('node:child_process');
const { pathToFileURL } = require('node:url');

const candidates = [process.env.CHROME_BIN, '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome', '/usr/bin/chromium', '/usr/bin/chromium-browser', '/usr/bin/google-chrome'];
const chrome = candidates.find(candidate => candidate && fs.existsSync(candidate));
const page = fs.readFileSync(path.join(__dirname, 'autodocs-dashboard.html'), 'utf8');
const script = fs.readFileSync(path.join(__dirname, 'autodocs-dashboard.js'), 'utf8');

function checkInBrowser(t, data, warnings, assertions, status = 200, options = {}) {
	if (!chrome) return t.skip('Set CHROME_BIN to run the browser fixture checks');
	const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'autodocs-browser-'));
	t.after(() => fs.rmSync(directory, { recursive: true, force: true }));
	const setup = `
		window.fetch = async (url) => url.includes('cluster-info')
			? {ok: true, json: async () => ({clusterName: 'fixture'})}
			: {ok: ${status === 200}, status: ${status}, headers: {get: () => ${JSON.stringify(JSON.stringify(warnings))}}, json: async () => ${JSON.stringify(data)}};
		function check(value, message) { if (!value) throw new Error(message); }
		const tick = () => new Promise(resolve => setTimeout(resolve, 0));
		${options.setup || ''}
	`;
	const checks = `document.addEventListener('DOMContentLoaded', () => setTimeout(async () => {
		try {
			${assertions}
			document.getElementById('test-result').textContent = 'PASS';
		} catch (error) { document.getElementById('test-result').textContent = 'FAIL: ' + error.stack; }
	}, 0));`;
	const html = page.replace('<script src="/static/autodocs-dashboard.js" defer></script>',
		`<pre id="test-result">PENDING</pre><script>${setup}</script><script>${script}</script><script>${checks}</script>`);
	const file = path.join(directory, 'fixture.html');
	fs.writeFileSync(file, html);
	const result = spawnSync(chrome, ['--headless', '--disable-gpu', '--no-first-run', '--no-default-browser-check', '--disable-background-networking', `--user-data-dir=${path.join(directory, 'profile')}`, '--dump-dom', '--virtual-time-budget=2000', pathToFileURL(file).href + (options.query || '')], { encoding: 'utf8', timeout: 20000, maxBuffer: 2 * 1024 * 1024 });
	assert.equal(result.error, undefined, result.error?.message);
	assert.equal(result.status, 0, result.stderr);
	const marker = result.stdout.match(/<pre id="test-result">([\s\S]*?)<\/pre>/)?.[1];
	assert.equal(marker, 'PASS', marker || result.stderr);
}

test('mixed directory uses resolved URLs, namespaces, kinds and safe text', t => {
	checkInBrowser(t, [
		{name: 'legacy', namespace: 'default', kind: 'ingress', hostnames: [{host: 'legacy.example.com'}], endpoints: [{url: 'https://legacy.example.com', host: 'legacy.example.com'}]},
		{name: '<img src=x onerror="window.injected=true">', namespace: 'media', kind: 'HTTPRoute', endpoints: [{url: 'https://app.example.com:8443', gateway: {namespace: 'edge', name: 'shared'}, listener: 'https'}], relatedResources: [{name: 'backend', namespace: 'media', kind: 'Service'}]},
	], [], `
		check(document.querySelectorAll('.endpoint-button').length === 2, 'two source cards');
		check(document.getElementById('total-resources').textContent === '2', 'resource count');
		check(document.getElementById('total-hosts').textContent === '2', 'endpoint count');
		check(document.getElementById('total-namespaces').textContent === '2', 'namespace count');
		const links = [...document.querySelectorAll('.endpoint-link')];
		check(links[1].getAttribute('href') === 'https://app.example.com:8443', 'exact resolved URL');
		check(links.every(link => link.target === '_blank' && link.rel === 'noopener noreferrer'), 'new tab attributes');
		check(!document.querySelector('.endpoint-button img') && !window.injected, 'metadata is text');
		performSearch('httproute');
		check(document.querySelectorAll('.endpoint-button').length === 1, 'search source kind');
		performSearch('edge');
		check(document.querySelectorAll('.endpoint-button').length === 1, 'search Gateway namespace');
		performSearch('backend');
		check(document.querySelectorAll('.endpoint-button').length === 1, 'search related resources');
		performSearch('unmatched');
		check(document.querySelector('.empty-directory').textContent.includes('No matching'), 'empty search');
	`);
});

test('unresolved resources and collection warnings remain visible', t => {
	checkInBrowser(t, [{name: 'wildcard', namespace: 'media', kind: 'HTTPRoute', endpoints: [], issues: ['No concrete hostname'], relatedResources: []}], ['Cannot list Ingresses: forbidden'], `
		check(document.querySelectorAll('.endpoint-button').length === 1, 'unresolved card');
		check(document.querySelectorAll('.endpoint-link').length === 0, 'no invented link');
		check(document.querySelector('.endpoint-issues').textContent.includes('No concrete hostname'), 'issue visible');
		check(!document.getElementById('collection-warnings').hidden, 'collection warning visible');
		check(document.getElementById('collection-warnings').textContent.includes('forbidden'), 'warning reason');
	`);
});

test('empty directory renders zero counts', t => {
	checkInBrowser(t, [], [], `
		check(document.querySelector('.empty-directory').textContent === 'No published resources found.', 'empty directory');
		check(document.getElementById('total-resources').textContent === '0', 'zero resources');
		check(document.getElementById('total-hosts').textContent === '0', 'zero URLs');
		check(document.getElementById('collection-warnings').hidden, 'no warning');
	`);
});

test('failed requests display an error', t => {
	checkInBrowser(t, [], [], `
		check(document.getElementById('loading').textContent.includes('Request failed (500)'), 'request error');
	`, 500);
});

test('groups span namespaces and preserve exact names and a distinct fallback', t => {
	const resource = (name, namespace, group) => ({name, namespace, kind: 'HTTPRoute', labels: group === undefined ? {} : {'k8sautodocs.io/group': group}, endpoints: [{url: `https://${name}.example.com`}, {url: `http://${name}.example.com`}]});
	const unsafeGroup = '<img src=x onerror="window.injected=true">';
	checkInBrowser(t, [resource('second', 'z', 'media'), resource('first', 'a', 'media'), resource('capital', 'a', 'Media'), resource('literal', 'a', 'Ungrouped'), resource('missing', 'a'), resource('empty', 'a', ''), resource('unsafe', 'a', unsafeGroup)], [], `
		const sections = [...document.querySelectorAll('section.namespace-section')];
		check(sections.length === 5, 'four named groups plus fallback');
		check(sections.map(section => section.querySelector('h2').textContent).join('|') === ${JSON.stringify([unsafeGroup, 'Media', 'Ungrouped', 'media', 'Ungrouped'].join('|'))}, 'exact ordered group headings');
		check(sections[2].querySelectorAll('.endpoint-button').length === 1, 'literal group separate');
		check(sections[4].querySelectorAll('.endpoint-button').length === 2, 'missing and empty group share fallback');
		check([...sections[3].querySelectorAll('.endpoint-name')].map(node => node.textContent).join(',') === 'first,second', 'namespace ordering inside cross-namespace group');
		check(sections[3].textContent.includes('Namespace: a') && sections[3].textContent.includes('Namespace: z'), 'namespace context retained');
		check(document.querySelectorAll('.endpoint-button').length === 7 && document.querySelectorAll('.endpoint-link').length === 14, 'one card per source with all URLs');
		check(!document.querySelector('section img') && !window.injected, 'group values rendered as text');
		performSearch('media');
		check(document.querySelectorAll('.endpoint-button').length === 3, 'search group values case-insensitively');
		check(document.querySelectorAll('section.namespace-section').length === 2, 'omit empty groups');
	`);
});

const viewSetup = `
	const published = [{name:'curated', namespace:'apps', kind:'ingress', labels:{'k8sautodocs.io/group':'media'}, endpoints:[{url:'https://curated.example.com'}]}];
	const discovered = [{name:'candidate', namespace:'other', kind:'HTTPRoute', labels:{'k8sautodocs.io/group':'media'}, endpoints:[], issues:['Missing parent']}, {name:'another', namespace:'other', kind:'ingress', endpoints:[]}];
	window.fetch = async url => url.includes('cluster-info') ? {ok:true,json:async()=>({clusterName:'fixture'})}
		: {ok:true, headers:{get:()=>url.includes('discovered') ? ' ["Discovered warning"] ' : '[]'}, json:async()=>url.includes('discovered') ? discovered : published};
`;

test('tabs preserve search and scope counts, warnings, keyboard selection and history', t => {
	checkInBrowser(t, [], [], `
		const search = document.getElementById('search-input');
		search.value = 'media'; performSearch(search.value);
		await selectView('discovered');
		check(search.value === 'media', 'query retained');
		check(document.querySelectorAll('.endpoint-button').length === 1, 'query applied to discovery');
		check(document.querySelectorAll('section.namespace-section').length === 0, 'flat discovery even with group label');
		check(document.getElementById('total-resources').textContent === '2', 'active view resource count');
		check(document.getElementById('total-namespaces').textContent === '1', 'active view namespace count');
		check(document.getElementById('total-hosts').textContent === '0', 'active view URL count');
		check(document.getElementById('filtered-count').textContent === 'Showing 1 of 2 resources', 'filtered subset count');
		check(document.getElementById('collection-warnings').textContent.includes('Discovered warning'), 'own warnings');
		check(document.querySelector('.endpoint-issues').textContent.includes('Missing parent'), 'unresolved discovered issue');
		check(new URL(location.href).searchParams.get('view') === 'discovered', 'bookmarkable view');
		check(document.getElementById('directory-api-link').getAttribute('href').endsWith('view=discovered'), 'view-specific JSON link');
		const discoveredTab = document.getElementById('discovered-tab');
		check(discoveredTab.getAttribute('aria-selected') === 'true' && discoveredTab.tabIndex === 0, 'accessible selected state');
		discoveredTab.dispatchEvent(new KeyboardEvent('keydown', {key:'Home', bubbles:true})); await tick();
		check(document.activeElement.id === 'published-tab', 'Home focuses Applications');
		check(document.getElementById('published-tab').getAttribute('aria-selected') === 'true', 'Home selects Applications');
		check(document.getElementById('collection-warnings').hidden, 'previous view warning cleared');
		document.getElementById('published-tab').dispatchEvent(new KeyboardEvent('keydown', {key:'ArrowRight', bubbles:true})); await tick();
		check(document.activeElement.id === 'discovered-tab', 'ArrowRight selects next tab');
		history.back();
		for (let i=0; i<10 && activeView !== 'published'; i++) await tick();
		check(activeView === 'published', 'browser back restores Applications');
		history.forward();
		for (let i=0; i<10 && activeView !== 'discovered'; i++) await tick();
		check(activeView === 'discovered', 'browser forward restores Discovered');
	`, 200, {setup: viewSetup});
});

test('direct discovered URL restores selection and handles empty discovered results', t => {
	checkInBrowser(t, [], [], `
		check(activeView === 'discovered' && document.getElementById('discovered-tab').getAttribute('aria-selected') === 'true', 'direct-link selection');
		check(document.querySelector('.empty-directory').textContent === 'No discovered resources found.', 'discovered empty message');
		check(document.getElementById('total-resources').textContent === '0', 'empty active counts');
	`, 200, {query:'?view=discovered'});
});

test('unknown page view falls back to Applications', t => {
	checkInBrowser(t, [], [], `check(activeView === 'published', 'unknown page view defaults to Applications');`, 200, {query:'?view=unknown'});
});

test('late previous responses and failures cannot overwrite the selected view', t => {
	checkInBrowser(t, [], [], `
		check(pending.length === 1, 'initial Applications request pending');
		const discoveryLoad = selectView('discovered');
		check(document.querySelectorAll('.endpoint-button').length === 0 && document.getElementById('loading').style.display === 'block', 'clear old cards while loading');
		pending[1].resolve(response([{name:'found', namespace:'apps', kind:'HTTPRoute', endpoints:[]}], ['discovered warning']));
		await discoveryLoad;
		pending[0].resolve(response([{name:'stale', namespace:'apps', kind:'ingress', endpoints:[]}], ['stale warning'])); await tick();
		check(document.querySelector('.endpoint-name').textContent === 'found', 'ignore late Applications response');
		check(!document.getElementById('collection-warnings').textContent.includes('stale'), 'ignore stale warning');
		const failureLoad = selectView('published');
		check(document.getElementById('collection-warnings').hidden && !document.querySelector('.endpoint-button'), 'clear prior cards and warnings');
		pending[2].reject(new Error('offline')); await failureLoad;
		check(document.getElementById('loading').textContent.includes('Error loading Applications: offline'), 'active view error');
		check(!document.querySelector('.endpoint-button'), 'no other view stale results on failure');
	`, 200, {setup:`
		const pending = [];
		const response = (data, warnings=[]) => ({ok:true, headers:{get:()=>JSON.stringify(warnings)}, json:async()=>data});
		window.fetch = url => url.includes('cluster-info') ? Promise.resolve({ok:true,json:async()=>({clusterName:'fixture'})}) : new Promise((resolve,reject)=>pending.push({url,resolve,reject}));
	`});
});
