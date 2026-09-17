import { overlayAudit } from "./overlayAudit.js";
import assert from "node:assert/strict";
import { test } from "node:test";

function fakeWin({ top, bottom, left, right, display = "block", rows = [] }) {
  const el = {
    getBoundingClientRect: () => ({ top, bottom, left, right, width: right - left, height: bottom - top }),
  };
  return {
    innerHeight: 800,
    innerWidth: 1200,
    document: {
      querySelectorAll: (sel) => {
        if (sel === ".cockpit-pop") return [el];
        if (sel === "[data-align-row]") return rows;
        return [];
      },
    },
    getComputedStyle: () => ({ display, visibility: "visible" }),
  };
}

test("in-viewport overlay is ok", () => {
  const r = overlayAudit(fakeWin({ top: 40, bottom: 200, left: 10, right: 300 }));
  assert.equal(r.ok, true);
  assert.equal(r.hits[0].clipTop, false);
});

test("uneven align-row is fail", () => {
  const row = {
    children: [
      { getBoundingClientRect: () => ({ top: 10, height: 32, width: 200 }) },
      { getBoundingClientRect: () => ({ top: 8, height: 40, width: 80 }) },
    ],
  };
  const r = overlayAudit(fakeWin({ top: 40, bottom: 200, left: 10, right: 300, rows: [row] }));
  assert.equal(r.ok, false);
  assert.equal(r.rows[0].misaligned, true);
});

test("clipped top is fail", () => {
  const r = overlayAudit(fakeWin({ top: -40, bottom: 80, left: 10, right: 300 }));
  assert.equal(r.ok, false);
  assert.equal(r.hits[0].clipTop, true);
});

// The class the owner hit on 2026-09-16: an HTML layer over the native work
// browser is invisible until the page gets out of the way. The tab marks its
// host when it hides the view; a layer over an unmarked host is a defect the
// review must catch by itself.
function browserWin({ covered }) {
  const host = {
    getBoundingClientRect: () => ({ top: 80, left: 300, right: 1100, bottom: 700, width: 800, height: 620 }),
    getAttribute: (name) => (name === "data-covered" && covered ? "1" : null),
  };
  const layer = {
    getBoundingClientRect: () => ({ top: 40, left: 700, right: 1090, bottom: 300, width: 390, height: 260 }),
  };
  return {
    innerHeight: 800,
    innerWidth: 1200,
    document: {
      querySelectorAll: (sel) => {
        if (sel === ".web-tab-host") return [host];
        if (sel === ".cockpit-pop") return [layer];
        return [];
      },
    },
    getComputedStyle: () => ({ display: "block", visibility: "visible" }),
  };
}

test("a layer over the work browser with the page still up is fail", () => {
  const r = overlayAudit(browserWin({ covered: false }));
  assert.equal(r.ok, false);
  assert.equal(r.uncovered.length, 1);
});

test("that same layer with the page parked is ok", () => {
  const r = overlayAudit(browserWin({ covered: true }));
  assert.equal(r.ok, true);
  assert.deepEqual(r.uncovered, []);
});

for (const [name, wraps, rects, ok] of [
  ["aligned controls", false, [[10, 36], [10, 36]], true],
  ["undeclared wrap", false, [[10, 36], [52, 36]], false],
  ["declared wrap", true, [[10, 36], [10, 36], [52, 36]], true],
  ["uneven height after wrap", true, [[10, 36], [52, 32]], false],
  ["misaligned controls on the same wrapped line", true, [[10, 36], [12, 36], [52, 36]], false],
  ["misaligned controls on a later wrapped line", true, [[10, 36], [52, 36], [54, 36]], false],
]) test(name, () => {
  const row = {
    hasAttribute: name => name === "data-align-wrap" && wraps,
    children: rects.map(([top, height]) => ({ getBoundingClientRect: () => ({ top, height, width: 100 }) })),
  };
  const report = overlayAudit(fakeWin({ top: 40, bottom: 200, left: 10, right: 300, rows: [row] }));
  assert.equal(report.ok, ok);
});
