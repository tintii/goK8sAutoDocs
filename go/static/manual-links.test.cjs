const {test} = require('node:test');
const assert = require('node:assert/strict');
const {browserFixture, chrome} = require('../../scripts/browser-fixture.cjs');
const url = 'https://wiki.example:8006/runbooks/media?q=a%20b&next=%2F#recovery';
const manual = (id,scope={}) => ({sourceType:'manual-link', kind:'manual-link',name:id,title:id,provenance:{clusterId:'a',namespace:'central-storage',configMapName:'links',entryId:id},classification:{},labels:{'k8sautodocs.io/group':'documentation'},endpoints:[{url}],...scope});
const discovered = (environment) => ({name:'Discovered',kind:'HTTPRoute',namespace:'media',clusterId:'b',clusterName:'B',classification:environment ? {environment:{id:environment,display:environment}} : {},labels:{'k8sautodocs.io/group':'documentation'},endpoints:[{url:'https://discovered.example/'}]});
const rows=[manual('Confluence',{collection:{error:'access forbidden'}}),manual('Proxmox',{classification:{datacentre:{id:'home',display:'Home'}}}),manual('Runbook',{clusterId:'b',clusterName:'B',namespace:'media',classification:{environment:{id:'prod',display:'Production'},datacentre:{id:'home',display:'Home'}}}),manual('Namespace only',{namespace:'media'}),discovered()];
function setup(extra='') {
 return `window.rows=${JSON.stringify(rows)};window.requests=[];
 const time=new Date(Date.now()-600000).toISOString();
 const part={lastSuccessAt:new Date().toISOString(),error:'',values:{environment:{prod:true},datacentre:{home:true}}};
 window.shared={generation:1,serverTime:new Date().toISOString(),running:false,total:2,completed:2,dimensions:{environment:{prod:'Production'},datacentre:{home:'Home'}},clusters:['a','b'].map(id=>({id,name:id.toUpperCase(),parts:{ingresses:part,httproutes:part,services:part,workloads:part,'manual-links':id==='a'?{lastSuccessAt:time,error:'access forbidden',documents:{'central-storage/links':{lastSuccessAt:time,error:'',scopes:[{sourceType:'manual-link',classification:{}}]}}}:{lastSuccessAt:new Date().toISOString(),documents:{}}}}))};
 window.fetch=async(path)=>{requests.push(path);const payload=path.includes('discovery-status')?shared:path.endsWith('/health')?{status:'running'}:path.endsWith('/home')?{title:'Overview'}:path.includes('discovered')?[]:rows;return {ok:true,status:200,headers:{get:()=> '[]'},json:async()=>structuredClone(payload)};};
 ${extra}`;
}
async function fixture(t,options={}) { if(!chrome){t.skip('Chrome required');return;}const b=await browserFixture(t,{setup:setup(),...options});await b.wait('Inventory.status?.generation===1');await b.wait('document.querySelectorAll(".endpoint-button").length>0');return b; }

test('manual dimensions are independently shared and intersect with namespace, kind, search, groups and favourites',async t=>{
 const b=await fixture(t);if(!b)return;
 await b.evaluate(`Inventory.setScope({cluster:'b',environment:'prod',datacentre:'home'});`);
 assert.equal(await b.evaluate('document.querySelectorAll(".endpoint-button").length'),4);
 await b.evaluate(`document.getElementById('namespace-filter').value='media';performSearch('');`);
 assert.equal(await b.evaluate('document.querySelectorAll(".endpoint-button").length'),4);
 await b.evaluate(`Inventory.setScope({cluster:'a',environment:'other',datacentre:'office'});`);
 assert.deepEqual(await b.evaluate('[...document.querySelectorAll(".endpoint-name")].map(n=>n.textContent)'),['Confluence','Namespace only']);
 await b.evaluate(`document.getElementById('namespace-filter').append(new Option('other','other'));document.getElementById('namespace-filter').value='other';performSearch('confluence documentation manual link');`);
 assert.equal(await b.evaluate('document.querySelectorAll(".endpoint-button").length'),1);
 assert.equal(await b.evaluate('document.querySelector(".endpoint-link").getAttribute("href")'),url);
 assert.equal(await b.evaluate('document.querySelector(".endpoint-link").rel'),'noopener noreferrer');
 assert.equal(await b.evaluate('document.querySelector(".card-meta").textContent.includes("central-storage")'),false);
 assert.match(await b.evaluate('document.querySelector(".related-resources").textContent'),/a \/ central-storage \/ ConfigMap links \/ Confluence/);
 await b.evaluate(`document.getElementById('kind-filter').value='HTTPRoute';performSearch('');`);
 assert.equal(await b.evaluate('document.querySelectorAll(".endpoint-button").length'),0);
 await b.evaluate(`document.getElementById('kind-filter').value='manual-link';performSearch('central-storage');`);
 assert.equal(await b.evaluate('document.querySelectorAll(".endpoint-button").length'),0);
 await b.evaluate(`performSearch('');document.getElementById('favourites-toggle').click();`);
 assert.equal(await b.evaluate('document.querySelectorAll(".endpoint-button").length'),0);
 await b.evaluate(`toggleStar('linkStars',model.linkID(allData[0],allData[0].endpoints[0].url));`);
 assert.equal(await b.evaluate('document.querySelectorAll(".endpoint-button").length'),1);
 assert.equal(await b.evaluate('requests.filter(path=>path.includes("discovery-refresh")).length'),0);
});

