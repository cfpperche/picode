import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { sparklinePath } from "./sparkline.js";

describe("sparklinePath", () => {
  it("returns null for fewer than 2 points", () => {
    assert.equal(sparklinePath([]), null);
    assert.equal(sparklinePath([5]), null);
  });

  it("builds a main path and a separate 2-point head path", () => {
    const p = sparklinePath([1, 2, 3, 4], { width: 40, height: 20, pad: 2 });
    assert.ok(p);
    assert.match(p.mainPath, /^M/);
    // main covers all but the last segment: 3 points -> "M ... L ... L ..."
    assert.equal(p.mainPath.split(" ").filter((s) => s.startsWith("M") || s.startsWith("L")).length, 3);
    // head covers exactly the last 2 points
    assert.equal(p.headPath.split(" ").filter((s) => s.startsWith("M") || s.startsWith("L")).length, 2);
    assert.equal(p.dot.x, 40 - 2); // last point sits at the right edge minus padding
  });

  it("does not divide by zero on a flat series", () => {
    const p = sparklinePath([3, 3, 3], { width: 30, height: 10, pad: 0 });
    assert.ok(p);
    assert.ok(Number.isFinite(p.dot.y));
    // every y must be the same level line
    const ys = [...p.mainPath.matchAll(/[ML]([\d.]+),([\d.]+)/g)].map((m) => Number(m[2]));
    for (const y of ys) assert.equal(y, ys[0]);
  });

  it("ignores non-numeric entries", () => {
    const p = sparklinePath([1, null, 2, undefined, 3]);
    assert.ok(p);
  });
});
