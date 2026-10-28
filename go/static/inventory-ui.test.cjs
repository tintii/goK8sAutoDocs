const {test} = require('node:test');
const assert = require('node:assert/strict');
const {browserFixture, chrome} = require('../../scripts/browser-fixture.cjs');
const row = (cluster, environment, name = 'same') => ({clusterId:cluster, clusterName:cluster.toUpperCase(), name, namespace:'apps', kind:'HTTPRoute', labels:{'k8sautodocs.io/group':'shared'}, classification:environment ? {environment:{id:environment.toLowerCase(),display:environment.toUpperCase()},datacentre:{id:'pgt',display:'pgt'}} : {}, endpoints:[{url:'https://' + cluster + '.example.com'}], containers:[{name:'main',image:'app',tag:'v1'}]});
const data = [row('a','NPE00'), row('b','npe00'), row('a',null,'unknown')];
function setup(extra = '') {
	return [
		'window.rows='+JSON.stringify(data)+';window.requests=[];',
		'const part = {lastSuccessAt:new Date(Date.now()-120000).toISOString(),lastAttemptAt:new Date().toISOString(),error:"",warnings:[],values:{environment:{npe00:true},datacentre:{pgt:true}}};',
		'window.shared={generation:1,running:false,completed:2,total:2,serverTime:new Date().toISOString(),nextAutomaticAt:new Date(Date.now()+180000).toISOString(),nextManualAt:new Date(Date.now()-1000).toISOString(),legacyClusterID:"a",dimensions:{environment:{npe00:"NPE00"},datacentre:{pgt:"pgt"}},clusters:["a","b"].map(id=>({id,name:id.toUpperCase(),parts:{ingresses:part,httproutes:part,workloads:part}}))};',
		'window.fetch=async(url,options={})=>{requests.push({url,method:options.method||"GET",body:options.body});if(url.endsWith("/discovery-refresh")){shared={...shared,running:true,completed:0};}const payload=url.includes("/discovery-")?{...shared,serverTime:new Date().toISOString()}:url.endsWith("/health")?{status:"running"}:url.endsWith("/home")?{title:"Overview",description:"",environment:"",motd:"",footer:""}:url.includes("discovered")?[]:rows;return {ok:true,status:200,headers:{get:()=> "[]"},json:async()=>structuredClone(payload)};};',
		extra
	].join('\n');
}
async function fixture(t, options = {}) { if(!chrome){t.skip('Chrome required');return;} const b=await browserFixture(t,{setup:setup(),...options});await b.wait('Inventory.status?.generation===1');return b; }

test('cluster identities, canonical groups and missing labels remain independent',async t=>{
	const b=await fixture(t);if(!b)return;
	assert.equal(await b.evaluate('new Set(allData.map(DirectoryModel.sourceID)).size'),3);
	await b.evaluate("modifyDirectory(p=>{p.groupBy='environment';});");
	assert.deepEqual(await b.evaluate('[...document.querySelectorAll(".namespace-header")].map(n=>n.textContent)'),['NPE00','Unclassified environment']);
	await b.evaluate("modifyDirectory(p=>{p.groupBy='cluster-environment';});");
	assert.deepEqual(await b.evaluate('[...document.querySelectorAll(".namespace-header")].map(n=>n.textContent)'),['A / NPE00','A / Unclassified environment','B / NPE00']);
	await b.evaluate("toggleStar('linkStars',DirectoryModel.linkID(allData.find(r=>r.clusterId==='a'&&r.name==='same'),'https://a.example.com'));document.getElementById('favourites-toggle').click();");
	assert.equal(await b.evaluate('document.querySelectorAll(".endpoint-button").length'),1);
	assert.match(await b.evaluate('document.querySelector(".endpoint-link").href'),/a\.example/);
	await b.evaluate("document.getElementById('favourites-toggle').click();Inventory.setScope({environment:'npe00',cluster:'b'});");
	assert.equal(await b.evaluate('document.querySelectorAll(".endpoint-button").length'),1);
	assert.equal(await b.evaluate('document.getElementById("total-resources").textContent'),'1');
	assert.equal(await b.evaluate('requests.filter(r=>r.url.includes("discovery-refresh")).length'),0);
	await b.evaluate("Inventory.setScope({cluster:'',environment:''});allData[0].endpoints=[{url:location.origin+'/'}];directoryIndex=model.index(allData,directoryPrefs());performSearch('');Inventory.setScope({cluster:'a'});");
	assert.equal(await b.evaluate('document.querySelector(".endpoint-link").getAttribute("href")===location.origin+"/"'),true);
});