test('unknown and shared typed groups remain distinct from literal names in every dimension and composite',async t=>{
 const b=await fixture(t);if(!b)return;
 const result=await b.evaluate(`(()=>{
  const prefs=directoryPrefs();
  const global=allData[0],unknown=allData.find(row=>row.kind==='HTTPRoute');
  const literal={...global,classification:{environment:{id:'literal',display:'Shared across environments'}}};
  const groups=[global,unknown,literal].map(row=>model.group(row,'environment',prefs));
  modifyDirectory(p=>{p.groupBy='environment'});
  return {ids:groups.map(g=>g.id),names:[...document.querySelectorAll('.namespace-header')].map(n=>n.textContent),
   cluster:model.group(global,'cluster',prefs).name,
   ns:model.group(allData.find(r=>r.name==='Namespace only'),'namespace',prefs).name,
   partial:model.group({...global,clusterId:'b',clusterName:'B'},'namespace',prefs).name,
   composite:model.group({...global,classification:{environment:{id:'prod',display:'Production'}}},'environment-cluster',prefs).name};
 })()`);
 assert.equal(new Set(result.ids).size,3);
 assert.deepEqual(result.names,['Production','Shared across environments','Unclassified environment']);
 assert.equal(result.cluster,'Shared across clusters');assert.equal(result.ns,'Shared across clusters / media');assert.equal(result.partial,'B / Shared across namespaces');assert.equal(result.composite,'Production / Shared across clusters');
 // Selecting a concrete group does not pull the separate shared bucket into it.
 await b.evaluate(`modifyDirectory(p=>{p.groupBy='cluster'});Inventory.setScope({cluster:'b'});document.querySelectorAll('.group-choice')[1].click();`);
 assert.equal(await b.evaluate('[...document.querySelectorAll(".endpoint-name")].some(n=>n.textContent==="Confluence")'),false);
 await b.evaluate(`Inventory.setScope({cluster:'',environment:'\u0000',datacentre:'\u0000'});selectedGroup='';performSearch('');`);
 assert.deepEqual(await b.evaluate('[...document.querySelectorAll(".endpoint-name")].map(n=>n.textContent).sort()'),['Confluence','Discovered','Namespace only']);
});

test('manual edits retain custom assignment and exact URL stars; duplicates and URL edits are independent',async t=>{
 const b=await fixture(t);if(!b)return;
 await b.evaluate(`modifyDirectory(p=>{p.groups=[{id:'mine',name:'Mine'}];p.assignments[model.sourceID(allData[0])]='mine';p.linkStars=[model.linkID(allData[0],allData[0].endpoints[0].url)];});
 rows[0].name=rows[0].title='Edited title <script>alert(1)</script>';rows[0].namespace='new';rows[0].clusterId='b';rows[0].classification={environment:{id:'prod',display:'Production'}};rows[0].labels['k8sautodocs.io/group']='edited';loadDirectory();`);
 await b.wait('directoryReady && allData[0].name.startsWith("Edited")');
 assert.equal(await b.evaluate('directoryPrefs().assignments[model.sourceID(allData[0])]'),'mine');
 assert.equal(await b.evaluate('directoryPrefs().linkStars.includes(model.linkID(allData[0],allData[0].endpoints[0].url))'),true);
 assert.equal(await b.evaluate('document.querySelectorAll(".endpoint-name script").length'),0);
 await b.evaluate(`document.getElementById('favourites-toggle').click();`);
 assert.equal(await b.evaluate('document.querySelectorAll(".endpoint-button").length'),1);
 await b.navigate('/');await b.wait('directoryReady');
 assert.equal(await b.evaluate('directoryPrefs().assignments[model.sourceID(allData[0])]'),'mine');
 await b.evaluate(`rows[0].endpoints=[{url:'https://wiki.example/new'}];loadDirectory();`);await b.wait('directoryReady && allData[0].endpoints[0].url.endsWith("new")');
 await b.evaluate(`document.getElementById('favourites-toggle').click();`);
 assert.equal(await b.evaluate('document.querySelectorAll(".endpoint-button").length'),0);
 assert.equal(await b.evaluate('directoryPrefs().assignments[model.sourceID(allData[0])]'),'mine');
});

