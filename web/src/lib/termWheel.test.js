import assert from "node:assert/strict";
import { test } from "node:test";
import { applyTermWheel, sgrWheelBytes, wheelLineCount } from "./termWheel.js";

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
});

test("xterm scrollback: wheel moves viewport, no SGR", () => {
  const sent = [];
  const t = fakeTerm(20);
  assert.equal(applyTermWheel(t, { deltaY: -120 }, (b) => sent.push(b)), "xterm");
  assert.equal(t.buffer.active.viewportY, 17);
  assert.equal(sent.length, 0);
});

test("no scrollback: SGR wheel at row 2 after threshold", () => {
  const sent = [];
  const t = fakeTerm(0, 0);
  assert.equal(applyTermWheel(t, { deltaY: -10 }, (b) => sent.push(b)), "hold");
  assert.equal(applyTermWheel(t, { deltaY: -120 }, (b) => sent.push(b)), "sgr");
  assert.deepEqual(sent[0], sgrWheelBytes(-130));
});

test("sgrWheelBytes: up is button 64, down is 65", () => {
  const up = new TextDecoder().decode(sgrWheelBytes(-40));
  const down = new TextDecoder().decode(sgrWheelBytes(80));
  assert.equal(up, "\x1b[<64;2;2M");
  assert.equal(down, "\x1b[<65;2;2M\x1b[<65;2;2M");
});

test("shift+wheel and empty event are skipped", () => {
  const t = fakeTerm(8);
  assert.equal(applyTermWheel(t, { deltaY: -120, shiftKey: true }), "skip");
  assert.equal(t.buffer.active.viewportY, 8);
  assert.equal(applyTermWheel(null, { deltaY: -120 }), "skip");
});
