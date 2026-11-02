const {test} = require('node:test');
const assert = require('node:assert/strict');
const {browserFixture, chrome} = require('../../scripts/browser-fixture.cjs');
const {launchpad, setup} = require('../../scripts/dashboard-fixtures.cjs');
async function fixture(t, options = {}) {
 if (!chrome) { t.skip('Set CHROME_BIN to run browser checks'); return; }
 const {extra = '', ...rest} = options;
 const b = await browserFixture(t, {route:'/', setup:setup(launchpad, undefined, extra), ...rest});
 await b.send('Emulation.setEmulatedMedia', {features:[{name:'prefers-reduced-motion',value:'reduce'}]});
 await b.wait('directoryReady');
 return b;
}
const names = '[...document.querySelectorAll(".group-choice-name")].map(n=>n.textContent)';
const selected = 'document.querySelector(".group-choice[aria-pressed=true]").dataset.control';
const choose = 'document.querySelectorAll(".group-choice")[1].click()';
const mode = value => 'modifyDirectory(p=>p.groupBy=' + JSON.stringify(value) + ')';

test('both directory routes restore view and scope; overview stays secondary', async t => {
 for (const route of ['/', '/autodocs-dashboard-page']) {
  const b = await fixture(t, {route, query:'?view=discovered&cluster=b'}); if (!b) return;
  assert.deepEqual(await b.evaluate('[activeView,Inventory.scope.cluster,document.querySelector("h1").textContent]'), ['discovered','b','Discovered']);
  assert.equal(await b.evaluate('document.querySelector("a.nav-item[aria-current]").getAttribute("aria-label")'), 'Directory');
  assert.equal(await b.evaluate('new URL(document.querySelector("a.brand").href).pathname'), '/');
  assert.equal(await b.evaluate('document.querySelector(".rail-footer a[aria-label=Overview]").getAttribute("href")'), '/overview-page?cluster=b');
 }
});

test('sidebar keeps all identities and zero counts while AND filters combine locally', async t => {
 const b = await fixture(t); if (!b) return;
 assert.equal((await b.evaluate(names)).length, 15);
 assert.equal((await b.evaluate(names)).filter(n=>n==='Ungrouped').length, 2);
 await b.evaluate(choose);
 assert.equal(await b.evaluate('document.querySelectorAll(".endpoint-button").length'), 1);
 const requests = await b.evaluate('requestLog.length');
 await b.evaluate('performSearch("no-such-app"); document.getElementById("favourites-toggle").click(); Inventory.setScope({cluster:"b"})');
 assert.equal((await b.evaluate(names)).length, 15);
 assert.equal(await b.evaluate('document.querySelector(".group-choice[aria-pressed=true] .group-choice-count").textContent'), '0');
 assert.match(await b.evaluate('document.getElementById("active-filters").textContent'), /Group:.*Search:.*Favourites.*cluster:/);
 assert.equal(await b.evaluate('document.querySelectorAll(".endpoint-button").length'), 0);
 assert.equal(await b.evaluate('requestLog.length'), requests);
 await b.evaluate('document.querySelector(".empty-state button").click()');
 assert.equal(await b.evaluate('document.querySelectorAll(".endpoint-button").length'), 14);
 assert.equal(await b.evaluate(selected), 'group-choice-');
 await b.evaluate(choose + '; modifyDirectory(p=>p.linkStars=[model.linkID(allData[0],allData[0].endpoints[0].url)]); document.getElementById("favourites-toggle").click(); performSearch("app-0 apps"); document.getElementById("kind-filter").value="HTTPRoute"; document.getElementById("namespace-filter").value="apps"; performSearch("app-0 apps")');
 assert.equal(await b.evaluate('document.querySelectorAll(".endpoint-button").length'), 1);
 assert.equal(await b.evaluate('document.querySelectorAll(".endpoint-link").length'), 1);
 await b.evaluate('document.getElementById("namespace-filter").append(new Option("other","other"));document.getElementById("namespace-filter").value="other";performSearch("app-0")');
 assert.equal(await b.evaluate('document.querySelectorAll(".endpoint-button").length'), 0);
});

