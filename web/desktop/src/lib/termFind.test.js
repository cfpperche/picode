import { test } from "node:test";
import assert from "node:assert/strict";
import { findLabel, findKey } from "./termFind.js";

test("the counter stays quiet until there is a query", () => {
  assert.equal(findLabel("", 0, -1), "");
  assert.equal(findLabel("", 12, 3), "");
});

test("a query with no match says so in words", () => {
  assert.equal(findLabel("nope", 0, -1), "No results");
});

test("matches count from one, and a search with no active match shows the total", () => {
  assert.equal(findLabel("go", 17, 2), "3/17");
  assert.equal(findLabel("go", 17, 0), "1/17");
  assert.equal(findLabel("go", 17, -1), "17");
});

test("the field owns Enter, Shift+Enter and Escape, and nothing else", () => {
  assert.equal(findKey({ key: "Enter" }), "next");
  assert.equal(findKey({ key: "Enter", shiftKey: true }), "prev");
  assert.equal(findKey({ key: "Escape" }), "close");
  assert.equal(findKey({ key: "a" }), null);
  assert.equal(findKey({ key: "Tab", shiftKey: true }), null);
  assert.equal(findKey(null), null);
});
