import assert from "node:assert/strict";
import { test } from "node:test";
import { applyTermSlack, sendTermResize, termSlackPad } from "./termFit.js";

test("sendTermResize no-ops until the socket is open and the term has a size", () => {
  assert.equal(sendTermResize(null), false);
  assert.equal(sendTermResize({}), false);
  const sent = [];
  const sock = { readyState: 0, send: (s) => sent.push(s) };
  assert.equal(sendTermResize({ term: { cols: 80, rows: 24 }, sock }), false);
  sock.readyState = 1;
  assert.equal(sendTermResize({ term: { cols: 1, rows: 24 }, sock }), false);
  assert.equal(sendTermResize({ term: { cols: 80, rows: 24 }, sock }), true);
  assert.equal(sent[0], JSON.stringify({ type: "resize", cols: 80, rows: 24 }));
});

test("termSlackPad splits leftover pixels so both sides match", () => {
  assert.deepEqual(termSlackPad(1000, 1000), { slack: 0, left: 0, right: 0 });
  assert.deepEqual(termSlackPad(1000, 991), { slack: 9, left: 4, right: 5 });
  assert.deepEqual(termSlackPad(1000, 992), { slack: 8, left: 4, right: 4 });
  assert.deepEqual(termSlackPad(1000.4, 991.6), { slack: 8, left: 4, right: 4 });
  assert.deepEqual(termSlackPad(80, 81), { slack: 0, left: 0, right: 0 });
  assert.deepEqual(termSlackPad(0, 80), { slack: 0, left: 0, right: 0 });
  assert.deepEqual(termSlackPad(NaN, 80), { slack: 0, left: 0, right: 0 });
});

test("applyTermSlack no-ops without an opened xterm", () => {
  assert.deepEqual(applyTermSlack(null), { slack: 0, left: 0, right: 0 });
  assert.deepEqual(applyTermSlack({}), { slack: 0, left: 0, right: 0 });
  assert.deepEqual(applyTermSlack({ element: { querySelector: () => null } }), { slack: 0, left: 0, right: 0 });
});

test("applyTermSlack writes the split onto the screen", () => {
  const screen = { offsetWidth: 991, style: {} };
  const term = {
    element: {
      clientWidth: 1000,
      querySelector: (sel) => (sel === ".xterm-screen" ? screen : null),
    },
  };
  assert.deepEqual(applyTermSlack(term), { slack: 9, left: 4, right: 5 });
  assert.equal(screen.style.marginLeft, "4px");
  assert.equal(screen.style.marginRight, "5px");
});
