import assert from "node:assert/strict";
import test from "node:test";
import { plan } from "./land.mjs";

// The guard that keeps a shared root checkout from being swept (2026-09-16: a
// session's `git add -A` staged another agent's benchmarks; the pre-commit hook
// caught it, which was luck). Every row here is a state the root can be in.

test("a clean checkout with a fast-forwardable branch lands", () => {
  const v = plan({ branch: "feat/x", ahead: true });
  assert.equal(v.ok, true);
  assert.match(v.reason, /fast-forwarding main to feat\/x/);
});

// Measured (temp repo, git 2.43): a fast-forward updates the paths the branch
// changes and leaves a foreign staged change intact, so another agent's work in
// flight is not a reason to refuse — only an overlap is.
test("a dirty root is fine unless the branch changes that path too", () => {
  const foreign = plan({ branch: "feat/x", ahead: true, collisions: [] });
  assert.equal(foreign.ok, true);
  const overlap = plan({ branch: "feat/x", ahead: true, collisions: ["internal/server/browser.go"] });
  assert.equal(overlap.ok, false);
  assert.match(overlap.reason, /browser\.go/);
  assert.match(overlap.reason, /Commit or discard/);
});

test("an untracked file is nobody's collision until git says so", () => {
  const v = plan({ branch: "feat/x", ahead: true });
  assert.equal(v.ok, true);
});

test("a branch that cannot fast-forward refuses, and names the rite", () => {
  const v = plan({ branch: "feat/x", ahead: false });
  assert.equal(v.ok, false);
  assert.match(v.reason, /make close/);
});

test("no branch is a usage error, not a merge", () => {
  const v = plan({ branch: "", ahead: true });
  assert.equal(v.ok, false);
  assert.match(v.reason, /make land BRANCH=/);
});

test("an overlap wins over the fast-forward check: fix the tree first", () => {
  const v = plan({ branch: "feat/x", ahead: false, collisions: ["a.go"] });
  assert.equal(v.ok, false);
  assert.match(v.reason, /a\.go/);
});
