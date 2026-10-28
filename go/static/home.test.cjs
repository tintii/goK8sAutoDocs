const {test} = require('node:test');
const assert = require('node:assert/strict');
const {browserFixture, chrome} = require('../../scripts/browser-fixture.cjs');

const config = {
	title: 'Cluster overview', description: 'Application links, workload status and container versions.',
	environment: 'Homelab', motd: 'Maintenance window: Saturday 10:00–11:00.\nBackups run as usual.',
	links: [{label: 'Argo CD', url: 'https://argocd.example.com'}], footer: 'AutoDocs · Homelab'
};
const status = {
	ready: 2, attention: 1, inactive: 1, checkedAt: '2026-10-28T12:00:00Z',
	workloads: [
		{name:'database',namespace:'data',kind:'StatefulSet',ready:1,desired:2,state:'attention'},
		{name:'argocd',namespace:'argocd',kind:'Deployment',ready:1,desired:1,state:'ready'},
		{name:'agent',namespace:'system',kind:'DaemonSet',ready:1,desired:1,state:'ready'},
		{name:'sonarr',namespace:'media',kind:'Deployment',ready:0,desired:0,state:'inactive'}
	]
};
function setup(extra = '') {
	return `window.homeConfig = ${JSON.stringify(config)}; window.serviceStatus = ${JSON.stringify(status)};
	window.failConfig = false; window.failStatus = false;
	window.homeRequests = [];
	window.fetch = async (url, options = {}) => {
		homeRequests.push({url, method:options.method || 'GET'});
		const failed = url.endsWith('/home') ? failConfig : url.endsWith('/service-status') ? failStatus : false;
		return {ok:!failed, status:failed ? 503 : 200, json:async () => url.includes('cluster-info') ? {clusterName:'homelab / k3s'} : url.endsWith('/home') ? homeConfig : serviceStatus};
	};` + extra;
}
async function fixture(t, options = {}) {
	if (!chrome) { t.skip('Set CHROME_BIN to run the browser fixture checks'); return; }
	const browser = await browserFixture(t, {route:'/', setup:setup(), ...options});
	await browser.wait('!document.getElementById("home-refresh").disabled');
	return browser;
}
async function refresh(browser) {
	await browser.evaluate('document.getElementById("home-refresh").click()');
	await browser.wait('!document.getElementById("home-refresh").disabled');
}

test('home displays configured text, real availability and inactive workloads', async t => {
	const browser = await fixture(t);
	if (!browser) return;
	const result = await browser.evaluate(`({
		title:document.querySelector('h1').textContent,
		motd:document.getElementById('home-motd').textContent,
		summary:document.getElementById('status-summary').textContent,
		counts:['ready','attention','inactive'].map(key => document.getElementById('status-'+key).textContent),
		states:[...document.querySelectorAll('.status-pill')].map(node => node.textContent),
		link:document.querySelector('#home-links a').getAttribute('href'),
		requests:homeRequests
	})`);
	assert.equal(result.title, config.title);
	assert.equal(result.motd, config.motd);
	assert.equal(result.summary, '1 workload needs attention.');
	assert.deepEqual(result.counts, ['2','1','1']);
	assert.deepEqual(result.states, ['Needs attention','Ready','Ready','Inactive']);
	assert.equal(result.link, config.links[0].url);
	assert.ok(result.requests.every(r => r.method === 'GET' && r.url.startsWith('/api/v1/')));
	await browser.evaluate('serviceStatus = {...serviceStatus, ready:0, attention:0, inactive:0, workloads:[]}');
	await refresh(browser);
	assert.equal(await browser.evaluate('document.getElementById("status-summary").textContent'), 'No workloads found.');
	assert.equal(await browser.evaluate('document.querySelectorAll(".workload-status").length'), 0);
});

