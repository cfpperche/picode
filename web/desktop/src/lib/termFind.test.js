import { test } from "node:test";
import assert from "node:assert/strict";
import { findLabel, findKey, findMode, setFindMode, findProblem, modeChanged } from "./termFind.js";

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

test("the search mode is remembered, one flag at a time", () => {
  assert.deepEqual(findMode(), { caseSensitive: false, wholeWord: false, regex: false });
  assert.deepEqual(setFindMode({ regex: true }), { caseSensitive: false, wholeWord: false, regex: true });
  assert.deepEqual(setFindMode({ caseSensitive: true }), { caseSensitive: true, wholeWord: false, regex: true });
  assert.equal(setFindMode({ wholeWord: true }).wholeWord, true);
  setFindMode({ bogus: true });
  assert.deepEqual(findMode(), { caseSensitive: true, wholeWord: true, regex: true });
  // the getter hands out a copy, so nobody edits the mode by accident
  const copy = findMode();
  copy.regex = false;
  assert.equal(findMode().regex, true);
  setFindMode({ caseSensitive: false, wholeWord: false, regex: false });
});

test("a mode change is what forces the addon to scan again", () => {
  const off = { caseSensitive: false, wholeWord: false, regex: false };
  assert.equal(modeChanged(off, { ...off }), false);
  assert.equal(modeChanged(off, { ...off, wholeWord: true }), true);
  assert.equal(modeChanged(off, { ...off, regex: true }), true);
  // absent is off, not different
  assert.equal(modeChanged({}, off), false);
});

test("a half-typed pattern is named, and only while regex is on", () => {
  assert.equal(findProblem("alpha [", true), "Invalid pattern");
  assert.equal(findProblem("alpha [", false), "");
  assert.equal(findProblem("", true), "");
  assert.equal(findProblem("alpha \\d+", true), "");
});
