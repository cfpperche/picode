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
