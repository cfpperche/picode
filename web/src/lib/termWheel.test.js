import assert from "node:assert/strict";
import { test } from "node:test";
import { altScreenWheelBytes } from "./termWheel.js";

function term(type, mouse) {
  return {
    buffer: { active: { type } },
    modes: { mouseTrackingMode: mouse },
  };
}

test("normal buffer: xterm keeps the wheel", () => {
  assert.equal(altScreenWheelBytes(term("normal", "none"), { deltaY: 120 }), null);
});

test("shift+wheel: xterm keeps the wheel", () => {
  assert.equal(altScreenWheelBytes(term("alternate", "none"), { deltaY: 120, shiftKey: true }), null);
});

test("alt-screen: PageUp / PageDown after threshold (mouse on or off)", () => {
  const t = term("alternate", "none");
  assert.equal(altScreenWheelBytes(t, { deltaY: 20 }).length, 0);
  const down = altScreenWheelBytes(t, { deltaY: 120 });
  assert.deepEqual(down, new TextEncoder().encode("\x1b[6~"));
  const t2 = term("alternate", "vt200");
  const up = altScreenWheelBytes(t2, { deltaY: -120 });
  assert.deepEqual(up, new TextEncoder().encode("\x1b[5~"));
});

test("missing term or event is a no-op", () => {
  assert.equal(altScreenWheelBytes(null, { deltaY: 120 }), null);
  assert.equal(altScreenWheelBytes(term("alternate", "none"), null), null);
});
