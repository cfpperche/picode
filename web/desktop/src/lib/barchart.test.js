import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { barChart } from "./barchart.js";

describe("barChart", () => {
  it("is null with no values", () => {
    assert.equal(barChart([]), null);
    assert.equal(barChart(null), null);
  });
  it("scales bar height to the max and shares width evenly", () => {
    const c = barChart([1, 2, 4], { width: 30, height: 40, gap: 0 });
    assert.equal(c.max, 4);
    assert.equal(c.bars.length, 3);
    assert.deepEqual(c.bars.map((b) => b.h), [10, 20, 40]);
    assert.deepEqual(c.bars.map((b) => b.x), [0, 10, 20]);
    assert.equal(c.bars[2].y, 0);
    assert.equal(c.bars[0].y, 30);
  });
  it("renders an all-zero series as a flat baseline, not NaN", () => {
    const c = barChart([0, 0], { width: 20, height: 10, minH: 2 });
    assert.equal(c.max, 0);
    assert.deepEqual(c.bars.map((b) => b.h), [0, 0]);
    assert.ok(c.bars.every((b) => !Number.isNaN(b.y)));
  });
  it("treats non-numbers as zero and honours minH for tiny non-zero values", () => {
    const c = barChart([null, 0.001, 100], { width: 30, height: 100, gap: 0, minH: 1 });
    assert.equal(c.bars[0].h, 0);
    assert.equal(c.bars[1].h, 1);
    assert.equal(c.bars[2].h, 100);
  });
});
