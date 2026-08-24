import { statusSegments } from "./statusbar.js";
import assert from "node:assert/strict";
import { test } from "node:test";

test("hides empty segments", () => {
  assert.deepEqual(statusSegments({}), []);
  assert.equal(statusSegments({ cwd: "~/picode" })[0].text, "~/picode");
});

test("git worktree dirty and extras", () => {
  const parts = statusSegments({
    cwd: "~/w",
    branch: "main",
    worktree: "hotfix",
    dirty: true,
    contextWindow: 200000,
    contextPercent: 81,
    autoCompact: true,
    input: 12000,
    output: 3000,
    cacheRead: 8000,
    cacheHit: 40,
    cost: 0.12,
    sessionName: "refactor-auth",
  });
  assert.equal(parts.find((p) => p.key === "git").text, "main@hotfix*");
  const ctx = parts.find((p) => p.key === "ctx");
  assert.equal(ctx.tone, "warn");
  assert.ok(ctx.text.includes("(auto)"));
  assert.equal(parts.find((p) => p.key === "io").text, "↑12k ↓3k");
  assert.equal(parts.find((p) => p.key === "ch").text, "CH40%");
  assert.equal(parts.find((p) => p.key === "cost").text, "$0.12");
  assert.equal(parts.find((p) => p.key === "name").text, "refactor-auth");
});