test('all grouping modes reuse section identities, reset selections and keep duplicate names distinct', async t => {
 const b = await fixture(t); if (!b) return;
 await b.evaluate('modifyDirectory(p=>{p.groups=[{id:"daily",name:"Daily"}];p.assignments[model.sourceID(allData[0])]="daily"})');
 for (const grouping of ['namespace','kind','custom','cluster','environment','datacentre','environment-cluster','cluster-environment','none','label']) {
  await b.evaluate(mode(grouping));
  assert.equal(await b.evaluate(selected), 'group-choice-');
  assert.deepEqual(await b.evaluate('[...document.querySelectorAll(".group-choice")].slice(1).map(n=>n.dataset.control.slice(13))'), await b.evaluate('model.project(directoryIndex,directoryPrefs(),{query:"",unscoped:true}).sections.filter(g=>g.id!=="flat").map(g=>g.id)'));
  if (grouping === 'namespace') assert.equal((await b.evaluate(names)).filter(n=>n==='Same cluster name / apps').length, 2);
  if (grouping === 'environment-cluster') assert.equal((await b.evaluate(names)).filter(n=>n==='Same environment name / Same cluster name').length, 2);
  if (grouping === 'none') assert.deepEqual(await b.evaluate(names), ['All groups']);
  else await b.evaluate(choose);
 }
 await b.evaluate('fixtureData=fixtureData.filter(r=>r.name!=="app-0");loadDirectory()');
 await b.send('Emulation.setEmulatedMedia', {features:[{name:'prefers-reduced-motion',value:'reduce'}]});
 await b.wait('directoryReady');
 assert.equal(await b.evaluate(selected), 'group-choice-');
 await b.evaluate(choose + '; selectView("discovered")'); await b.wait('directoryReady && activeView==="discovered"');
 assert.equal(await b.evaluate(selected), 'group-choice-');
 await b.reload(); assert.equal(await b.evaluate(selected), 'group-choice-');
});

for (const theme of ['light','dark']) for (const width of [375,1440]) {
 test(`group navigation and drawer at ${width}px in ${theme}`, async t => {
  const b = await fixture(t, {width, height:800, extra:`localStorage.setItem('theme',${JSON.stringify(theme)});`}); if (!b) return;
  if (width === 375) {
   assert.equal(await b.evaluate('document.getElementById("app-navigation").inert'), true);
   await b.evaluate('document.getElementById("navigation-trigger").click()');
   assert.equal(await b.evaluate('document.querySelector(".workspace").inert'), true);
   assert.equal(await b.evaluate('document.getElementById("app-navigation").getAttribute("aria-modal")'), 'true');
   await b.evaluate('document.querySelector(".project-link:last-child").focus()');
   await b.send('Input.dispatchKeyEvent', {type:'keyDown', key:'Tab', code:'Tab', windowsVirtualKeyCode:9});
   assert.equal(await b.evaluate('document.activeElement.className'), 'brand');
   await b.send('Input.dispatchKeyEvent', {type:'keyDown', key:'Tab', code:'Tab', windowsVirtualKeyCode:9, modifiers:8});
   assert.equal(await b.evaluate('document.activeElement.getAttribute("aria-label")'), 'AutoDocs on GitHub (opens in a new tab)');
  }
  await b.evaluate(`(() => {
   const luminance = value => value.match(/[\\d.]+/g).slice(0,3).map(Number).map(x=>{const v=x/255;return v<=.04045?v/12.92:((v+.055)/1.055)**2.4}).reduce((sum,v,i)=>sum+v*[.2126,.7152,.0722][i],0);
   for (const node of document.querySelectorAll('.brand,.nav-item,.group-choice-name,.group-choice-count,.groups-label,.inventory-menu-toggle,.active-filters,.inventory-summary,.directory-heading,.endpoint-name,.endpoint-link')) {
    if (!node.getClientRects().length) continue;
    let parent=node, background=getComputedStyle(document.body).backgroundColor;
    while(parent) { const color=getComputedStyle(parent).backgroundColor;if(color!=='rgba(0, 0, 0, 0)'){background=color;break;}parent=parent.parentElement; }
    const a=luminance(getComputedStyle(node).color),b=luminance(background),ratio=(Math.max(a,b)+.05)/(Math.min(a,b)+.05);
    if(ratio<4.5) throw Error(node.className+' contrast '+ratio);
   }
  })()`);
  if (process.env.AUTODOCS_SCREENSHOTS) await b.screenshot(require('node:path').join(process.env.AUTODOCS_SCREENSHOTS, `launchpad-${width}-${theme}.png`));
  assert.equal(await b.evaluate('document.documentElement.scrollWidth<=innerWidth'), true);
  assert.equal(await b.evaluate('getComputedStyle(document.getElementById("directory-groups")).overflowY'), 'auto');
  assert.equal(await b.evaluate('document.getElementById("directory-groups").scrollHeight > document.getElementById("directory-groups").clientHeight'), true);
  await b.evaluate('document.getElementById("directory-groups").scrollTop=10000');
  assert.equal(await b.evaluate('document.querySelector(".main-nav").getBoundingClientRect().top >= 0 && document.getElementById("sidebar-toggle").getBoundingClientRect().bottom < innerHeight'), true);
  await b.evaluate(choose);
  if (width === 375) {
   assert.equal(await b.evaluate('document.documentElement.dataset.drawer'), 'closed');
   assert.equal(await b.evaluate('document.activeElement.id'), 'navigation-trigger');
   await b.evaluate('document.getElementById("navigation-trigger").click()');
   await b.send('Input.dispatchKeyEvent', {type:'keyDown', key:'Escape', code:'Escape', windowsVirtualKeyCode:27});
   assert.equal(await b.evaluate('document.activeElement.id'), 'navigation-trigger');
   await b.evaluate('document.getElementById("navigation-trigger").click(); document.getElementById("navigation-backdrop").click()');
   assert.equal(await b.evaluate('document.querySelector(".workspace").inert'), false);
   await b.evaluate('document.getElementById("navigation-trigger").click()');
   await b.send('Emulation.setDeviceMetricsOverride', {width:1440,height:800,deviceScaleFactor:1,mobile:false});
   await b.wait('document.documentElement.dataset.drawer === "closed"');
   assert.equal(await b.evaluate('document.querySelector(".workspace").inert'), false);
   assert.equal(await b.evaluate('AutoDocs.preferences.sidebarCollapsed'), null);
   await b.send('Emulation.setDeviceMetricsOverride', {width:375,height:800,deviceScaleFactor:1,mobile:false});
   await b.wait('document.getElementById("app-navigation").inert');
  } else {
   await b.evaluate('document.getElementById("sidebar-toggle").click()');
   await b.reload();
   assert.equal(await b.evaluate('document.documentElement.dataset.sidebar'), 'collapsed');
   assert.equal(await b.evaluate('document.querySelector(".app-navigation").getBoundingClientRect().width'), 72);
   await b.evaluate('document.getElementById("sidebar-toggle").click()');
  }
 });
}

