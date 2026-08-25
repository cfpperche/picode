import assert from "node:assert/strict";
import { test } from "node:test";
import { repoLine, shortPath } from "./repoLine.js";

test("git branch and worktree", () => {
  assert.deepEqual(repoLine({ git: { branch: "main" } }, null), { git: true, text: "main" });
  assert.deepEqual(
    repoLine({}, { git: { worktree: "hotfix", branch: "main" } }),
    { git: true, text: "hotfix / main" },
  );
});

test("not a repo shows the dir", () => {
  assert.deepEqual(
    repoLine({ workPath: "/home/goat/.picode/work/grok" }, null),
    { git: false, text: "~/.picode/work/grok" },
  );
  assert.deepEqual(
    repoLine({}, { path: "/home/goat/picode" }),
    { git: false, text: "~/picode" },
  );
});

test("shortPath", () => {
  assert.equal(shortPath(""), "—");
  assert.equal(shortPath("/tmp/x"), "/tmp/x");
});
