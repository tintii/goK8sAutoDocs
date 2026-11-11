const {test} = require('node:test');
const assert = require('node:assert/strict');
const {browserFixture, chrome} = require('../../scripts/browser-fixture.cjs');

const service = (name, extra = {}) => ({clusterId:'a',clusterName:'Cluster A',name,namespace:'apps',kind:'Service',serviceType:'ExternalName',externalName:'vendor.example.net',labels:{'k8sautodocs.io/group':'external'},annotations:{},hostnames:[{host:'app.example.com'}],dns:{targets:['192.0.2.10','2001:db8::1'],targetSource:'annotation'},endpoints:[],issues:[],...extra});
const published = [
	service('database'),
	service('website',{annotations:{'k8sautodocs.io/scheme':'https'},hostnames:[{host:'app.example.com'},{host:'other.example.com'}],endpoints:[{url:'https://app.example.com',host:'app.example.com'},{url:'https://other.example.com:8443',host:'other.example.com'}]}),
	service('conflict',{annotations:{'external-dns.alpha.kubernetes.io/hostname':'a.example.com','external-dns.kubernetes.io/hostname':'<img src=x onerror="window.injected=true">'},hostnames:[],issues:['Conflicting ExternalDNS hostname annotations'],dns:{targets:[]}}),
	service('database',{clusterId:'b',clusterName:'Cluster B',dns:{targets:['second.vendor.example'],targetSource:'externalName'}}),
];
const discovered = [service('candidate',{annotations:{'external-dns.kubernetes.io/hostname':'candidate.example.com'},hostnames:[{host:'candidate.example.com'}]})];
function setup(extra = '') {
	return `
		const published=${JSON.stringify(published)}, discovered=${JSON.stringify(discovered)};
		window.requests=[];window.copied=[];window.copyFails=false;
		Object.defineProperty(navigator,'clipboard',{configurable:true,value:{writeText:async value=>{if(copyFails)throw new Error('denied');copied.push(value);}}});
		window.fetch=async(url,init={})=>{requests.push({url,method:init.method||'GET'});return {ok:true,headers:{get:()=>url.includes('discovered')?'["Service fixture warning"]':'[]'},json:async()=>url.endsWith('/health')?{status:'running'}:url.endsWith('/home')?{title:'Overview',description:'',environment:'',motd:'',footer:''}:url.includes('view=discovered')?discovered:published};};
		${extra}
	`;
}
async function fixture(t, options = {}) {
	if (!chrome) {t.skip('Set CHROME_BIN to run browser checks');return;}
	const b = await browserFixture(t,{setup:setup(),...options});
	if (!options.route || options.route !== '/overview-page') await b.wait('directoryReady');
	return b;
}

test('Services display separate declarations as safe text and search DNS metadata',async t=>{
	const b=await fixture(t);if(!b)return;
	assert.equal(await b.evaluate('document.querySelectorAll(".endpoint-button").length'),4);
	assert.equal(await b.evaluate('document.getElementById("filtered-count").textContent'),'Showing 4 of 4 loaded sources · 2 URLs');
	assert.equal(await b.evaluate('document.querySelectorAll(".endpoint-link").length'),2);
	assert.equal(await b.evaluate('[...document.querySelectorAll(".endpoint-button")].find(c=>c.querySelector(".endpoint-name").textContent==="database").querySelector(".endpoint-host").textContent'),'DNS only · No web link configured');
	assert.equal(await b.evaluate('Boolean(document.querySelector(".service-dns img")) || Boolean(window.injected)'),false);
	assert.match(await b.evaluate('document.querySelector(".endpoint-issues").textContent'),/Conflicting ExternalDNS hostname/);
	assert.match(await b.evaluate('[...document.querySelectorAll(".service-dns")].find(c=>c.textContent.includes("Advertised name")).textContent'),/Advertised nameapp.example.com/);
	assert.match(await b.evaluate('document.querySelector(".service-dns").textContent'),/Declared externalNamevendor.example.net/);
	assert.match(await b.evaluate('[...document.querySelectorAll(".service-dns")].find(c=>c.textContent.includes("Selected target")).textContent'),/Selected target192.0.2.10/);
	assert.match(await b.evaluate('document.querySelector(".card-meta").textContent'),/Service type: ExternalName/);
	assert.deepEqual(await b.evaluate('[...document.querySelectorAll(".endpoint-link")].map(a=>[a.getAttribute("href"),a.target,a.rel])'),[['https://app.example.com','_blank','noopener noreferrer'],['https://other.example.com:8443','_blank','noopener noreferrer']]);
	await b.evaluate('document.getElementById("kind-filter").value="Service";performSearch("second.vendor.example externalname");');
	assert.equal(await b.evaluate('document.querySelectorAll(".endpoint-button").length'),1);
	assert.equal(await b.evaluate('document.getElementById("filtered-count").textContent'),'Showing 1 of 4 loaded sources · 0 URLs');
	await b.evaluate('performSearch("vendor.example.net");');
	assert.equal(await b.evaluate('document.querySelectorAll(".endpoint-button").length'),4);
	await b.evaluate('performSearch("2001:db8::1");');
	assert.equal(await b.evaluate('document.querySelectorAll(".endpoint-button").length'),2);
});

