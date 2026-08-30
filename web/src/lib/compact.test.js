import { describe, test } from "node:test";
import assert from "node:assert/strict";
import { readCompacting, writeCompacting } from "./compact.js";

function fakeStore() {
  const m = new Map();
  return {
    getItem: (k) => (m.has(k) ? m.get(k) : null),
    setItem: (k, v) => m.set(k, String(v)),
  };
}

describe("compact persistence", () => {
  test("round-trips a map and drops stale entries", () => {
    const s = fakeStore();
    writeCompacting({ "a-1": Date.now() - 1000, "b-2": Date.now() - 45 * 60 * 1000 }, s);
    const got = readCompacting(s);
    assert.deepEqual(Object.keys(got), ["a-1"]);
  });

  test("bad JSON reads as empty", () => {
    const s = fakeStore();
    s.setItem("picode-compacting", "{nope");
    assert.deepEqual(readCompacting(s), {});
  });

  test("null store reads as empty and write is a no-op", () => {
    assert.deepEqual(readCompacting(null), {});
    writeCompacting({ "c-3": Date.now() }, null);
  });

  test("write then clear", () => {
    const s = fakeStore();
    writeCompacting({ "c-3": Date.now() }, s);
    assert.ok(readCompacting(s)["c-3"] > 0);
    writeCompacting({}, s);
    assert.deepEqual(readCompacting(s), {});
  });
});
