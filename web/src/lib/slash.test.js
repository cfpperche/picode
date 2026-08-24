import { filterSlash } from "./slash.js";
import assert from "node:assert/strict";
import { test } from "node:test";

test("slash only when leading /", () => {
  assert.equal(filterSlash("hello").length, 0);
  assert.ok(filterSlash("/").length > 3);
});

test("filters by prefix", () => {
  const hits = filterSlash("/log");
  assert.equal(hits[0].id, "login");
});
