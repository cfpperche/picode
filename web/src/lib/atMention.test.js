import { atQuery, insertAtPath } from "./atMention.js";
import assert from "node:assert/strict";
import { test } from "node:test";

test("atQuery needs a word-starting @", () => {
  assert.equal(atQuery("hello", 5), null);
  assert.equal(atQuery("email@x.com", 11), null);
  assert.deepEqual(atQuery("@", 1), { start: 0, query: "" });
  assert.deepEqual(atQuery("see @fo", 7), { start: 4, query: "fo" });
  assert.equal(atQuery("see @fo more", 12), null);
});

test("insertAtPath replaces the token and adds a space", () => {
  const got = insertAtPath("see @fo", 7, "src/app.go");
  assert.equal(got.text, "see @src/app.go ");
  assert.equal(got.caret, "see @src/app.go ".length);
});

test("insertAtPath quotes paths with spaces", () => {
  const got = insertAtPath("@a", 2, "my file.txt");
  assert.equal(got.text, '@"my file.txt" ');
});
