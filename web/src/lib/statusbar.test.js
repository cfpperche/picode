import { statusSegments, formatSessionCost, fmtElapsed, workspaceStatusPath } from "./statusbar.js";
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
  assert.equal(parts.find((p) => p.key === "ch").text, "40% cached");
  assert.equal(parts.find((p) => p.key === "cost").text, "$0.12");
  assert.equal(parts.find((p) => p.key === "name").text, "refactor-auth");
});

test("formatSessionCost hides zero", () => {
  assert.equal(formatSessionCost(0), "");
  assert.equal(formatSessionCost(null), "");
  assert.equal(formatSessionCost(0.05), "$0.05");
  assert.equal(formatSessionCost(1.2), "$1.20");
});

test("compacting never surfaces in the composer bar (progress is a chat line)", () => {
  const parts = statusSegments({ compacting: Date.now() - 65000, cwd: "~/p", cost: 0.5 });
  assert.ok(!parts.some((p) => p.key === "compact"));
  assert.ok(parts.some((p) => p.key === "cwd"));
});

test("fmtElapsed formats minutes and seconds", () => {
  assert.equal(fmtElapsed(0), "0:00");
  assert.equal(fmtElapsed(9500), "0:09");
  assert.equal(fmtElapsed(65000), "1:05");
  assert.equal(fmtElapsed(600000), "10:00");
});

test("workspaceStatusPath scopes the bar to one agent", () => {
  // A workspace agent: the id is mandatory, or the server answers with
  // the workspace's FIRST agent and every later agent's screen shows
  // that agent's context, spend and cache.
  assert.equal(
    workspaceStatusPath("ws1", "agent-9"),
    "/api/workspaces/ws1/status?agent=agent-9"
  );
  // A free agent has no workspace to scope into: no query.
  assert.equal(workspaceStatusPath("ws_free", ""), "/api/workspaces/ws_free/status");
  assert.equal(workspaceStatusPath("ws_free", null), "/api/workspaces/ws_free/status");
  // Ids are user-visible strings; encode them.
  assert.equal(
    workspaceStatusPath("w s", "a&b"),
    "/api/workspaces/w%20s/status?agent=a%26b"
  );
});
