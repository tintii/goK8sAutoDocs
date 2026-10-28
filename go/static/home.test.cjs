const {test} = require('node:test');
const assert = require('node:assert/strict');
const {browserFixture, chrome} = require('../../scripts/browser-fixture.cjs');
const {published, discovered} = require('../../scripts/dashboard-fixtures.cjs');
const config = {title:'Cluster overview',description:'Application links and container versions.',environment:'Homelab',motd:'Maintenance window: Saturday 10:00–11:00.\nBackups run as usual.',footer:'AutoDocs · Homelab'};
function setup(extra = '') {
	return `window.homeConfig = ${JSON.stringify(config)}; window.published = ${JSON.stringify(published)}; window.discovered = ${JSON.stringify(discovered)};
	window.failConfig = false; window.failHealth = false; window.failPublished = false; window.failDiscovered = false;
	window.homeRequests = []; window.healthChecks = 0;
	window.fetch = async (url, options = {}) => {
		homeRequests.push({url,method:options.method || 'GET'});
		if (url.endsWith('/health')) healthChecks++;
		const failed = url.endsWith('/home') ? failConfig : url.endsWith('/health') ? failHealth : url.includes('view=discovered') ? failDiscovered : failPublished;
		return {ok:!failed,status:failed ? 503 : 200,headers:{get:()=> '[]'},json:async()=>url.endsWith('/home') ? homeConfig : url.endsWith('/health') ? {status:'running'} : url.includes('view=discovered') ? discovered : published};
	};` + extra;
}
async function fixture(t, options = {}) {
	if (!chrome) { t.skip('Set CHROME_BIN to run the browser fixture checks'); return; }
	const browser = await browserFixture(t, {route:'/',setup:setup(),...options});
	await browser.wait('document.getElementById("service-indicator").dataset.state !== "checking"');
	return browser;
}
async function refresh(browser) {
	const count = await browser.evaluate('healthChecks');
	await browser.evaluate('document.dispatchEvent(new Event("visibilitychange"))');
	await browser.wait('healthChecks > '+count);
}
const starGroup = `AutoDocs.update(prefs => {prefs.directory.groupStars = [DirectoryModel.groupID('label','Infrastructure')];})`;

test('home folds MOTD and restored art into hero, with compact application status and no refresh', async t => {
	const b=await fixture(t); if (!b) return;
	assert.equal(await b.evaluate('document.querySelector("h1").textContent'),config.title);
	assert.equal(await b.evaluate('document.getElementById("home-motd").textContent'),config.motd);
	assert.equal(await b.evaluate('Boolean(document.querySelector(".landing-hero #motd-panel")) && Boolean(document.querySelector(".orbit-art svg"))'),true);
	assert.equal(await b.evaluate('document.getElementById("service-label").textContent'),'Running');
	assert.equal(await b.evaluate('document.getElementById("service-indicator").parentElement === document.getElementById("theme-toggle").parentElement'),true);
	assert.equal(await b.evaluate('Boolean(document.getElementById("home-refresh") || document.getElementById("status-counts") || document.getElementById("cluster-name"))'),false);
	assert.equal(await b.evaluate('homeRequests.some(r=>r.url.includes("service-status") || r.url.includes("cluster-info") || r.url.includes("auto-docs-dashboard"))'),false);
});

test('header reports application failures and recovery independently of home configuration', async t => {
	const b=await fixture(t); if (!b) return;
	await b.evaluate('failHealth = true; failConfig = true;');await refresh(b);
	assert.equal(await b.evaluate('document.getElementById("service-label").textContent'),'Unavailable');
	assert.equal(await b.evaluate('document.getElementById("home-config-notice").hidden'),false);
	assert.equal(await b.evaluate('document.getElementById("home-motd").textContent'),config.motd);
	await b.evaluate('failHealth=false;');await refresh(b);
	assert.equal(await b.evaluate('document.getElementById("service-label").textContent'),'Running');
	assert.equal(await b.evaluate('document.getElementById("home-config-notice").hidden'),false);
	await b.evaluate('failConfig=false;homeConfig.motd="Update complete.";');await refresh(b);
	assert.equal(await b.evaluate('document.getElementById("home-motd").textContent'),'Update complete.');
});

test('MOTD remains plain text, updates automatically and hides when empty', async t => {
	const b=await fixture(t); if (!b) return;
	await b.evaluate('homeConfig.title='+JSON.stringify('<img src=x onerror="window.injected=true">')+';homeConfig.motd='+JSON.stringify('<script>window.injected=true</script>\nUpdate complete.')+';');
	await refresh(b);
	assert.equal(await b.evaluate('document.getElementById("home-title").textContent'),'<img src=x onerror="window.injected=true">');
	assert.equal(await b.evaluate('Boolean(document.querySelector("main img, #home-motd script")) || Boolean(window.injected)'),false);
	await b.evaluate('homeConfig.motd="";');await refresh(b);
	assert.equal(await b.evaluate('document.getElementById("motd-panel").hidden'),true);
});