test('scope restored from URL includes stale shared declarations collected in another cluster and pending collectors',async t=>{
 const b=await fixture(t);if(!b)return;
 await b.navigate('/?cluster=b&environment=prod&datacentre=home');await b.wait('directoryReady && Inventory.status?.generation===1');
 assert.equal(await b.evaluate('document.querySelectorAll(".endpoint-button").length'),4);
 assert.match(await b.evaluate('document.getElementById("inventory-state").textContent'),/(?:9|10)m ago.*incomplete or stale/);
 assert.match(await b.evaluate('document.getElementById("inventory-details-list").textContent'),/A \/ manual-links \/ central-storage\/links.*access forbidden/);
 await b.evaluate(`shared.clusters[0].parts['manual-links']={lastSuccessAt:null,error:'access forbidden',documents:{}};shared.generation++;document.dispatchEvent(new Event('visibilitychange'));`);
 await b.wait('Inventory.status.generation===2');
 assert.match(await b.evaluate('document.getElementById("inventory-details-list").textContent'),/A \/ manual-links: never collected.*access forbidden/);
});

for(const width of [375,1440]) for(const density of ['grid','list']) test(`manual cards preserve text, exact URL and keyboard controls at ${width}px in ${density}`,async t=>{
 const b=await fixture(t,{width});if(!b)return;
 await b.evaluate(`modifyDirectory(p=>{p.density=${JSON.stringify(density)}});`);
 assert.equal(await b.evaluate('document.documentElement.scrollWidth<=innerWidth'),true);
 assert.equal(await b.evaluate('document.querySelector(".endpoint-link").getAttribute("href")'),url);
 await b.evaluate(`document.querySelector('.endpoint-link').focus()`);
 assert.equal(await b.evaluate('document.activeElement.className'),'endpoint-link');
 assert.equal(await b.evaluate('[...document.querySelectorAll(".star-button")].every(n=>n.getAttribute("aria-label"))'),true);
});

test('overview favourites share manual scope, identity and stale warning semantics',async t=>{
 if(!chrome){t.skip('Chrome required');return;}
 const b=await browserFixture(t,{setup:setup()});
 await b.wait('directoryReady');
 await b.evaluate(`toggleStar('linkStars',model.linkID(allData[0],allData[0].endpoints[0].url));`);
 await b.navigate('/overview-page?cluster=b&environment=prod&datacentre=home');
 await b.wait('document.querySelectorAll(".favourite-card").length===1');
 assert.equal(await b.evaluate('document.querySelector(".favourite-name").textContent'),'Confluence');
 assert.equal(await b.evaluate('document.querySelector(".favourite-link").getAttribute("href")'),url);
 assert.equal(await b.evaluate('document.querySelector(".favourite-context").textContent'),'Manual link');
 assert.match(await b.evaluate('document.querySelector(".favourite-card .endpoint-issues").textContent'),/Retained declaration/);
});


test('document freshness follows relevance independently within a partially invalid component',async t=>{
 const b=await fixture(t);if(!b)return;
 await b.evaluate(`
  shared.clusters[0].parts['manual-links']={lastSuccessAt:new Date(Date.now()-600000).toISOString(),error:'manual declaration validation incomplete',documents:{
   'central-storage/outside':{lastSuccessAt:new Date(Date.now()-600000).toISOString(),error:'links[0].url: invalid HTTP(S) destination',scopes:[{sourceType:'manual-link',clusterId:'a',classification:{}}]},
   'central-storage/inside':{lastSuccessAt:new Date().toISOString(),error:'',scopes:[{sourceType:'manual-link',clusterId:'b',classification:{}}]}
  }};
  shared.generation++;document.dispatchEvent(new Event('visibilitychange'));
 `);
 await b.wait('Inventory.status.generation===2');
 await b.evaluate(`Inventory.setScope({cluster:'b'});`);
 assert.doesNotMatch(await b.evaluate('document.getElementById("inventory-state").textContent'),/incomplete or stale|10m ago/);
 assert.doesNotMatch(await b.evaluate('document.getElementById("inventory-details-list").textContent'),/central-storage\/outside/);
 await b.evaluate(`Inventory.setScope({cluster:'a'});`);
 assert.match(await b.evaluate('document.getElementById("inventory-state").textContent'),/(?:9|10)m ago.*incomplete or stale/);
});