test('scope survives page navigation and history with unknown values clearable',async t=>{
	const b=await fixture(t);if(!b)return;
	await b.evaluate("Inventory.setScope({environment:'npe00',cluster:'b'});");
	const href=await b.evaluate('document.querySelector("a.nav-item[href*=versions]").getAttribute("href")');
	await b.navigate(href);await b.wait('document.getElementById("table-container").style.display==="block"');
	assert.equal(await b.evaluate('document.getElementById("total-resources").textContent'),'1');
	assert.equal(await b.evaluate('document.querySelector("#table-body tr td:nth-child(9)").textContent'),'B');
	await b.evaluate("Inventory.setScope({environment:'gone'});");
	assert.equal(await b.evaluate('document.getElementById("scope-environment").selectedOptions[0].textContent'),'gone (unavailable)');
	assert.match(await b.evaluate('document.getElementById("filtered-count").textContent'),/Showing 0/);
	await b.send('Runtime.evaluate',{expression:'history.back()'});await b.wait('Inventory.scope.environment==="npe00"');
});

test('legacy preferences migrate only to configured original cluster',async t=>{
	const legacy={version:1,theme:'dark',directory:{groupBy:'custom',density:'grid',collapsed:[],groups:[{id:'g',name:'Mine'}],assignments:{'["HTTPRoute","apps","same"]':'g'},groupStars:[],linkStars:['["link",["HTTPRoute","apps","same"],"https://a.example.com"]']},versions:{widths:{resource:250},sort:[]}};
	const b=await fixture(t,{setup:setup('localStorage.setItem("k8sautodocs.preferences",'+JSON.stringify(JSON.stringify(legacy))+');')});if(!b)return;
	assert.equal(await b.evaluate('AutoDocs.preferences.version'),2);
	assert.deepEqual(await b.evaluate('Object.keys(AutoDocs.preferences.directory.assignments)'),['["a","HTTPRoute","apps","same"]']);
	assert.equal(await b.evaluate('AutoDocs.preferences.versions.widths.resource'),250);
	assert.equal(await b.evaluate('AutoDocs.preferences.theme'),'dark');
	assert.equal(await b.evaluate('DirectoryModel.group(allData.find(r=>r.clusterId==="b"),"custom",AutoDocs.preferences.directory).fallback'),true);
});

test('manual update posts shared generation and freshness remains separate from availability',async t=>{
	const b=await fixture(t);if(!b)return;
	assert.match(await b.evaluate('document.getElementById("inventory-state").textContent'),/Updated 2m ago/);
	await b.evaluate('document.getElementById("inventory-update").click()');await b.wait('Inventory.status.running');
	assert.equal(await b.evaluate('document.getElementById("inventory-update").disabled'),true);
	assert.equal(await b.evaluate('document.getElementById("service-label").textContent'),'Running');
	assert.deepEqual(await b.evaluate('requests.filter(r=>r.method==="POST").map(r=>JSON.parse(r.body))'),[{generation:1}]);
	await b.evaluate("shared.running=false;shared.generation=2;shared.nextManualAt=new Date(Date.now()+30000).toISOString();shared.clusters[1]={...shared.clusters[1],parts:{...shared.clusters[1].parts,httproutes:{...shared.clusters[1].parts.httproutes,error:'access forbidden'}}};document.dispatchEvent(new Event('visibilitychange'));");
	await b.wait('Inventory.status.generation===2');
	assert.match(await b.evaluate('document.getElementById("inventory-state").textContent'),/incomplete or stale/);
	assert.match(await b.evaluate('document.getElementById("inventory-update").textContent'),/Update in/);
	assert.equal(await b.evaluate('document.querySelectorAll(".endpoint-button").length'),3);
});

test('mobile scope and mixed inventory fit without document overflow',async t=>{
	const b=await fixture(t,{width:375});if(!b)return;
	assert.equal(await b.evaluate('document.documentElement.scrollWidth<=innerWidth'),true);
	await b.screenshot('/tmp/autodocs-multicluster-mobile.png');
});
