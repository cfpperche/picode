import { test } from "node:test";
import assert from "node:assert/strict";
import { forkRequest, forkSlug, forkWorktreePath } from "./forkAgent.js";

test("forkSlug is one safe path segment named after the fork", () => {
  assert.equal(forkSlug("Fix the race!"), "fix-the-race");
  assert.equal(forkSlug("Corrigir a falha de sessão"), "corrigir-a-falha-de-sessao");
  assert.equal(forkSlug("   "), "fork");
  assert.equal(forkSlug("../../etc"), "etc");
  assert.ok(forkSlug("x".repeat(90)).length <= 48);
});

test("forkSlug steps past a taken folder or branch", () => {
  const graph = {
    worktrees: [{ path: "/r" }, { path: "/r/.worktrees/fix-race" }],
    refs: [{ name: "fix-race-2", kind: "head" }, { name: "fix-race-3", kind: "remote" }],
  };
  assert.equal(forkSlug("fix race", graph), "fix-race-3");
});

test("forkWorktreePath finds the folder once git made it", () => {
  assert.equal(forkWorktreePath({ worktrees: [{ path: "/r" }] }, "fix"), "");
  assert.equal(forkWorktreePath({ worktrees: [{ path: "/r/.worktrees/fix" }, { path: "/r/.worktrees/fix-2" }] }, "fix"), "/r/.worktrees/fix");
  assert.equal(forkWorktreePath(null, "fix"), "");
});

test("forkRequest carries a name and, for a worktree, its folder — no task", () => {
  assert.deepEqual(forkRequest({ name: " fix ", workPath: "/r/.worktrees/fix" }), { name: "fix", workPath: "/r/.worktrees/fix" });
  assert.deepEqual(forkRequest({ name: "x" }), { name: "x" });
});
