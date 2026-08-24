import { statusSegments } from "./statusbar.js";
import assert from "node:assert/strict";
import { test } from "node:test";

test("hides empty segments", () => {
  assert.deepEqual(statusSegments({}), []);
  assert.equal(statusSegments({ cwd: "~/picode" })[0].text, "~/picode");
});

test("git worktree and context tones", () => {
  const parts = statusSegments({
    cwd: "~/w",
    branch: "main",
    worktree: "hotfix",
    contextWindow: 200000,
    contextPercent: 81,
    cost: 0.12,
  });
  assert.equal(parts.find((p) => p.key === "git").text, "main@hotfix");
  assert.equal(parts.find((p) => p.key === "ctx").tone, "warn");
  assert.equal(parts.find((p) => p.key === "cost").text, "$0.12");
});
