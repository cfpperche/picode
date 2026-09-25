import assert from "node:assert/strict";
import { test } from "node:test";
import { lineFor, previewTopFor, syncBlocks } from "./scrollSync.js";

const blocks = syncBlocks([
  { line: 5, top: 100, height: 40 },
  { line: 1, top: 0, height: 60 },
  { line: 10, top: 300, height: 200 },
  { line: 10, top: 320, height: 10 }, // nested, same line: dropped
  { line: 7, top: 400, height: 10 },  // runs backwards: dropped
]);

test("syncBlocks orders and keeps lines moving forward", () => {
  assert.deepEqual(blocks.map((b) => b.line), [1, 5, 10]);
});

test("previewTopFor", () => {
  const cases = [
    [1, 0],
    [3, 50],     // halfway between line 1 (0) and line 5 (100)
    [5, 100],
    [7.5, 200],  // halfway between 5 (100) and 10 (300)
    [10, 300],
    [10.5, 400], // inside the last block
    [99, 500],   // clamped to the last block's end
  ];
  for (const [line, want] of cases) assert.equal(previewTopFor(blocks, line), want, `line ${line}`);
  assert.equal(previewTopFor([], 4), 0);
  assert.equal(previewTopFor(syncBlocks([{ line: 3, top: 40, height: 10 }]), 2), 20);
});

test("lineFor inverts previewTopFor between blocks", () => {
  for (const line of [1, 2, 3, 5, 6.5, 9, 10]) {
    assert.ok(Math.abs(lineFor(blocks, previewTopFor(blocks, line)) - line) < 1e-9, `line ${line}`);
  }
  assert.equal(lineFor(blocks, 400, 30), 10.5);
  assert.equal(lineFor(blocks, 10_000, 30), 11);
  assert.equal(lineFor([], 50), 1);
  assert.equal(lineFor(syncBlocks([{ line: 3, top: 40, height: 10 }]), 20), 2);
});
