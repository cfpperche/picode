import assert from "node:assert/strict";
import { test } from "node:test";
import { readOpenTabs, writeOpenTabs, filterOpenTabs, moveTab } from "./openTabs.js";

test("filterOpenTabs drops missing agents", () => {
  const got = filterOpenTabs(
    { ids: ["a", "gone", "b"], selected: "gone" },
    (id) => id === "a" || id === "b",
  );
  assert.deepEqual(got.ids, ["a", "b"]);
  assert.equal(got.selected, "a");
});

test("moveTab reorders and no-ops on bad ids", () => {
  assert.deepEqual(moveTab(["a", "b", "c"], "a", "c"), ["b", "c", "a"]);
  assert.deepEqual(moveTab(["a", "b", "c"], "c", "a"), ["c", "a", "b"]);
  assert.deepEqual(moveTab(["a", "b"], "a", "a"), ["a", "b"]);
  assert.deepEqual(moveTab(["a", "b"], "z", "a"), ["a", "b"]);
});

test("roundtrip", () => {
  const store = {};
  globalThis.localStorage = {
    getItem: (k) => (k in store ? store[k] : null),
    setItem: (k, v) => { store[k] = String(v); },
  };
  writeOpenTabs(["x", "y"], "y");
  const got = readOpenTabs();
  assert.deepEqual(got.ids, ["x", "y"]);
  assert.equal(got.selected, "y");
});