test('home favourites include starred groups and exact links from both directory views', async t => {
	const b=await fixture(t); if (!b) return;
	await b.evaluate(starGroup);
	await b.wait('document.querySelectorAll(".favourite-card").length === 4');
	assert.deepEqual(await b.evaluate('[...document.querySelectorAll(".favourite-name")].map(el=>el.textContent)'),['argocd','rancher','unpublished-service','grafana']);
	await b.evaluate(`AutoDocs.update(prefs => {prefs.directory.groupStars=[];prefs.directory.linkStars=[DirectoryModel.linkID(published[3], published[3].endpoints[0].url)];});`);
	assert.deepEqual(await b.evaluate('[...document.querySelectorAll(".favourite-name")].map(el=>el.textContent)'),['sonarr']);
	assert.equal(await b.evaluate('document.querySelector(".favourite-link").rel'),'noopener noreferrer');
	await b.reload();
	await b.wait('document.querySelectorAll(".favourite-card").length===1');
	assert.equal(await b.evaluate('document.querySelector(".favourite-name").textContent'),'sonarr');
	await b.evaluate('AutoDocs.update(prefs=>{prefs.directory.linkStars=[];});');
	assert.equal(await b.evaluate('document.querySelectorAll(".favourite-card").length'),0);
	assert.match(await b.evaluate('document.getElementById("favourites-empty").textContent'),/Star links or groups/);
});

test('home favourites react to another tab and never launch links saved only in preferences', async t => {
	const b=await fixture(t); if (!b) return;
	await b.evaluate(`const next=AutoDocs.preferences;next.directory.linkStars=[DirectoryModel.linkID(published[0], published[0].endpoints[0].url),JSON.stringify(['link',['HTTPRoute','missing','missing'],'https://stale.example.com'])];window.dispatchEvent(new StorageEvent('storage',{key:AutoDocs.key,newValue:JSON.stringify(next)}));`);
	await b.wait('document.querySelectorAll(".favourite-link").length===1');
	assert.equal(await b.evaluate('document.querySelector(".favourite-name").textContent'),'argocd');
	assert.equal(await b.evaluate('Boolean(document.querySelector("a[href*=stale]"))'),false);
	await b.evaluate('published=[];');await refresh(b);
	await b.wait('document.querySelectorAll(".favourite-card").length===0');
});

test('partial favourites failures keep live matches, unresolved group members and recovery', async t => {
	const b=await fixture(t); if (!b) return;
	await b.evaluate(`failPublished=true;AutoDocs.update(prefs=>{prefs.directory.groupStars=[DirectoryModel.groupID('label','Infrastructure'),DirectoryModel.groupID('label')];});`);
	await b.wait('document.querySelectorAll(".favourite-card").length===2');
	assert.equal(await b.evaluate('document.getElementById("favourites-notice").hidden'),false);
	assert.match(await b.evaluate('document.getElementById("home-favourites").textContent'),/No resolved URL/);
	assert.equal(await b.evaluate('document.getElementById("service-label").textContent'),'Running');
	await b.evaluate('failPublished=false;');await refresh(b);
	await b.wait('document.querySelectorAll(".favourite-card").length===6');
	assert.equal(await b.evaluate('document.getElementById("favourites-notice").hidden'),true);
});

test('initial home-content error leaves the header and navigation usable', async t => {
	const b=await fixture(t,{setup:setup('failConfig=true;')}); if (!b) return;
	assert.equal(await b.evaluate('document.querySelector("h1").textContent'),'Cluster overview');
	assert.equal(await b.evaluate('document.getElementById("service-label").textContent'),'Running');
	assert.equal(await b.evaluate('document.querySelectorAll(".destination").length'),2);
});

for (const width of [375,1440]) for (const theme of ['light','dark']) {
	test(`compact home fits ${width}px in ${theme} theme`, async t => {
		const b=await fixture(t,{width,setup:setup(`localStorage.setItem('theme',${JSON.stringify(theme)});homeConfig.motd+='\\n'+'LongNotice'.repeat(30);`)});if(!b)return;
		await b.evaluate(starGroup);await b.wait('document.querySelectorAll(".favourite-card").length===4');
		assert.equal(await b.evaluate('document.documentElement.scrollWidth<=innerWidth'),true);
		assert.equal(await b.evaluate('document.documentElement.dataset.theme'),theme);
		assert.equal(await b.evaluate('getComputedStyle(document.getElementById("home-motd")).whiteSpace'),'pre-wrap');
		assert.equal(await b.evaluate('getComputedStyle(document.querySelector(".orbit-art")).display!=="none"'),width>760);
		if(process.env.HOME_PREVIEW_DIR)await b.screenshot(process.env.HOME_PREVIEW_DIR+`/home-${width}-${theme}.png`);
	});
}
