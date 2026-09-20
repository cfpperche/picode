import { test } from 'node:test';
import assert from 'node:assert/strict';
import { backgroundRGB, paintRect, collectNativeLayers, installNativeLayers } from './nativeLayers.js';

function node(rect, { parentElement = null, hidden = false, display = 'block' } = {}) {
  return { parentElement, closest: () => hidden ? {} : null,
    getBoundingClientRect: () => ({ ...rect, width: rect.right-rect.left, height: rect.bottom-rect.top }),
    style: { display, visibility: 'visible' } };
}
function fixture(pages, overlays) {
  const document = { querySelectorAll: s => s.startsWith('.web-tab-host') ? pages : overlays };
  const window = { innerWidth:1000,innerHeight:800,getComputedStyle:el=>el.style };
  return {document, window};
}
const pageRect = {left:300,top:100,right:1000,bottom:800};
const menuRect = {left:600,top:80,right:850,bottom:220};

test('native geometry keeps page holes and overlapping menu regions separately', () => {
  const body = node({left:0,top:0,right:1000,bottom:800});
  const page=node(pageRect,{parentElement:body});
  const {document,window}=fixture([page],[node(menuRect)]);
  const state=collectNativeLayers(document,window);
  assert.deepEqual(state.pages,[pageRect]);
  assert.deepEqual(state.overlays,[menuRect]);
  assert.deepEqual([...state.clear],[page,body]);
});

test('closed tabs and hidden or offscreen overlays cannot consume input', () => {
  const {document,window}=fixture(
    [node(pageRect),node(pageRect,{hidden:true})],
    [node(menuRect,{hidden:true}),node(menuRect,{display:'none'}),node({left:0,top:900,right:20,bottom:950})],
  );
  const state=collectNativeLayers(document,window);
  assert.equal(state.pages.length,1);
  assert.deepEqual(state.overlays,[]);
});

test('no visible native page means no transparent ancestors or floating regions', () => {
  const {document,window}=fixture([node(pageRect,{hidden:true})],[node(menuRect)]);
  const state=collectNativeLayers(document,window);
  assert.deepEqual(state.pages,[]);
  assert.deepEqual(state.overlays,[]);
  assert.equal(state.clear.size,0);
});

test('a modal backdrop owns the entire viewport without removing the live page', () => {
  const backdrop={left:0,top:0,right:1000,bottom:800};
  const {document,window}=fixture([node(pageRect)],[node(backdrop),node(menuRect)]);
  const state=collectNativeLayers(document,window);
  assert.deepEqual(state.pages,[pageRect]);
  assert.deepEqual(state.overlays,[backdrop,menuRect]);
});

test('theme colors retain opaque native window background behind non-page content', () => {
  assert.deepEqual(backgroundRGB(' #0e0e11 '),[14,14,17]);
  assert.deepEqual(backgroundRGB('#f0f2f7'),[240,242,247]);
  assert.throws(()=>backgroundRGB('transparent'));
});

test('plain browsers and older shells do not install native geometry hooks', () => {
  for(const window of [{},{__TAURI__:{core:{invoke:()=>assert.fail('unexpected IPC')}}}]) {
    const dispose=installNativeLayers({window,document:{}});
    assert.equal(typeof dispose,'function');dispose();
  }
});

test('native paint includes shadow fade outside the overlay layout box', () => {
  assert.deepEqual(paintRect(menuRect, 'rgba(0, 0, 0, 0.2) 0px 10px 20px 0px'),
    {left:570, top:60, right:880, bottom:260});
  assert.deepEqual(paintRect(menuRect, 'inset 0px 0px 8px 2px'), menuRect);
});

// Exercise the bridge contract: only one update may be in flight, and a
// later layout must be delivered after its predecessor acknowledges.
test('native updates serialize layout changes and dispose transparent state', async () => {
  const callbacks = new Map(); let id = 0, mutation;
  const attrs = new Map(); const classes = new Set();
  let rect = pageRect;
  const host = node(pageRect);
  host.getBoundingClientRect = () => ({...rect,width:rect.right-rect.left,height:rect.bottom-rect.top});
  host.classList = {add:x=>classes.add(x),remove:x=>classes.delete(x)};
  host.setAttribute = (k,v)=>attrs.set(k,v);
  host.removeAttribute = k=>attrs.delete(k);
  const calls = []; const releases = [];
  const invoke = (command,payload) => {calls.push({command,payload});return new Promise(r=>releases.push(r));};
  const root = {};
  const doc = {
    documentElement: root,
    body: {},
    querySelectorAll: s => s.startsWith('.web-tab-host') ? [host] : [],
    querySelector: () => null,
    addEventListener() {}, removeEventListener() {},
  };
  const win = {
    innerWidth:1000,innerHeight:800,__PICODE_LIVE_LAYERS__:true,__TAURI__:{core:{invoke}},
    getComputedStyle: el => el === root ? {getPropertyValue:()=> '#0e0e11'} : el.style,
    MutationObserver: class {constructor(fn){mutation=fn}observe(){}disconnect(){}},
    ResizeObserver: class {observe(){}unobserve(){}disconnect(){}},
    requestAnimationFrame: fn => { callbacks.set(++id,fn);return id; },
    cancelAnimationFrame: i => callbacks.delete(i),
    addEventListener() {},removeEventListener() {},
  };
  const frame = () => {const queued=[...callbacks.values()];callbacks.clear();return queued.map(fn=>fn());};
  const dispose = installNativeLayers({window:win,document:doc});
  const [first]=frame();
  assert.equal(calls.length,1);
  assert.equal(attrs.has('data-native-layers-ready'),false);
  rect={...pageRect,left:350};mutation();await Promise.all(frame());
  assert.equal(calls.length,1,'pending update cannot race a newer rectangle');
  releases.shift()();await Promise.resolve();
  assert.equal(calls.length,2);
  assert.equal(calls[1].payload.pages[0].left,350);
  assert.equal(attrs.has('data-native-layers-ready'),false);
  releases.shift()();await first;
  assert.equal(attrs.get('data-native-layers-ready'),'true');
  assert.equal(classes.has('picode-native-clear'),true);
  rect={...pageRect,left:400};mutation();
  const [pending]=frame();
  dispose();
  releases.shift()();await pending;
  assert.equal(classes.size,0);
  assert.equal(attrs.has('data-native-layers-ready'),false);
  mutation();assert.equal(callbacks.size,0);
});