test('home reloads content safely and hides empty optional panels', async t => {
	const browser = await fixture(t);
	if (!browser) return;
	await browser.evaluate('Object.assign(homeConfig, ' + JSON.stringify({
		title: '<img src=x onerror="window.injected=true">', motd: '<script>window.injected=true</script>\nUpdate complete.',
		links: [{label:'<b>Runbook</b>',url:'/runbook'},{label:'Bad',url:'javascript:alert(1)'},{label:'Bad slash',url:'/\\evil.example'}]
	}) + ')');
	await refresh(browser);
	assert.equal(await browser.evaluate('document.getElementById("home-title").textContent'), '<img src=x onerror="window.injected=true">');
	assert.equal(await browser.evaluate('Boolean(document.querySelector("main img, #home-motd script, #home-links b")) || Boolean(window.injected)'), false);
	assert.equal(await browser.evaluate('document.querySelectorAll("#home-links a").length'), 1);
	assert.equal(await browser.evaluate('document.querySelector("#home-links a").rel'), 'noopener noreferrer');
	await browser.evaluate('homeConfig.motd = ""; homeConfig.links = [];');
	await refresh(browser);
	assert.equal(await browser.evaluate('document.getElementById("motd-panel").hidden && document.getElementById("home-links-panel").hidden'), true);
});

test('status failures remove previous healthy counts and recover independently from config', async t => {
	const browser = await fixture(t);
	if (!browser) return;
	await browser.evaluate('failStatus = true; failConfig = true;');
	await refresh(browser);
	assert.equal(await browser.evaluate('document.getElementById("status-counts").hidden'), true);
	assert.equal(await browser.evaluate('document.querySelectorAll(".status-pill").length'), 0);
	assert.match(await browser.evaluate('document.getElementById("status-summary").textContent'), /unavailable/);
	assert.equal(await browser.evaluate('document.getElementById("home-config-notice").hidden'), false);
	assert.equal(await browser.evaluate('document.getElementById("home-motd").textContent'), config.motd);
	await browser.evaluate('failStatus = false; serviceStatus.attention = 0; serviceStatus.workloads = serviceStatus.workloads.filter(item => item.state !== "attention");');
	await refresh(browser);
	assert.equal(await browser.evaluate('document.getElementById("status-summary").textContent'), 'All active workloads are ready.');
	assert.equal(await browser.evaluate('document.getElementById("status-counts").hidden'), false);
	assert.equal(await browser.evaluate('document.getElementById("home-config-notice").hidden'), false);
	await browser.evaluate('failConfig = false; homeConfig.motd = "Service restored.";');
	await refresh(browser);
	assert.equal(await browser.evaluate('document.getElementById("home-config-notice").hidden'), true);
	assert.equal(await browser.evaluate('document.getElementById("home-motd").textContent'), 'Service restored.');
});

test('initial API errors retain usable default navigation and unknown status', async t => {
	const browser = await fixture(t, {setup:setup('failConfig = true; failStatus = true;')});
	if (!browser) return;
	assert.equal(await browser.evaluate('document.querySelector("h1").textContent'), 'Cluster overview');
	assert.equal(await browser.evaluate('document.getElementById("status-counts").hidden'), true);
	assert.equal(await browser.evaluate('document.querySelectorAll(".destination").length'), 2);
	assert.equal(await browser.evaluate('document.getElementById("motd-panel").hidden'), true);
});

for (const width of [375,1440]) for (const theme of ['light','dark']) {
	test(`configured home fits ${width}px in ${theme} theme`, async t => {
		const browser = await fixture(t, {width, setup:setup(`localStorage.setItem('theme', ${JSON.stringify(theme)}); homeConfig.motd += '\\n' + 'LongNotice'.repeat(30); serviceStatus.workloads[0].name = 'long-service-name-'.repeat(12);`)});
		if (!browser) return;
		assert.equal(await browser.evaluate('document.documentElement.scrollWidth <= innerWidth'), true);
		assert.equal(await browser.evaluate('document.documentElement.dataset.theme'), theme);
		assert.equal(await browser.evaluate('getComputedStyle(document.getElementById("home-motd")).whiteSpace'), 'pre-wrap');
		if (process.env.HOME_PREVIEW_DIR) await browser.screenshot(process.env.HOME_PREVIEW_DIR + `/home-${width}-${theme}.png`);
	});
}