test('DNS copy supports keyboard activation, exact values and honest failure feedback',async t=>{
	const b=await fixture(t);if(!b)return;
	await b.evaluate('window.dnsCard=[...document.querySelectorAll(".endpoint-button")].find(c=>c.querySelector(".endpoint-name").textContent==="database");dnsCard.querySelector(".dns-copy").focus()');
	assert.equal(await b.evaluate('document.activeElement.tagName'),'BUTTON');
	await b.send('Input.dispatchKeyEvent',{type:'keyDown',key:'Enter',code:'Enter',windowsVirtualKeyCode:13,text:'\r',unmodifiedText:'\r'});
	await b.send('Input.dispatchKeyEvent',{type:'keyUp',key:'Enter',code:'Enter',windowsVirtualKeyCode:13});
	await b.wait('copied.length===1');
	assert.deepEqual(await b.evaluate('copied'),['app.example.com']);
	assert.equal(await b.evaluate('dnsCard.querySelector(".dns-copy-status").textContent'),'Copied app.example.com');
	await b.evaluate('[...document.querySelectorAll(".dns-copy")].find(n=>n.getAttribute("aria-label")==="Copy selected target: 2001:db8::1").click()');
	await b.wait('copied.length===2');
	assert.deepEqual(await b.evaluate('copied'),['app.example.com','2001:db8::1']);
	await b.evaluate('copyFails=true;dnsCard.querySelector(".dns-copy").click()');
	await b.wait('dnsCard.querySelector(".dns-copy-status").textContent.startsWith("Could not copy")');
	assert.equal(await b.evaluate('copied.length'),2);
	assert.equal(await b.evaluate('location.pathname'),'/autodocs-dashboard-page');
});

test('Services preserve cluster identity, groups, DNS-only favourites and web-only link stars',async t=>{
	const b=await fixture(t);if(!b)return;
	assert.equal(await b.evaluate('new Set(allData.map(DirectoryModel.sourceID)).size'),4);
	await b.evaluate('document.querySelector(".group-heading .star-button").click();document.getElementById("favourites-toggle").click()');
	assert.equal(await b.evaluate('document.querySelectorAll(".endpoint-button").length'),4);
	assert.equal(await b.evaluate('document.querySelectorAll(".link-list .star-button").length'),2);
	await b.evaluate('modifyDirectory(p=>{p.groups=[{id:"mine",name:"My DNS"}];p.assignments[model.sourceID(allData.find(r=>r.name==="database"&&r.clusterId==="a"))]="mine";p.groupBy="custom";});performSearch("My DNS");');
	assert.equal(await b.evaluate('document.querySelectorAll(".endpoint-button").length'),1);
	await b.evaluate('performSearch("");modifyDirectory(p=>{p.groupStars=[];});');
	assert.equal(await b.evaluate('document.querySelectorAll(".endpoint-button").length'),0);
	await b.evaluate('document.getElementById("favourites-toggle").click();toggleStar("linkStars",model.linkID(allData.find(r=>r.name==="website"),"https://app.example.com"));document.getElementById("favourites-toggle").click()');
	assert.equal(await b.evaluate('document.querySelectorAll(".endpoint-button").length'),1);
	assert.equal(await b.evaluate('document.querySelectorAll(".endpoint-link").length'),1);
	await b.navigate('/overview-page');
	await b.wait('document.querySelectorAll(".favourite-link").length===1');
	assert.equal(await b.evaluate('document.querySelector(".favourite-link").getAttribute("href")'),'https://app.example.com');
});

test('DNS Services retain grid/list content and view-specific results without overflow',async t=>{
	const b=await fixture(t,{width:375});if(!b)return;
	for (const density of ['grid','list']) {
		await b.evaluate('modifyDirectory(p=>{p.density='+JSON.stringify(density)+';})');
		assert.equal(await b.evaluate('document.querySelectorAll(".service-dns").length'),4);
		assert.equal(await b.evaluate('document.documentElement.scrollWidth<=innerWidth'),true);
	}
	await b.evaluate('selectView("discovered")');
	assert.equal(await b.evaluate('document.querySelectorAll(".endpoint-button").length'),1);
	assert.equal(await b.evaluate('document.querySelector(".endpoint-name").textContent'),'candidate');
	assert.match(await b.evaluate('document.getElementById("collection-warnings").textContent'),/Service fixture warning/);
	assert.equal(await b.evaluate('document.querySelectorAll(".endpoint-link").length'),0);
	await b.evaluate('selectView("published")');
	assert.equal(await b.evaluate('document.querySelectorAll(".service-dns").length'),4);
	assert.equal(await b.evaluate('document.getElementById("collection-warnings").hidden'),true);
	await b.screenshot('/tmp/autodocs-services-mobile.png');
});
