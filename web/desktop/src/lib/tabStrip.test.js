import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { revealLeft } from "./tabStrip.js";

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
