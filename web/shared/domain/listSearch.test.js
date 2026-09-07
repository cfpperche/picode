import test from "node:test";
import assert from "node:assert/strict";
import { matchesListSearch } from "./listSearch.js";

test("matches all query words across fields, ignoring case and accents", () => {
  assert.equal(matchesListSearch("CAFE sonnet", "Café", "Sonnet"), true);
  assert.equal(matchesListSearch("cafe missing", "Café", "Sonnet"), false);
  assert.equal(matchesListSearch("pi model", "Settings", "Pi: model, thinking, prompt", "settings"), true);
});

test("blank query matches everything; whitespace-only words are dropped", () => {
  assert.equal(matchesListSearch("", "Anything"), true);
  assert.equal(matchesListSearch("   ", "Anything"), true);
  assert.equal(matchesListSearch(undefined, "Anything"), true);
});

test("word order does not matter", () => {
  assert.equal(matchesListSearch("prompt model", "Settings", "Pi: model, thinking, prompt"), true);
});
