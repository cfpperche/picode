import assert from "node:assert/strict";
import { test } from "node:test";
import { applyTermWheel, sgrWheelBytes } from "./termWheel.js";

function fakeTerm(extra) {
  const o = extra || {};
  return {
    rows: 24,
    modes: { mouseTrackingMode: o.mouse || "none" },
    buffer: {
      active: {
        type: o.type || "normal",
        viewportY: o.y || 0,
        baseY: o.baseY || 0,
        length: o.length || 24,
      },
    },
  };
}

test("mouse tracking on: let xterm send SGR (issue 426)", () => {
  const sent = [];
  assert.equal(applyTermWheel(fakeTerm({ mouse: "vt200", type: "alternate" }), { deltaY: -120 }, (b) => sent.push(b)), "skip");
  assert.equal(sent.length, 0);
});

test("normal buffer with scrollback: let xterm scroll", () => {
  const sent = [];
  assert.equal(applyTermWheel(fakeTerm({ baseY: 40, length: 64 }), { deltaY: -120 }, (b) => sent.push(b)), "skip");
  assert.equal(sent.length, 0);
});

test("alt-screen without mouse: SGR at row 2, not arrows", () => {
  const sent = [];
  const t = fakeTerm({ type: "alternate" });
  assert.equal(applyTermWheel(t, { deltaY: -10 }, (b) => sent.push(b)), "hold");
  assert.equal(applyTermWheel(t, { deltaY: -120 }, (b) => sent.push(b)), "sgr");
  assert.deepEqual(sent[0], sgrWheelBytes(-130));
});

test("sgrWheelBytes: up is button 64, down is 65", () => {
  assert.equal(new TextDecoder().decode(sgrWheelBytes(-40)), "\x1b[<64;2;2M");
  assert.equal(new TextDecoder().decode(sgrWheelBytes(80)), "\x1b[<65;2;2M\x1b[<65;2;2M");
});

test("shift+wheel is left to xterm", () => {
  assert.equal(applyTermWheel(fakeTerm({ type: "alternate" }), { deltaY: -120, shiftKey: true }), "skip");
});
