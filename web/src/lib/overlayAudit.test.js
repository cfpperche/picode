import { overlayAudit } from "./overlayAudit.js";
import assert from "node:assert/strict";
import { test } from "node:test";

function fakeWin({ top, bottom, left, right, display = "block" }) {
  const el = {
    getBoundingClientRect: () => ({ top, bottom, left, right, width: right - left, height: bottom - top }),
  };
  return {
    innerHeight: 800,
    innerWidth: 1200,
    document: { querySelectorAll: (sel) => (sel === ".cockpit-pop" ? [el] : []) },
    getComputedStyle: () => ({ display, visibility: "visible" }),
  };
}

test("in-viewport overlay is ok", () => {
  const r = overlayAudit(fakeWin({ top: 40, bottom: 200, left: 10, right: 300 }));
  assert.equal(r.ok, true);
  assert.equal(r.hits[0].clipTop, false);
});

test("clipped top is fail", () => {
  const r = overlayAudit(fakeWin({ top: -40, bottom: 80, left: 10, right: 300 }));
  assert.equal(r.ok, false);
  assert.equal(r.hits[0].clipTop, true);
});
