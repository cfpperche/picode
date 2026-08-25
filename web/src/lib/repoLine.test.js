import assert from "node:assert/strict";
import { test } from "node:test";
import { repoLine } from "./repoLine.js";

test("git branch and worktree", () => {
  assert.deepEqual(repoLine({ git: { branch: "main" } }, null), { git: true, text: "main" });
  assert.deepEqual(
    repoLine({}, { git: { worktree: "hotfix", branch: "main" } }),
    { git: true, text: "hotfix / main" },
  );
});

test("not a repo is local", () => {
  assert.deepEqual(repoLine({}, { path: "/tmp/x" }), { git: false, text: "local" });
});
