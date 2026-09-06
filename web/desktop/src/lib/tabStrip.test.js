import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { revealLeft, stripState, wheelToScroll, arrowStep } from "./tabStrip.js";

const strip = { scrollLeft: 0, clientWidth: 995, scrollWidth: 1049 };

describe("revealLeft", () => {
  it("leaves a fully visible tab alone", () => {
    assert.equal(revealLeft(strip, { left: 100, width: 178 }), null);
  });
  it("aligns a tab clipped on the right to the right edge", () => {
    // The owner's screenshot: active tab starts past the visible width.
    assert.equal(revealLeft(strip, { left: 916, width: 133 }), 54);
  });
  it("aligns a tab clipped on the left to the left edge", () => {
    assert.equal(revealLeft({ ...strip, scrollLeft: 300, scrollWidth: 2000 }, { left: 198, width: 178 }), 198);
  });
  it("never scrolls past the content", () => {
    assert.equal(revealLeft(strip, { left: 1000, width: 400 }), 54);
  });
  it("aligns the start of a tab wider than the strip", () => {
    const narrow = { scrollLeft: 0, clientWidth: 100, scrollWidth: 600 };
    assert.equal(revealLeft(narrow, { left: 250, width: 200 }), 250);
    assert.equal(revealLeft({ ...narrow, scrollLeft: 250 }, { left: 250, width: 200 }), null);
  });
  it("does nothing when the strip does not overflow", () => {
    assert.equal(revealLeft({ scrollLeft: 0, clientWidth: 995, scrollWidth: 600 }, { left: 500, width: 100 }), null);
  });
});

describe("stripState", () => {
  it("reports no overflow when content fits (with a 1px tolerance)", () => {
    const s = stripState({ scrollLeft: 0, clientWidth: 995, scrollWidth: 995.5 });
    assert.deepEqual(s, { overflow: false, atStart: true, atEnd: true, thumb: { left: 0, width: 1 } });
  });
  it("marks the start, the middle and the end", () => {
    const box = { clientWidth: 500, scrollWidth: 1000 };
    assert.deepEqual(stripState({ ...box, scrollLeft: 0 }).atStart, true);
    assert.deepEqual(stripState({ ...box, scrollLeft: 0 }).atEnd, false);
    const mid = stripState({ ...box, scrollLeft: 250 });
    assert.equal(mid.atStart, false);
    assert.equal(mid.atEnd, false);
    assert.deepEqual(mid.thumb, { left: 0.25, width: 0.5 });
    assert.equal(stripState({ ...box, scrollLeft: 499.5 }).atEnd, true);
  });
});

describe("wheelToScroll", () => {
  it("turns a vertical wheel into horizontal pixels", () => {
    assert.equal(wheelToScroll({ deltaY: 120 }, 995), 120);
    assert.equal(wheelToScroll({ deltaY: -120 }, 995), -120);
  });
  it("leaves trackpad horizontal gestures and pinches to the browser", () => {
    assert.equal(wheelToScroll({ deltaX: 30, deltaY: 2 }, 995), 0);
    assert.equal(wheelToScroll({ deltaX: -5, deltaY: 120 }, 995), 0);
    assert.equal(wheelToScroll({ deltaY: 120, ctrlKey: true }, 995), 0);
    assert.equal(wheelToScroll({ deltaY: 0 }, 995), 0);
  });
  it("scales line and page deltas", () => {
    assert.equal(wheelToScroll({ deltaY: 3, deltaMode: 1 }, 995), 120);
    assert.equal(wheelToScroll({ deltaY: 1, deltaMode: 2 }, 995), 995);
  });
});

describe("arrowStep", () => {
  it("moves most of a viewport, never less than a tab", () => {
    assert.equal(arrowStep(995), 597);
    assert.equal(arrowStep(100), 80);
  });
});