test('blocked storage keeps desktop session collapse independent of mobile state', async t => {
 const b = await fixture(t, {extra:'Object.defineProperty(window,"localStorage",{get(){throw Error("blocked")}});'}); if (!b) return;
 await b.evaluate('document.getElementById("sidebar-toggle").click()');
 assert.equal(await b.evaluate('AutoDocs.preferences.sidebarCollapsed'), true);
 await b.send('Emulation.setDeviceMetricsOverride',{width:375,height:800,deviceScaleFactor:1,mobile:false});
 await b.wait('document.getElementById("app-navigation").inert');
 await b.evaluate('document.getElementById("navigation-trigger").click();document.getElementById("sidebar-toggle").click()');
 assert.equal(await b.evaluate('AutoDocs.preferences.sidebarCollapsed'), true);
 await b.send('Emulation.setDeviceMetricsOverride',{width:1440,height:800,deviceScaleFactor:1,mobile:false});
 await b.wait('!document.getElementById("app-navigation").inert');
 assert.equal(await b.evaluate('document.documentElement.dataset.sidebar'), 'collapsed');
});


test('tile hit targets preserve exact URLs and keep star actions separate', async t => {
 const b = await fixture(t); if (!b) return;
 const result = await b.evaluate(`(() => {
  const single = document.querySelector('[data-single-endpoint="true"]'); single.scrollIntoView({block:'center'});
  const title = single.querySelector('.endpoint-name').getBoundingClientRect();
  const hit = document.elementFromPoint(title.x+title.width/2,title.y+title.height/2).closest('a');
  const multi = document.querySelector('[data-single-endpoint="false"]');
  return {hit:hit?.getAttribute('href'), expected:single.querySelector('a').getAttribute('href'), multiple:[...multi.querySelectorAll('.endpoint-link')].map(a=>a.getAttribute('href')), nested:!!document.querySelector('a button,a select'), star:single.querySelector('.star-button').getBoundingClientRect().toJSON()};
 })()`);
 assert.equal(result.hit, result.expected);
 assert.deepEqual(result.multiple, ['https://one.test/path?x=1','https://two.test:8443/second']);
 assert.equal(result.nested, false);
 const {x,y,width,height} = result.star;
 await b.send('Input.dispatchMouseEvent', {type:'mousePressed',x:x+width/2,y:y+height/2,button:'left',clickCount:1});
 await b.send('Input.dispatchMouseEvent', {type:'mouseReleased',x:x+width/2,y:y+height/2,button:'left',clickCount:1});
 assert.equal(await b.evaluate('AutoDocs.preferences.directory.linkStars.length'), 1);
 assert.equal(await b.evaluate('location.pathname'), '/');
 assert.equal(await b.evaluate('AutoDocs.preferences.directory.groupStars.length'), 0);
 assert.equal(await b.evaluate('document.querySelectorAll(".endpoint-button:not(:has(a))").length'), 1);
});
