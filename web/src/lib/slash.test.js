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

test("/scoped-models opens settings", () => {
  const hits = filterSlash("/scoped");
  assert.equal(hits[0].id, "scoped-models");
  assert.equal(hits[0].run, "go-scoped");
});

test("/settings is a PiCode route, not a TUI proxy", () => {
  const hits = filterSlash("/settings");
  assert.equal(hits[0].id, "settings");
  assert.equal(hits[0].run, "go-settings");
});
