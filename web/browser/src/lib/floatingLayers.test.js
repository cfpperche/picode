import { test } from "node:test";
import assert from "node:assert/strict";
import { layerRects, overlaps, overlapsLayers, rectOf } from "./floatingLayers.js";

// The rule the work browser hides by. Every row here is a shape the app
// actually produces (a menu under the tab strip, a toast in the corner, a
// sidebar parked off-screen) — a wrong answer means either a covered overlay
// or a page that flickers for nothing.

function el({ top = 0, left = 0, width = 100, height = 100, display = "block", visibility = "visible" } = {}) {
  return {
    getBoundingClientRect: () => ({ top, left, width, height, right: left + width, bottom: top + height }),
    __style: { display, visibility },
  };
}

function fakeWin(els = [], { w = 1200, h = 800 } = {}) {
  return {
    innerWidth: w,
    innerHeight: h,
    document: {
      body: {},
      documentElement: {},
      // One selector's worth of layers is enough to test the rule; the real
      // list is exercised by the app itself.
      querySelectorAll: (sel) => (sel === "[data-radix-popper-content-wrapper]" ? els : []),
    },
    getComputedStyle: (node) => node.__style || { display: "block", visibility: "visible" },
  };
}

test("a visible layer becomes a rect; a hidden one does not", () => {
  const win = fakeWin([el({ top: 10, left: 10, width: 200, height: 100 })]);
  assert.equal(layerRects(win.document, win).length, 1);

  const hidden = fakeWin([el({ display: "none" })]);
  assert.deepEqual(layerRects(hidden.document, hidden), []);
  const invisible = fakeWin([el({ visibility: "hidden" })]);
  assert.deepEqual(layerRects(invisible.document, invisible), []);
});

test("degenerate rects are not layers", () => {
  const win = fakeWin([el({ width: 0, height: 40 }), el({ width: 120, height: 0 })]);
  assert.deepEqual(layerRects(win.document, win), []);
});

test("chrome parked outside the viewport is not over the page", () => {
  // The focus-mode sidebar waits off the left edge; a menu animates in from
  // below. Neither should hide the page.
  const win = fakeWin([el({ top: 40, left: -300, width: 280, height: 700 }), el({ top: 900, left: 40, width: 200, height: 120 })]);
  assert.deepEqual(layerRects(win.document, win), []);
});

test("overlap is strict: touching edges do not count", () => {
  const page = { left: 100, top: 100, right: 600, bottom: 500 };
  // Sharing an edge is not covering: a menu that ends exactly where the page
  // begins must not hide it.
  assert.equal(overlaps(page, { left: 0, top: 100, right: 100, bottom: 500 }), false);
  assert.equal(overlaps(page, { left: 100, top: 0, right: 600, bottom: 100 }), false);
  // One pixel in, and it is over the page.
  assert.equal(overlaps(page, { left: 0, top: 100, right: 101, bottom: 500 }), true);
  assert.equal(overlaps(page, { left: 100, top: 0, right: 600, bottom: 101 }), true);
});

test("the tab-strip menu over the page is exactly the reported bug", () => {
  // The tab strip's dropdown hangs below the strip, into the browser region:
  // this must be true or the menu stays invisible.
  const menu = { left: 900, top: 36, right: 1180, bottom: 240 };
  const page = { left: 320, top: 40, right: 1200, bottom: 780 };
  assert.equal(overlaps(page, menu), true);
  assert.equal(overlapsLayers(page, [menu]), true);
});

test("a menu in the sidebar does not touch the page", () => {
  const sidebarMenu = { left: 8, top: 200, right: 260, bottom: 400 };
  const page = { left: 320, top: 40, right: 1200, bottom: 780 };
  assert.equal(overlapsLayers(page, [sidebarMenu]), false);
});

test("a dialog the class list did not know is still a layer", () => {
  // The command palette wears `.palette`, not the app's `.dlg`: role+state is
  // what catches it.
  const win = fakeWin([el({ top: 89, left: 380, width: 520, height: 366 })]);
  const layers = layerRects(win.document, win);
  assert.equal(layers.length, 1);
  assert.equal(overlapsLayers({ left: 244, top: 88, right: 1280, bottom: 633 }, layers), true);
});

test("no layers means not covered", () => {
  assert.equal(overlapsLayers({ left: 0, top: 0, right: 10, bottom: 10 }, []), false);
  assert.equal(overlapsLayers(null, [{ left: 0, top: 0, right: 10, bottom: 10 }]), false);
});

test("rectOf tolerates a missing node", () => {
  assert.equal(rectOf(null), null);
  assert.equal(rectOf({}), null);
});
