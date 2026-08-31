import assert from "node:assert/strict";
import { test } from "node:test";
import { matchCommits, MIN_QUERY } from "./gitgraphSearch.js";

const commits = [
  { hash: "aaa111", subject: "fix: the roles form", author: "Ana" },
  { hash: "bbb222", subject: "feat(gg): search", author: "Bruno" },
  { hash: "abc333", subject: "docs: notes", author: "ana maria" },
];

test("subject and author match case-insensitively as substrings", () => {
  assert.deepEqual([...matchCommits(commits, "ROLES")], ["aaa111"]);
  assert.deepEqual([...matchCommits(commits, "ana")], ["aaa111", "abc333"]);
  assert.deepEqual([...matchCommits(commits, "feat(gg)")], ["bbb222"]);
});

test("a hash matches by prefix, not by substring", () => {
  assert.deepEqual([...matchCommits(commits, "ab")], ["abc333"]);
  assert.equal(matchCommits(commits, "b222").size, 0);
});

test("a query below the minimum matches nothing", () => {
  assert.ok(MIN_QUERY >= 2);
  assert.equal(matchCommits(commits, "a").size, 0);
  assert.equal(matchCommits(commits, "").size, 0);
  assert.equal(matchCommits(commits, "   ").size, 0);
});

test("empty input is safe", () => {
  assert.equal(matchCommits(null, "abc").size, 0);
  assert.equal(matchCommits([], "abc").size, 0);
});
