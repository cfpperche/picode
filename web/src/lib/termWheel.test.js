import assert from "node:assert/strict";
import { test } from "node:test";
import { applyTermWheel, pageBytesFor, wheelLineCount } from "./termWheel.js";

function fakeTerm(viewportY, maxY) {
  const cap = maxY == null ? 1000 : maxY;
  const buf = { viewportY };
  return {
    buffer: { active: buf },
    scrollLines(n) {
      buf.viewportY = Math.max(0, Math.min(cap, buf.viewportY + n));
    },
  };
}

test("wheelLineCount: pixels become lines, up is negative", () => {
  assert.equal(wheelLineCount({ deltaY: -120 }), -3);
  assert.equal(wheelLineCount({ deltaY: 40 }), 1);
  assert.equal(wheelLineCount({ deltaY: 0 }), 0);
  assert.equal(wheelLineCount({ deltaY: -2, deltaMode: 1 }), -2);
});

test("xterm scrollback: wheel moves viewport, no PageUp", () => {
  const sent = [];
  const t = fakeTerm(20);
  assert.equal(applyTermWheel(t, { deltaY: -120 }, (b) => sent.push(b)), "xterm");
  assert.equal(t.buffer.active.viewportY, 17);
  assert.equal(sent.length, 0);
});

test("no scrollback: PageUp / PageDown after threshold", () => {
  const sent = [];
  const t = fakeTerm(0, 0);
  assert.equal(applyTermWheel(t, { deltaY: -20 }, (b) => sent.push(b)), "hold");
  assert.equal(sent.length, 0);
  assert.equal(applyTermWheel(t, { deltaY: -120 }, (b) => sent.push(b)), "page");
  assert.deepEqual(sent[0], new TextEncoder().encode("\x1b[5~"));
  const t2 = fakeTerm(0, 0);
  applyTermWheel(t2, { deltaY: 120 }, (b) => sent.push(b));
  assert.deepEqual(sent[1], new TextEncoder().encode("\x1b[6~"));
});

test("shift+wheel and empty event are skipped", () => {
  const t = fakeTerm(8);
  assert.equal(applyTermWheel(t, { deltaY: -120, shiftKey: true }), "skip");
  assert.equal(t.buffer.active.viewportY, 8);
  assert.equal(applyTermWheel(t, null), "skip");
  assert.equal(applyTermWheel(null, { deltaY: -120 }), "skip");
});

test("pageBytesFor accumulates small deltas", () => {
  const t = {};
  assert.equal(pageBytesFor(t, -10).length, 0);
  assert.deepEqual(pageBytesFor(t, -40), new TextEncoder().encode("\x1b[5~"));
});
