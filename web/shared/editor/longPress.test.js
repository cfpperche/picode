import assert from "node:assert/strict";
import { test } from "node:test";
import { HOLD_MS, pressTracker } from "./longPress.js";

// A clock the test drives: timers fire only when `advance` passes them.
function clock() {
  let now = 0;
  let seq = 0;
  const timers = new Map();
  return {
    setTimer: (fn, ms) => { timers.set(++seq, { fn, at: now + ms }); return seq; },
    clearTimer: (id) => { timers.delete(id); },
    advance(ms) {
      now += ms;
      for (const [id, t] of [...timers]) if (t.at <= now) { timers.delete(id); t.fn(); }
    },
  };
}

test("pressTracker", () => {
  const cases = [
    { name: "hold opens", steps: (p, c) => { p.start(10, 10, "a.md"); c.advance(HOLD_MS); }, opened: ["a.md"], end: "opened" },
    { name: "short press is a tap", steps: (p, c) => { p.start(10, 10, "a.md"); c.advance(HOLD_MS - 100); }, opened: [], end: "tap" },
    { name: "small jitter still holds", steps: (p, c) => { p.start(10, 10, "a.md"); p.move(16, 14); c.advance(HOLD_MS); }, opened: ["a.md"], end: "opened" },
    { name: "scrolling cancels", steps: (p, c) => { p.start(10, 10, "a.md"); p.move(10, 40); c.advance(HOLD_MS); }, opened: [], end: "none" },
    { name: "no link, nothing", steps: (p, c) => { p.start(10, 10, ""); c.advance(HOLD_MS); }, opened: [], end: "none" },
    { name: "cancel stops the timer", steps: (p, c) => { p.start(10, 10, "a.md"); p.cancel(); c.advance(HOLD_MS); }, opened: [], end: "none" },
  ];
  for (const k of cases) {
    const c = clock();
    const opened = [];
    const p = pressTracker({ open: (h) => opened.push(h), setTimer: c.setTimer, clearTimer: c.clearTimer });
    k.steps(p, c);
    assert.deepEqual(opened, k.opened, k.name);
    assert.equal(p.end(), k.end, k.name);
    assert.equal(p.end(), "none", k.name + " (reset)");
  }
});

test("pressTracker refuses the context menu only around its own press", () => {
  const c = clock();
  const p = pressTracker({ open() {}, setTimer: c.setTimer, clearTimer: c.clearTimer });
  assert.equal(p.holding(), false);
  p.start(0, 0, "x");
  assert.equal(p.holding(), true);
  c.advance(HOLD_MS);
  assert.equal(p.holding(), true);
  p.end();
  assert.equal(p.holding(), false);
});
