import test from "node:test";
import assert from "node:assert/strict";
import { wireTermTouch } from "./termWheel.js";

function fixture(t) {
  t.mock.method(globalThis, "Event", Event);
  const previous = globalThis.WheelEvent;
  globalThis.WheelEvent = class extends Event { constructor(type, opts) { super(type, opts); this.deltaY = opts.deltaY; this.clientX = opts.clientX; this.clientY = opts.clientY; } };
  t.after(() => { if (previous) globalThis.WheelEvent = previous; else delete globalThis.WheelEvent; });
  const listeners = new Map(), wheels = [];
  const positions = [];
  const target = { dispatchEvent: e => { wheels.push(e.deltaY); positions.push([e.clientX, e.clientY]); } };
  const host = {
    querySelector: () => target,
    addEventListener: (type, fn) => listeners.set(type, fn),
    removeEventListener: type => listeners.delete(type),
  };
  const unwire = wireTermTouch(host);
  let prevented = 0;
  const touch = (type, ...ys) => listeners.get(type)?.({ touches: ys.map(clientY => ({ clientY, clientX: 120 })), preventDefault() { prevented++; } });
  return { touch, wheels, positions, listeners, unwire, prevented: () => prevented };
}
test("one finger accumulates both directions into the Agent CLI wheel contract", t => {
  const f = fixture(t);
  f.touch("touchstart", 100); f.touch("touchmove", 92);
  assert.deepEqual(f.wheels, []);
  f.touch("touchmove", 68);
  assert.deepEqual(f.wheels, [40, 40]);
  assert.deepEqual(f.positions, [[120, 68], [120, 68]], "mouse-reporting TUIs receive the touched pane coordinates");
  f.touch("touchmove", 100);
  assert.deepEqual(f.wheels, [40, 40, -40, -40]);
  assert.equal(f.prevented(), 3);
});
test("pinch, cancellation and disposal cannot leave a scrolling gesture behind", t => {
  const f = fixture(t);
  f.touch("touchstart", 100); f.touch("touchmove", 90, 80); f.touch("touchmove", 40);
  f.touch("touchstart", 100); f.touch("touchcancel"); f.touch("touchmove", 40);
  assert.deepEqual(f.wheels, []);
  assert.equal(f.prevented(), 0);
  f.unwire();
  assert.equal(f.listeners.size, 0);
});
