const { test } = require('node:test');
const assert = require('node:assert/strict');
const { browserFixture, chrome } = require('../../scripts/browser-fixture.cjs');

async function checkInBrowser(t, data, warnings, assertions, status = 200, options = {}) {
	if (!chrome) return t.skip('Set CHROME_BIN to run the browser fixture checks');
	const setup = [
		'window.fetch = async (url) => url.includes("cluster-info")',
		'? {ok: true, json: async () => ({clusterName: "fixture"})}',
		': {ok: ' + (status === 200) + ', status: ' + status + ', headers: {get: () => ' + JSON.stringify(JSON.stringify(warnings)) + '}, json: async () => ' + JSON.stringify(data) + '};',
		'function check(value, message) { if (!value) throw new Error(message); }',
		'const tick = () => new Promise(resolve => setTimeout(resolve, 0));',
		options.setup || ''
	].join('\n');
	const browser = await browserFixture(t, {setup, query: options.query});
	const result = await browser.evaluate('(async () => { try { ' + assertions + '; return "PASS"; } catch (error) { return "FAIL: " + error.stack; } })()');
	assert.equal(result, 'PASS', result);
}

test('mixed directory uses resolved URLs, namespaces, kinds and safe text', t => {
	return checkInBrowser(t, [
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
	return checkInBrowser(t, [{name: 'wildcard', namespace: 'media', kind: 'HTTPRoute', endpoints: [], issues: ['No concrete hostname'], relatedResources: []}], ['Cannot list Ingresses: forbidden'], `
		check(document.querySelectorAll('.endpoint-button').length === 1, 'unresolved card');
		check(document.querySelectorAll('.endpoint-link').length === 0, 'no invented link');
		check(document.querySelector('.endpoint-issues').textContent.includes('No concrete hostname'), 'issue visible');
		check(!document.getElementById('collection-warnings').hidden, 'collection warning visible');
		check(document.getElementById('collection-warnings').textContent.includes('forbidden'), 'warning reason');
	`);
});

test('empty directory renders zero counts', t => {
	return checkInBrowser(t, [], [], `
		check(document.querySelector('.empty-directory').textContent === 'No published resources found.', 'empty directory');
		check(document.getElementById('total-resources').textContent === '0', 'zero resources');
		check(document.getElementById('total-hosts').textContent === '0', 'zero URLs');
		check(document.getElementById('collection-warnings').hidden, 'no warning');
	`);
});

test('failed requests display an error', t => {
	return checkInBrowser(t, [], [], `
		check(document.getElementById('loading').textContent.includes('Request failed (500)'), 'request error');
	`, 500);
});

test('groups span namespaces and preserve exact names and a distinct fallback', t => {
	const resource = (name, namespace, group) => ({name, namespace, kind: 'HTTPRoute', labels: group === undefined ? {} : {'k8sautodocs.io/group': group}, endpoints: [{url: `https://${name}.example.com`}, {url: `http://${name}.example.com`}]});
	const unsafeGroup = '<img src=x onerror="window.injected=true">';
	return checkInBrowser(t, [resource('second', 'z', 'media'), resource('first', 'a', 'media'), resource('capital', 'a', 'Media'), resource('literal', 'a', 'Ungrouped'), resource('missing', 'a'), resource('empty', 'a', ''), resource('unsafe', 'a', unsafeGroup)], [], `
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
	return checkInBrowser(t, [], [], `
		const search = document.getElementById('search-input');
		search.value = 'media'; performSearch(search.value);
		await selectView('discovered');
		check(search.value === 'media', 'query retained');
		check(document.querySelectorAll('.endpoint-button').length === 1, 'query applied to discovery');
		check(document.querySelectorAll('section.namespace-section').length === 1, 'discovery uses selected label grouping');
		check(document.getElementById('total-resources').textContent === '2', 'active view resource count');
		check(document.getElementById('total-namespaces').textContent === '1', 'active view namespace count');
		check(document.getElementById('total-hosts').textContent === '0', 'active view URL count');
		check(document.getElementById('filtered-count').textContent === 'Showing 1 of 2 resources · 0 URLs', 'filtered subset count');
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
		const wentBack = new Promise(resolve => window.addEventListener('popstate', resolve, {once:true}));
		history.back(); await wentBack;
		check(activeView === 'published', 'browser back restores Applications');
		const wentForward = new Promise(resolve => window.addEventListener('popstate', resolve, {once:true}));
		history.forward(); await wentForward;
		check(activeView === 'discovered', 'browser forward restores Discovered');
	`, 200, {setup: viewSetup});
});

test('direct discovered URL restores selection and handles empty discovered results', t => {
	return checkInBrowser(t, [], [], `
		check(activeView === 'discovered' && document.getElementById('discovered-tab').getAttribute('aria-selected') === 'true', 'direct-link selection');
		check(document.querySelector('.empty-directory').textContent === 'No discovered resources found.', 'discovered empty message');
		check(document.getElementById('total-resources').textContent === '0', 'empty active counts');
	`, 200, {query:'?view=discovered'});
});

test('unknown page view falls back to Applications', t => {
	return checkInBrowser(t, [], [], `check(activeView === 'published', 'unknown page view defaults to Applications');`, 200, {query:'?view=unknown'});
});

test('late previous responses and failures cannot overwrite the selected view', t => {
	return checkInBrowser(t, [], [], `
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

const {resource, setup: fixtureSetup} = require('../../scripts/dashboard-fixtures.cjs');
const run = (browser, callback) => browser.evaluate('(' + callback.toString() + ')()');
async function fixture(t, options = {}) {
	if (!chrome) { t.skip('Set CHROME_BIN to run the browser fixture checks'); return; }
	return browserFixture(t, {setup: fixtureSetup(), ...options});
}

test('runtime grouping, custom group lifecycle and resets operate without fetches', async t => {
	const browser = await fixture(t);
	if (!browser) return;
	await run(browser, () => {
		const assert = (value, reason) => { if (!value) throw new Error(reason); };
		const change = (id, value) => { const field = document.getElementById(id); field.value = value; field.dispatchEvent(new Event('change')); };
		const requests = requestLog.length;
		change('group-by', 'namespace');
		assert(document.querySelectorAll('.namespace-section').length === 7, 'namespace sections');
		change('group-by', 'kind');
		assert(document.querySelectorAll('.namespace-section').length === 2, 'kind sections');
		change('group-by', 'none');
		assert(!document.querySelector('.namespace-section') && document.querySelectorAll('.endpoint-button').length === 9, 'flat mode');
		document.getElementById('edit-groups').click();
		assert(document.activeElement.id === 'new-group-name', 'editor focus');
		const create = name => { document.getElementById('new-group-name').value = name; document.getElementById('new-group-form').requestSubmit(); };
		create('  ');
		assert(document.getElementById('group-error').textContent, 'empty name validation');
		create('Essentials'); create('Essentials');
		assert(AutoDocs.preferences.directory.groups.length === 1 && document.getElementById('group-error').textContent, 'duplicate validation');
		create('<img src=x onerror="window.injected=true">');
		assert(!document.querySelector('#editor-groups img') && !window.injected, 'custom text is safe');
		const group = AutoDocs.preferences.directory.groups[0];
		const source = model.sourceID(allData[0]);
		const assignment = [...document.querySelectorAll('.assignment')].find(node => node.dataset.control === 'assign' + source);
		assignment.value = group.id; assignment.dispatchEvent(new Event('change'));
		toggleStar('groupStars', model.groupID('custom', group.id));
		const input = document.getElementById('group-name-' + group.id); input.value = 'Daily';
		[...document.querySelectorAll('.editor-row button')].find(node => node.textContent === 'Rename').click();
		assert(AutoDocs.preferences.directory.groups[0].name === 'Daily', 'renamed');
		assert(AutoDocs.preferences.directory.groupStars.includes(model.groupID('custom', group.id)), 'stable group star');
		[...document.querySelectorAll('.editor-row button')].find(node => node.textContent === 'Move down').click();
		assert(AutoDocs.preferences.directory.groups[1].id === group.id, 'keyboard reorder');
		const link = model.linkID(allData[0], allData[0].endpoints[0].url);
		toggleStar('linkStars', link); toggleStar('groupStars', model.groupID('label', 'Infrastructure'));
		deleteGroup(group.id);
		assert(!AutoDocs.preferences.directory.assignments[source], 'deletion unassigns');
		assert(AutoDocs.preferences.directory.linkStars.includes(link), 'deletion preserves link');
		create('Personal');
		document.getElementById('reset-layout').click();
		assert(AutoDocs.preferences.directory.groupBy === 'label' && !AutoDocs.preferences.directory.groups.length, 'layout reset');
		assert(AutoDocs.preferences.directory.groupStars.includes(model.groupID('label', 'Infrastructure')) && AutoDocs.preferences.directory.linkStars.includes(link), 'reset retains derived/link stars');
		create('Keep me');
		document.getElementById('clear-favourites').click();
		assert(AutoDocs.preferences.directory.groups.length === 1 && !AutoDocs.preferences.directory.linkStars.length && !AutoDocs.preferences.directory.groupStars.length, 'clear stars preserves groups');
		document.getElementById('close-editor').click();
		assert(!document.getElementById('group-editor').open, 'dialog closed');
		assert(requestLog.length === requests, 'personal actions never refetch or write');
	});
	await browser.wait('document.activeElement.id === "edit-groups"');
});

test('live group and independent link stars compose across regrouping and search', async t => {
	const data = [
		resource('alpha', 'apps', 'media', 'HTTPRoute', ['https://same.test', 'https://second.test', 'https://same.test']),
		resource('beta', 'other', 'media', 'ingress', ['https://same.test']),
		{...resource('unresolved', 'apps', 'media', 'HTTPRoute', []), issues: ['No hostname']},
		resource('single', 'apps', undefined, 'HTTPRoute', ['https://one.test', 'https://two.test'])
	];
	const browser = await fixture(t, {setup: fixtureSetup(data)});
	if (!browser) return;
	await run(browser, async () => {
		const assert = (value, reason) => { if (!value) throw new Error(reason); };
		const first = allData.find(item => item.name === 'alpha');
		const single = allData.find(item => item.name === 'single');
		const stars = () => AutoDocs.preferences.directory;
		toggleStar('linkStars', model.linkID(first, 'https://same.test'));
		favouritesOnly = true; syncControls(); performSearch('');
		assert(document.querySelectorAll('.endpoint-button').length === 1 && document.querySelectorAll('.endpoint-link').length === 1, 'same URL on other source is independent');
		toggleStar('groupStars', model.groupID('label', 'media'));
		toggleStar('linkStars', model.linkID(single, 'https://two.test'));
		modifyDirectory(prefs => { prefs.groupBy = 'namespace'; });
		assert(document.querySelectorAll('.endpoint-button').length === 4 && document.querySelectorAll('.endpoint-link').length === 4, 'union across different grouping and deduplicated URLs');
		assert(document.querySelector('.endpoint-issues').textContent.includes('No hostname'), 'unresolved member retained');
		toggleStar('linkStars', model.linkID(first, 'https://same.test'));
		assert(stars().groupStars.length === 1 && document.querySelectorAll('.endpoint-button').length === 4, 'unstar link leaves group inclusion');
		performSearch('APPS HTTPROUTE');
		assert(document.querySelectorAll('.endpoint-button').length === 3, 'terms span fields');
		document.getElementById('kind-filter').value = 'ingress';
		performSearch('media');
		assert(document.querySelectorAll('.endpoint-button').length === 1, 'kind combines with stars/query');
		document.getElementById('kind-filter').value = '';
		fixtureData.find(item => item.name === 'alpha').labels['k8sautodocs.io/group'] = 'moved';
		fixtureData.find(item => item.name === 'beta').labels['k8sautodocs.io/group'] = 'moved';
		performSearch(''); await loadDirectory();
		assert(document.querySelectorAll('.endpoint-button').length === 2, 'membership changes follow live group');
		fixtureData.find(item => item.name === 'single').endpoints = [];
		await loadDirectory();
		assert(document.querySelectorAll('.endpoint-button').length === 1 && stars().linkStars.length === 1, 'missing link dormant');
		fixtureData.find(item => item.name === 'single').endpoints = [{url:'https://two.test'}];
		await loadDirectory();
		assert(document.querySelectorAll('.endpoint-button').length === 2, 'exact returning link reattaches');
		const moving = fixtureData.splice(fixtureData.findIndex(item => item.name === 'single'), 1)[0];
		fixtureDiscovered = [moving];
		await loadDirectory();
		assert(document.querySelectorAll('.endpoint-button').length === 1, 'absent source not leaked from saved star');
		await selectView('discovered');
		assert(document.querySelectorAll('.endpoint-button').length === 1 && document.querySelector('.endpoint-link').getAttribute('href') === 'https://two.test', 'link star follows reclassification');
		assert(document.querySelectorAll('.namespace-section').length === 1, 'discovery retains namespace grouping');
	});
});

test('typed group identities and stable source URL identities do not collide', async t => {
	const data = [resource('same', 'a', 'Ungrouped'), resource('same', 'b', undefined), resource('same', 'a', 'media', 'ingress'), resource('other', 'a', 'Media')];
	const browser = await fixture(t, {setup: fixtureSetup(data)});
	if (!browser) return;
	await run(browser, async () => {
		const assert = (value, reason) => { if (!value) throw new Error(reason); };
		const label = model.groupID('label', 'Ungrouped'), fallback = model.groupID('label');
		toggleStar('groupStars', label);
		assert(!AutoDocs.preferences.directory.groupStars.includes(fallback), 'literal and fallback independent');
		const identities = allData.map(item => model.sourceID(item));
		assert(new Set(identities).size === allData.length, 'namespace and kind identity');
		const first = allData[0], identity = model.linkID(first, first.endpoints[0].url);
		toggleStar('linkStars', identity);
		fixtureData.reverse(); fixtureData.forEach(item => item.endpoints.reverse()); await loadDirectory();
		assert(AutoDocs.preferences.directory.linkStars.includes(identity), 'response order does not matter');
		modifyDirectory(prefs => { prefs.groupBy = 'namespace'; });
		toggleStar('groupStars', model.groupID('namespace', 'Ungrouped'));
		assert(new Set(AutoDocs.preferences.directory.groupStars).size === 2, 'group modes independent');
	});
});

test('keyboard shortcuts, filter retention and focus survive rerendering', async t => {
	const browser = await fixture(t);
	if (!browser) return;
	await run(browser, async () => {
		const assert = (value, reason) => { if (!value) throw new Error(reason); };
		document.body.dispatchEvent(new KeyboardEvent('keydown', {key:'k', ctrlKey:true, bubbles:true}));
		assert(document.activeElement.id === 'search-input', 'search shortcut');
		performSearch('media sonarr');
		const star = document.querySelector('.endpoint-row .star-button'); star.focus(); star.click();
		assert(document.activeElement.dataset.control === star.dataset.control, 'star focus retained');
		document.getElementById('namespace-filter').value = 'media';
		document.getElementById('search-input').dispatchEvent(new KeyboardEvent('keydown', {key:'Escape', bubbles:true}));
		assert(document.getElementById('namespace-filter').value === 'media' && !document.getElementById('search-input').value, 'Escape only clears query');
		await selectView('discovered');
		assert(document.getElementById('namespace-filter').value === 'media' && !document.querySelector('.endpoint-button'), 'nonmatching retained filter');
		assert(!document.getElementById('group-by').disabled, 'discovery can be regrouped');
		document.getElementById('edit-groups').click();
		document.getElementById('new-group-name').dispatchEvent(new KeyboardEvent('keydown', {key:'k', ctrlKey:true, bubbles:true}));
		assert(document.activeElement.id === 'new-group-name', 'editor shortcut not intercepted');
	});
});

test('preferences survive same-origin reload and navigation across all pages', async t => {
	const browser = await fixture(t, {setup: fixtureSetup(undefined, undefined, 'if (!sessionStorage.getItem("seeded")) { localStorage.setItem("theme","dark"); sessionStorage.setItem("seeded","1"); }')});
	if (!browser) return;
	await run(browser, () => {
		if (document.documentElement.dataset.theme !== 'dark') throw new Error('legacy theme import');
		modifyDirectory(prefs => {
			prefs.groupBy = 'custom'; prefs.density = 'list';
			prefs.groups.push({id:'stable-id',name:'Daily'});
			prefs.assignments[model.sourceID(allData[0])] = 'stable-id';
			prefs.groupStars.push(model.groupID('custom','stable-id'));
			prefs.linkStars.push(model.linkID(allData[0],allData[0].endpoints[0].url));
			prefs.collapsed.push(model.groupID('custom','stable-id'));
		});
	});
	await browser.reload();
	assert.deepEqual(await browser.evaluate('[AutoDocs.preferences.theme, AutoDocs.preferences.directory.groupBy, AutoDocs.preferences.directory.density, AutoDocs.preferences.directory.groups[0].name, AutoDocs.preferences.directory.linkStars.length, AutoDocs.preferences.directory.collapsed.length]'), ['dark','custom','list','Daily',1,1]);
	await browser.navigate('/versions-dashboard-page');
	assert.equal(await browser.evaluate('document.documentElement.dataset.theme'), 'dark');
	await browser.evaluate('document.getElementById("theme-toggle").click()');
	await browser.navigate('/');
	assert.equal(await browser.evaluate('document.documentElement.dataset.theme'), 'light');
	await browser.navigate('/autodocs-dashboard-page');
	assert.equal(await browser.evaluate('document.getElementById("group-by").value'), 'custom');
	assert.equal(await browser.evaluate('document.querySelectorAll(".endpoint-grid[data-density=list]").length > 0'), true);
});

for (const [name, seed] of [
	['malformed JSON', 'localStorage.setItem("k8sautodocs.preferences","{broken");'],
	['invalid shape', 'localStorage.setItem("k8sautodocs.preferences", JSON.stringify({version:1,theme:"dark",directory:{}}));'],
	['blocked reads', 'Object.defineProperty(window,"localStorage",{get(){throw new Error("blocked");}});'],
	['quota writes', 'Storage.prototype.setItem = function(){throw new DOMException("full","QuotaExceededError");};'],
	['future schema', 'localStorage.setItem("k8sautodocs.preferences", JSON.stringify({version:99,keep:"untouched"}));']
]) {
	test('browser settings tolerate ' + name, async t => {
		const browser = await fixture(t, {setup: fixtureSetup(undefined, undefined, seed)});
		if (!browser) return;
		await run(browser, () => {
			const before = document.documentElement.dataset.theme;
			document.getElementById('theme-toggle').click();
			if (before === document.documentElement.dataset.theme || !document.querySelector('.endpoint-button')) throw new Error('session controls remain usable');
		});
		assert.equal(await browser.evaluate('!document.getElementById("preference-notice").hidden'), name !== 'malformed JSON' && name !== 'invalid shape');
		if (name === 'future schema') assert.deepEqual(await browser.evaluate('JSON.parse(localStorage.getItem(AutoDocs.key))'), {version:99,keep:'untouched'});
	});
}

test('supported storage updates from a real second tab apply without refetching', async t => {
	const browser = await fixture(t);
	if (!browser) return;
	const count = await browser.evaluate('requestLog.length');
	const second = await browser.newTab('/empty');
	try {
		await second.send('Runtime.evaluate', {expression:'localStorage.setItem("k8sautodocs.preferences", ' + JSON.stringify(await browser.evaluate('JSON.stringify({...AutoDocs.preferences,theme:"dark",directory:{...AutoDocs.preferences.directory,groupBy:"kind"}})')) + ')'});
		await browser.wait('AutoDocs.preferences.directory.groupBy === "kind" && document.documentElement.dataset.theme === "dark"');
		assert.equal(await browser.evaluate('requestLog.length'), count);
	} finally { second.close(); }
});

test('a newer schema is not overwritten and a later supported tab update can restore persistence', async t => {
	const browser = await fixture(t, {setup: fixtureSetup(undefined, undefined, 'localStorage.setItem("k8sautodocs.preferences", JSON.stringify({version:99}));')});
	if (!browser) return;
	await browser.evaluate('document.getElementById("theme-toggle").click()');
	assert.equal(await browser.evaluate('JSON.parse(localStorage.getItem(AutoDocs.key)).version'), 99);
	await browser.evaluate('window.dispatchEvent(new StorageEvent("storage",{key:AutoDocs.key,newValue:JSON.stringify({version:1,theme:"dark",directory:AutoDocs.defaultDirectory()})}))');
	await browser.evaluate('document.getElementById("theme-toggle").click()');
	assert.equal(await browser.evaluate('JSON.parse(localStorage.getItem(AutoDocs.key)).version'), 1);
});

test('large fixture filtering and grouping remain local and responsive', async t => {
	const data = Array.from({length:1200}, (_, i) => resource('app-' + i, 'ns-' + (i % 12), 'group-' + (i % 20)));
	const browser = await fixture(t, {setup: fixtureSetup(data)});
	if (!browser) return;
	const result = await run(browser, () => {
		const requests = requestLog.length, start = performance.now();
		performSearch('app-1199 httproute');
		modifyDirectory(prefs => { prefs.groupBy = 'namespace'; });
		document.querySelector('.endpoint-row .star-button').click();
		if (document.querySelectorAll('.endpoint-button').length !== 1 || requestLog.length !== requests || requestLog.some(item => item.method !== 'GET')) throw new Error('local projection');
		return performance.now() - start;
	});
	assert.ok(result < 1500, 'search/regroup/star sequence under 1.5s: ' + result);
	t.diagnostic('1200-source search/regroup/star: ' + result.toFixed(1) + 'ms');
});

test('versions preserve resource/container search and render metadata safely', async t => {
	const browser = await fixture(t, {route:'/versions-dashboard-page', setup:fixtureSetup(undefined, undefined, 'fixtureVersions[0].name = "<img src=x onerror=window.injected=true>";')});
	if (!browser) return;
	await browser.wait('document.getElementById("table-body").children.length > 0');
	assert.equal(await browser.evaluate('document.querySelectorAll("#table-body tr").length'), 3);
	assert.equal(await browser.evaluate('Boolean(document.querySelector("#table-body img")) || Boolean(window.injected)'), false);
	await browser.evaluate('document.getElementById("search-input").value = "busybox"; document.getElementById("search-input").dispatchEvent(new Event("input"))');
	await browser.wait('document.getElementById("filtered-count").textContent === "Showing 1 of 2 resources"');
	assert.equal(await browser.evaluate('document.querySelectorAll("#table-body tr").length'), 2);
});

test('system theme follows media until an explicit theme is saved', async t => {
	const browser = await fixture(t);
	if (!browser) return;
	await browser.send('Emulation.setEmulatedMedia', {features:[{name:'prefers-color-scheme',value:'dark'}]});
	await browser.wait('document.documentElement.dataset.theme === "dark"');
	await browser.evaluate('document.getElementById("theme-toggle").click()');
	await browser.send('Emulation.setEmulatedMedia', {features:[{name:'prefers-color-scheme',value:'light'}]});
	assert.equal(await browser.evaluate('AutoDocs.preferences.theme'), 'light');
	await browser.send('Emulation.setEmulatedMedia', {features:[{name:'prefers-color-scheme',value:'dark'}]});
	assert.equal(await browser.evaluate('document.documentElement.dataset.theme'), 'light');
});

test('versions render a null API inventory as an honest empty state', async t => {
	const browser = await fixture(t, {route:'/versions-dashboard-page', setup:fixtureSetup(undefined, undefined, 'fixtureVersions = null;')});
	if (!browser) return;
	await browser.wait('document.getElementById("table-container").style.display === "block"');
	assert.equal(await browser.evaluate('document.getElementById("total-resources").textContent'), '0');
	assert.equal(await browser.evaluate('document.getElementById("total-containers").textContent'), '0');
	assert.equal(await browser.evaluate('document.getElementById("unique-images").textContent'), '0');
	assert.equal(await browser.evaluate('document.getElementById("table-body").textContent'), 'No container versions found.');
});

test('collapse, density and personal-group search restore through reload', async t => {
	const browser = await fixture(t);
	if (!browser) return;
	await run(browser, () => {
		modifyDirectory(prefs => { prefs.groups.push({id:'searchable',name:'Daily launchpad'}); prefs.assignments[model.sourceID(allData[0])] = 'searchable'; prefs.density = 'list'; });
		performSearch('daily httproute');
		if (document.querySelectorAll('.endpoint-button').length !== 1) throw new Error('custom name searchable in label mode');
		document.querySelector('.group-heading button[aria-expanded]').click();
		if (!document.querySelector('.endpoint-grid').hidden) throw new Error('collapsed');
	});
	await browser.reload();
	assert.equal(await browser.evaluate('document.querySelectorAll(".endpoint-grid[hidden]").length'), 1);
	assert.equal(await browser.evaluate('AutoDocs.preferences.directory.density'), 'list');
	assert.equal(await browser.evaluate('document.getElementById("search-input").value'), '');
});

for (const width of [375, 1440]) for (const theme of ['light','dark']) {
	for (const [route, name] of [['/','overview'], ['/autodocs-dashboard-page','directory'], ['/versions-dashboard-page','versions']]) {
		test(name + ' presents a responsive ' + width + 'px ' + theme + ' theme', async t => {
			const browser = await fixture(t, {route,width,setup:fixtureSetup(undefined,undefined,'localStorage.setItem("theme",' + JSON.stringify(theme) + ');')});
			if (!browser) return;
			await browser.send('Emulation.setEmulatedMedia', {features:[{name:'prefers-reduced-motion',value:'reduce'}]});
			const result = await run(browser, () => {
				const rgb = value => value.match(/[\d.]+/g).slice(0,3).map(Number);
				const luminance = color => rgb(color).map(x => { const v = x / 255; return v <= .04045 ? v / 12.92 : ((v + .055) / 1.055) ** 2.4; }).reduce((sum,v,i) => sum + v * [.2126,.7152,.0722][i],0);
				const contrast = (a,b) => (Math.max(luminance(a),luminance(b))+.05)/(Math.min(luminance(a),luminance(b))+.05);
				const canvas = getComputedStyle(document.body).backgroundColor;
				for (const selector of ['h1','.page-hero p','.stat-label','.endpoint-link','.endpoint-context','.badge','.group-mode','.page-footer','.landing-hero p']) {
					for (const node of document.querySelectorAll(selector)) {
						if (!node.getClientRects().length) continue;
						let parent = node, background;
						while (parent) {
							const color = getComputedStyle(parent).backgroundColor;
							if (color !== 'rgba(0, 0, 0, 0)') { background = color; break; }
							parent = parent.parentElement;
						}
						const ratio = contrast(getComputedStyle(node).color, background || canvas);
						if (ratio < 4.5) throw new Error(selector + ' contrast ' + ratio);
					}
				}
				if (document.documentElement.scrollWidth > innerWidth) throw new Error('Page overflows: ' + document.documentElement.scrollWidth + ' > ' + innerWidth);
				const table = document.querySelector('table');
				if (table && innerWidth < 760 && table.getBoundingClientRect().width <= document.getElementById('table-container').clientWidth) throw new Error('Versions table should scroll without crushing columns');
				if (document.querySelector('.endpoint-button') && getComputedStyle(document.querySelector('.endpoint-button')).transitionDuration !== '0s') throw new Error('reduced motion');
				const unlabeled = [...document.querySelectorAll('button,input,select')].filter(node => !node.getAttribute('aria-label') && !node.textContent.trim() && !node.labels?.length);
				if (unlabeled.length) throw new Error('unlabelled controls');
				return document.documentElement.dataset.theme;
			});
			assert.equal(result, theme);
			assert.ok(browser.requests.every(request => request.method === 'GET' && !request.path.startsWith('/api/')), 'fixture pages require only local assets');
			if (process.env.AUTODOCS_SCREENSHOTS) await browser.screenshot(require('node:path').join(process.env.AUTODOCS_SCREENSHOTS, name + '-' + width + '-' + theme + '.png'));
		});
	}
	test('group editor and empty directory fit ' + width + 'px ' + theme + ' theme', async t => {
		const browser = await fixture(t, {width,setup:fixtureSetup(undefined,undefined,'localStorage.setItem("theme",' + JSON.stringify(theme) + ');')});
		if (!browser) return;
		await browser.evaluate('document.getElementById("edit-groups").click(); document.getElementById("new-group-name").value = "Daily essentials"; document.getElementById("new-group-form").requestSubmit()');
		assert.equal(await browser.evaluate('document.querySelector("dialog").scrollWidth <= document.querySelector("dialog").clientWidth'), true);
		if (process.env.AUTODOCS_SCREENSHOTS) await browser.screenshot(require('node:path').join(process.env.AUTODOCS_SCREENSHOTS, 'editor-' + width + '-' + theme + '.png'));
		await browser.evaluate('document.getElementById("group-editor").close(); fixtureData = []; loadDirectory()');
		await browser.wait('directoryReady && allData.length === 0');
		assert.equal(await browser.evaluate('document.documentElement.scrollWidth <= innerWidth'), true);
		if (process.env.AUTODOCS_SCREENSHOTS) await browser.screenshot(require('node:path').join(process.env.AUTODOCS_SCREENSHOTS, 'empty-' + width + '-' + theme + '.png'));
	});
}

test('Discovered supports every grouping mode and persists personal assignments', async t => {
	const discovered = [
		resource('candidate', 'alpha', 'shared'),
		resource('another-candidate', 'beta', 'shared', 'ingress'),
		{...resource('unresolved', 'alpha', undefined, 'HTTPRoute', []), issues:['Missing parent']}
	];
	const browser = await fixture(t, {query:'?view=discovered', setup:fixtureSetup(undefined, discovered)});
	if (!browser) return;
	await run(browser, () => {
		const assert = (value, message) => { if (!value) throw new Error(message); };
		const change = mode => {
			document.getElementById('group-by').value = mode;
			document.getElementById('group-by').dispatchEvent(new Event('change'));
		};
		const sections = () => [...document.querySelectorAll('.namespace-header')].map(node => node.textContent);
		const requests = requestLog.length;
		assert(activeView === 'discovered' && !document.getElementById('group-by').disabled, 'enabled on a direct Discovered link');
		assert(sections().join('|') === 'shared|Ungrouped', 'source label grouping');
		assert(document.querySelector('.group-description').textContent.includes('Discovered endpoints'), 'fallback describes current view');
		change('namespace');
		assert(sections().join('|') === 'alpha|beta', 'namespace grouping');
		change('kind');
		assert(sections().join('|') === 'HTTPRoute|ingress', 'kind grouping');
		change('none');
		assert(!sections().length && document.querySelectorAll('.endpoint-button').length === 3, 'explicit flat grouping');
		document.getElementById('edit-groups').click();
		document.getElementById('new-group-name').value = 'Candidates';
		document.getElementById('new-group-form').requestSubmit();
		document.getElementById('group-editor').close();
		const group = directoryPrefs().groups[0];
		const source = allData.find(item => item.name === 'candidate');
		const assignment = [...document.querySelectorAll('.assignment')].find(node => node.dataset.control === 'assign' + model.sourceID(source));
		assert(assignment, 'Discovered provides personal group assignment');
		assignment.value = group.id; assignment.dispatchEvent(new Event('change'));
		assert(sections().join('|') === 'Candidates|Unassigned', 'custom grouping with fallback');
		assert(source.labels[model.groupLabel] === 'shared' && !Object.hasOwn(source.labels, 'k8sautodocs.io/publish'), 'personal assignment never changes labels or publishes');
		assert(document.querySelector('.endpoint-issues').textContent.includes('Missing parent'), 'unresolved context remains');
		assert(requestLog.length === requests, 'regrouping makes no requests');
	});
	await browser.reload();
	assert.equal(await browser.evaluate('activeView'), 'discovered');
	assert.equal(await browser.evaluate('document.getElementById("group-by").value'), 'custom');
	assert.deepEqual(await browser.evaluate('[...document.querySelectorAll(".namespace-header")].map(node => node.textContent)'), ['Candidates','Unassigned']);
	assert.equal(await browser.evaluate('document.querySelectorAll(".endpoint-button").length'), 3);
});

test('Discovered group favourites follow active-view membership across grouping and tab switches', async t => {
	const published = [resource('published-only', 'public', 'shared')];
	const discovered = [
		resource('candidate', 'alpha', 'shared'),
		{...resource('unresolved', 'beta', 'shared', 'ingress', []), issues:['Missing parent']}
	];
	const browser = await fixture(t, {query:'?view=discovered', setup:fixtureSetup(published, discovered)});
	if (!browser) return;
	await run(browser, async () => {
		const assert = (value, message) => { if (!value) throw new Error(message); };
		document.querySelector('.group-heading .star-button').click();
		document.getElementById('favourites-toggle').click();
		assert(document.querySelectorAll('.endpoint-button').length === 2, 'starred Discovered group includes current members');
		assert(document.querySelector('.endpoint-issues').textContent.includes('Missing parent'), 'unresolved favourite remains visible');
		performSearch('alpha httproute');
		assert(document.querySelectorAll('.endpoint-button').length === 1, 'search composes with Discovered group favourites');
		performSearch('');
		modifyDirectory(prefs => { prefs.groupBy = 'none'; });
		assert(document.querySelectorAll('.endpoint-button').length === 2 && !document.querySelector('.namespace-section'), 'group stars still work in no-grouping mode');
		await selectView('published');
		assert(document.querySelectorAll('.endpoint-button').length === 1 && document.querySelector('.endpoint-name').textContent === 'published-only', 'Applications only renders its own eligible membership');
		await selectView('discovered');
		assert(document.querySelectorAll('.endpoint-button').length === 2 && !document.getElementById('content-container').textContent.includes('published-only'), 'Discovered only renders its own eligible membership');
	});
});

const sortableVersions = [
	{name:'app-10',namespace:'beta',kind:'Deployment',containers:[{name:'main',type:'main',image:'registry.example/team/app',tag:'v10',sha:'b'.repeat(64)}]},
	{name:'app-2',namespace:'alpha',kind:'StatefulSet',containers:[{name:'z-main',type:'main',image:'registry.example/team/app',tag:'v2',sha:'a'.repeat(64)},{name:'a-init',type:'init',image:'setup',tag:'v1',sha:''}]},
	{name:'app-1',namespace:'beta',kind:'DaemonSet',containers:[{name:'main',type:'main',image:'agent',tag:'v1',sha:'c'.repeat(64)}]},
	{name:'app-10',namespace:'alpha',kind:'Deployment',containers:[{name:'main',type:'main',image:'registry.example/team/app',tag:'v10',sha:'a'.repeat(64)}]},
];
async function versionsBrowser(t, options = {}) {
	const {extra = '', data = sortableVersions, ...rest} = options;
	const browser = await fixture(t, {route:'/versions-dashboard-page', setup:fixtureSetup(undefined,undefined,'fixtureVersions = ' + JSON.stringify(data) + ';' + extra), ...rest});
	if (browser) await browser.wait('document.getElementById("table-container").style.display === "block" && document.querySelector("col").style.width !== ""');
	return browser;
}
const tableRows = '[...document.querySelectorAll("#table-body tr")].map(row => [...row.cells].map(cell => cell.textContent))';
const tableWidths = '[...document.querySelectorAll("#versions-columns col")].map(col => parseFloat(col.style.width))';
async function mouse(browser, type, x, y, more = {}) {
	await browser.send('Input.dispatchMouseEvent', {type,x,y,...more});
}

test('versions sort namespace then resource, cycle directions, promote keys and clear', async t => {
	const browser = await versionsBrowser(t);
	if (!browser) return;
	const original = await browser.evaluate(tableRows);
	await browser.evaluate('document.querySelector("[data-sort=namespace]").click(); document.querySelector("[data-sort=resource]").click()');
	let rows = await browser.evaluate(tableRows);
	assert.deepEqual(rows.map(row => [row[1],row[0]]), [['alpha','app-2'],['alpha','app-2'],['alpha','app-10'],['beta','app-1'],['beta','app-10']]);
	assert.deepEqual(await browser.evaluate('AutoDocs.preferences.versions.sort'), [{key:'namespace',direction:'asc'},{key:'resource',direction:'asc'}]);
	assert.equal(await browser.evaluate('document.querySelector("th[aria-sort]").dataset.column'), 'namespace');
	assert.deepEqual(await browser.evaluate('[...document.querySelectorAll(".column-sort[data-active=true] .sort-indicator")].map(node=>node.textContent)'), ['↑2','↑1']);
	assert.ok((await browser.evaluate('document.getElementById("sort-summary").textContent')).includes('1. Namespace ↑ ascending, then 2. Resource ↑ ascending'));
	await browser.evaluate('document.querySelector("[data-sort=resource]").click()');
	rows = await browser.evaluate(tableRows);
	assert.deepEqual(rows.map(row => row[0]), ['app-10','app-2','app-2','app-10','app-1']);
	await browser.evaluate('document.querySelector("[data-sort=kind]").click()');
	assert.deepEqual(await browser.evaluate('AutoDocs.preferences.versions.sort.map(s=>s.key)'), ['namespace','kind']);
	await browser.evaluate('document.querySelector("[data-sort=namespace]").click(); document.querySelector("[data-sort=namespace]").click()');
	assert.deepEqual(await browser.evaluate('AutoDocs.preferences.versions.sort'), [{key:'kind',direction:'asc'}]);
	assert.equal(await browser.evaluate('document.querySelector("th[aria-sort]").dataset.column'), 'kind');
	await browser.evaluate('document.getElementById("clear-sort").click()');
	assert.deepEqual(await browser.evaluate(tableRows), original);
	assert.equal(await browser.evaluate('document.getElementById("clear-sort").disabled'), true);
});

test('versions sort all container fields naturally with stable ties, missing last and resource search', async t => {
	const browser = await versionsBrowser(t);
	if (!browser) return;
	for (const [key,index] of [['container',3],['type',4],['image',5],['tag',6],['sha',7]]) {
		await browser.evaluate('document.getElementById("clear-sort").click(); document.querySelector("[data-sort=' + key + ']").click()');
		const rows = await browser.evaluate(tableRows);
		const present = rows.map(row=>row[index]).filter(value=>value !== '—');
		assert.deepEqual(present, [...present].sort(new Intl.Collator(undefined,{numeric:true,sensitivity:'base'}).compare), key);
	}
	let rows = await browser.evaluate(tableRows);
	assert.deepEqual(rows.filter(row=>row[7]==='a'.repeat(64)).map(row=>row[0]), ['app-2','app-10']);
	assert.equal(rows.at(-1)[7], '—');
	await browser.evaluate('document.querySelector("[data-sort=sha]").click()');
	assert.equal((await browser.evaluate(tableRows)).at(-1)[7], '—');
	await browser.evaluate('document.getElementById("clear-sort").click(); document.querySelector("[data-sort=container]").click(); document.getElementById("search-input").value="setup"; document.getElementById("search-input").dispatchEvent(new Event("input"))');
	await browser.wait('document.getElementById("filtered-count").textContent === "Showing 1 of 4 resources"');
	assert.deepEqual((await browser.evaluate(tableRows)).map(row=>row[3]), ['a-init','z-main']);
	assert.equal(await browser.evaluate('document.getElementById("total-containers").textContent'), '5');
	assert.ok((await browser.evaluate('requestLog')).every(request=>request.method === 'GET'));
});

test('versions default widths keep badges intact, preserve full text and stay stable through search', async t => {
	const browser = await versionsBrowser(t, {data:sortableVersions.map(row=>({...row,name:row.name+'-a-long-workload-name',containers:row.containers.map(c=>({...c,image:'registry.example/long-team-path/'.repeat(5)}))}))});
	if (!browser) return;
	const widths = await browser.evaluate(tableWidths);
	const badges = await browser.evaluate('[...document.querySelectorAll(".resource-kind,.container-type")].map(node=>({nowrap:getComputedStyle(node).whiteSpace, fits:node.scrollWidth<=node.clientWidth, text:node.textContent, title:node.title}))');
	assert.ok(badges.every(node=>node.nowrap==='nowrap' && node.fits && node.text===node.title));
	const image = await browser.evaluate('({text:document.querySelector("td .image-name").textContent,title:document.querySelector("td .image-name").title,clipped:document.querySelector("td .image-name").scrollWidth>document.querySelector("td .image-name").clientWidth})');
	assert.equal(image.text,image.title); assert.equal(image.clipped,true);
	await browser.evaluate('document.getElementById("search-input").value="not-found"; document.getElementById("search-input").dispatchEvent(new Event("input"))');
	await browser.wait('document.getElementById("filtered-count").textContent === "Showing 0 of 4 resources"');
	assert.deepEqual(await browser.evaluate(tableWidths), widths);
});

test('versions pointer resizing, cancellation and double-click fit never sort', async t => {
	const browser = await versionsBrowser(t);
	if (!browser) return;
	await browser.evaluate('document.querySelector("[data-sort=namespace]").click()');
	const sort = await browser.evaluate('AutoDocs.preferences.versions.sort');
	const originalRows = await browser.evaluate(tableRows);
	const before = await browser.evaluate(tableWidths);
	const point = await browser.evaluate('(()=>{const r=document.querySelector("th[data-column=resource] .column-resize").getBoundingClientRect();return {x:r.x+r.width/2,y:r.y+r.height/2}})()');
	await mouse(browser,'mousePressed',point.x,point.y,{button:'left',clickCount:1});
	await mouse(browser,'mouseMoved',point.x+75,point.y,{button:'left',buttons:1});
	await mouse(browser,'mouseReleased',point.x+75,point.y,{button:'left',clickCount:1});
	const after = await browser.evaluate(tableWidths);
	assert.equal(after[0],before[0]+75); assert.deepEqual(after.slice(1),before.slice(1));
	assert.equal(await browser.evaluate('AutoDocs.preferences.versions.widths.resource'),after[0]);
	const saved = await browser.evaluate('JSON.stringify(AutoDocs.preferences.versions)');
	const cancel = await browser.evaluate('(()=>{const r=document.querySelector("th[data-column=resource] .column-resize").getBoundingClientRect();return {x:r.x+r.width/2,y:r.y+r.height/2}})()');
	await mouse(browser,'mousePressed',cancel.x,cancel.y,{button:'left',clickCount:1});
	await mouse(browser,'mouseMoved',cancel.x+50,cancel.y,{button:'left',buttons:1});
	await browser.send('Input.dispatchKeyEvent',{type:'keyDown',key:'Escape',code:'Escape',windowsVirtualKeyCode:27});
	await mouse(browser,'mouseReleased',cancel.x+50,cancel.y,{button:'left',clickCount:1});
	assert.deepEqual(await browser.evaluate(tableWidths),after);
	assert.equal(await browser.evaluate('JSON.stringify(AutoDocs.preferences.versions)'),saved);
	await browser.evaluate('document.getElementById("table-container").scrollLeft=9999');
	const sha = await browser.evaluate('(()=>{const r=document.querySelector("th[data-column=sha] .column-resize").getBoundingClientRect();return {x:r.x+r.width/2-3,y:r.y+r.height/2}})()');
	await mouse(browser,'mousePressed',sha.x,sha.y,{button:'left',clickCount:2});
	await mouse(browser,'mouseReleased',sha.x,sha.y,{button:'left',clickCount:2});
	assert.ok((await browser.evaluate(tableWidths))[7]>after[7], 'double-click expands digest to fit');
	assert.deepEqual(await browser.evaluate('AutoDocs.preferences.versions.sort'),sort);
	assert.deepEqual(await browser.evaluate(tableRows),originalRows);
});

test('versions native keyboard sorting, separator resizing, clamping and fit reset preserve focus and other preferences', async t => {
	const browser = await versionsBrowser(t);
	if (!browser) return;
	await browser.evaluate('document.querySelector("[data-sort=namespace]").focus()');
	await browser.send('Input.dispatchKeyEvent',{type:'keyDown',key:'Enter',code:'Enter',windowsVirtualKeyCode:13,text:'\r',unmodifiedText:'\r'});
	await browser.send('Input.dispatchKeyEvent',{type:'keyUp',key:'Enter',code:'Enter',windowsVirtualKeyCode:13});
	assert.deepEqual(await browser.evaluate('AutoDocs.preferences.versions.sort'),[{key:'namespace',direction:'asc'}]);
	const before = await browser.evaluate(tableWidths);
	await browser.evaluate('document.querySelector("th[data-column=resource] .column-resize").focus()');
	await browser.send('Input.dispatchKeyEvent',{type:'keyDown',key:'ArrowRight',code:'ArrowRight',windowsVirtualKeyCode:39});
	await browser.send('Input.dispatchKeyEvent',{type:'keyUp',key:'ArrowRight',code:'ArrowRight',windowsVirtualKeyCode:39});
	assert.equal((await browser.evaluate(tableWidths))[0],before[0]+10);
	assert.equal(await browser.evaluate('document.activeElement.getAttribute("aria-label")'),'Resize Resource column');
	await browser.evaluate('for(let i=0;i<30;i++)document.activeElement.dispatchEvent(new KeyboardEvent("keydown",{key:"ArrowLeft",shiftKey:true,bubbles:true}))');
	assert.equal(await browser.evaluate('Number(document.activeElement.getAttribute("aria-valuenow"))'),await browser.evaluate('Number(document.activeElement.getAttribute("aria-valuemin"))'));
	await browser.evaluate('for(let i=0;i<30;i++)document.activeElement.dispatchEvent(new KeyboardEvent("keydown",{key:"ArrowRight",shiftKey:true,bubbles:true}))');
	assert.equal(await browser.evaluate('Number(document.activeElement.getAttribute("aria-valuenow"))'),1200);
	await browser.evaluate('document.activeElement.dispatchEvent(new KeyboardEvent("keydown",{key:"Home",bubbles:true}))');
	assert.ok((await browser.evaluate(tableWidths))[0]<1200);
	const other = await browser.evaluate('JSON.stringify({theme:AutoDocs.preferences.theme,directory:AutoDocs.preferences.directory,sort:AutoDocs.preferences.versions.sort})');
	await browser.evaluate('document.getElementById("fit-columns").click()');
	assert.deepEqual(await browser.evaluate('AutoDocs.preferences.versions.widths'),{});
	assert.equal(await browser.evaluate('JSON.stringify({theme:AutoDocs.preferences.theme,directory:AutoDocs.preferences.directory,sort:AutoDocs.preferences.versions.sort})'),other);
});

test('versions width and two-key sort persist through reload, directory/theme edits and navigation', async t => {
	const browser = await versionsBrowser(t);
	if (!browser) return;
	await browser.evaluate('document.querySelector("[data-sort=namespace]").click(); document.querySelector("[data-sort=resource]").click(); document.querySelector("th[data-column=image] .column-resize").dispatchEvent(new KeyboardEvent("keydown",{key:"ArrowRight",bubbles:true}))');
	const saved = await browser.evaluate('AutoDocs.preferences.versions');
	await browser.reload();
	await browser.wait('document.querySelector("col").style.width !== ""');
	assert.deepEqual(await browser.evaluate('AutoDocs.preferences.versions'),saved);
	await browser.navigate('/autodocs-dashboard-page');
	await browser.evaluate('AutoDocs.update(prefs=>{prefs.theme="dark";prefs.directory.groupBy="namespace"})');
	await browser.navigate('/versions-dashboard-page');
	await browser.wait('document.querySelector("col").style.width !== ""');
	assert.deepEqual(await browser.evaluate('AutoDocs.preferences.versions'),saved);
	assert.deepEqual(await browser.evaluate('[AutoDocs.preferences.theme,AutoDocs.preferences.directory.groupBy]'),['dark','namespace']);
	assert.equal((await browser.evaluate(tableWidths))[5],saved.widths.image);
});

for (const [name, value] of [
	['old document', undefined],
	['invalid widths', {widths:{resource:-8},sort:[]}],
	['duplicate sort', {widths:{},sort:[{key:'namespace',direction:'asc'},{key:'namespace',direction:'desc'}]}],
	['unknown column', {widths:{},sort:[{key:'unknown',direction:'asc'}]}],
	['third level', {widths:{},sort:['namespace','resource','kind'].map(key=>({key,direction:'asc'}))}],
]) test('versions preference validation preserves existing settings: '+name, async t => {
	const seed={version:1,theme:'dark',directory:{groupBy:'namespace',density:'list',collapsed:[],groups:[],assignments:{},groupStars:[],linkStars:[]},versions:value};
	const browser = await versionsBrowser(t,{extra:'localStorage.setItem("k8sautodocs.preferences",'+JSON.stringify(JSON.stringify(seed))+');'});
	if (!browser) return;
	assert.deepEqual(await browser.evaluate('AutoDocs.preferences.versions'),{widths:{},sort:[]});
	assert.deepEqual(await browser.evaluate('[AutoDocs.preferences.theme,AutoDocs.preferences.directory.groupBy]'),['dark','namespace']);
	await browser.evaluate('document.querySelector("[data-sort=namespace]").click()');
	assert.equal(await browser.evaluate('AutoDocs.preferences.versions.sort[0].key'),'namespace');
});

for (const [name, seed] of [
	['blocked reads','Object.defineProperty(window,"localStorage",{get(){throw new Error("blocked")}});'],
	['quota writes','Storage.prototype.setItem=function(){throw new DOMException("full","QuotaExceededError")};'],
	['future schema','localStorage.setItem("k8sautodocs.preferences",JSON.stringify({version:99,keep:"untouched"}));'],
]) test('versions controls remain usable with '+name, async t => {
	const browser = await versionsBrowser(t,{extra:seed});
	if (!browser) return;
	await browser.evaluate('document.querySelector("[data-sort=namespace]").click(); document.querySelector("th[data-column=resource] .column-resize").dispatchEvent(new KeyboardEvent("keydown",{key:"ArrowRight",bubbles:true}))');
	assert.equal(await browser.evaluate('AutoDocs.preferences.versions.sort[0].key'),'namespace');
	assert.ok(await browser.evaluate('AutoDocs.preferences.versions.widths.resource')>0);
	assert.equal(await browser.evaluate('document.getElementById("preference-notice").hidden'),false);
	if(name==='future schema')assert.deepEqual(await browser.evaluate('JSON.parse(localStorage.getItem(AutoDocs.key))'),{version:99,keep:'untouched'});
});

test('versions live cross-tab preferences update sorting and widths without refetching', async t => {
	const browser = await versionsBrowser(t);
	if (!browser) return;
	const requests=await browser.evaluate('requestLog.length');
	const next=await browser.evaluate('({...AutoDocs.preferences,versions:{widths:{resource:350},sort:[{key:"namespace",direction:"asc"},{key:"resource",direction:"asc"}]}})');
	const second=await browser.newTab('/empty');
	try {
		await second.send('Runtime.evaluate',{expression:'localStorage.setItem("k8sautodocs.preferences",'+JSON.stringify(JSON.stringify(next))+')'});
		await browser.wait('AutoDocs.preferences.versions.widths.resource === 350');
		assert.equal((await browser.evaluate(tableWidths))[0],350);
		assert.deepEqual((await browser.evaluate(tableRows)).map(row=>row[1]),['alpha','alpha','alpha','beta','beta']);
		assert.equal(await browser.evaluate('requestLog.length'),requests);
	} finally { second.close(); }
});
