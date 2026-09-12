import assert from "node:assert/strict";
import test from "node:test";
import { flightLines, formatAge, stallReason } from "./worktree-status.mjs";

// ADR-0123: the board's In flight section comes from here, so these are the
// rows that decide whether the owner sees real work, finished work, or noise.
const now = Date.parse("2026-09-12T12:00:00Z");
const aged = (hours) => now - hours * 3600_000;

const tree = (over = {}) => ({
  branch: "feat/x",
  rel: ".worktrees/x",
  ahead: 2,
  behind: 0,
  dirty: 0,
  lastCommitAt: aged(1),
  greenAt: aged(1),
  greenNow: true,
  isRoot: false,
  merged: false,
  ...over,
});

test("ages read as a human would say them", () => {
  assert.equal(formatAge(0), "just now");
  assert.equal(formatAge(5 * 60_000), "5 min ago");
  assert.equal(formatAge(3 * 3600_000), "3 h ago");
  assert.equal(formatAge(2 * 86_400_000), "2 d ago");
  assert.equal(formatAge(null), "—");
});

test("a merged branch is finished, not stalled", () => {
  assert.equal(stallReason(tree({ merged: true, ahead: 0 }), now), null);
});

test("an idle branch with work in it is stalled, and says since when", () => {
  const reason = stallReason(tree({ ahead: 3, lastCommitAt: aged(4 * 24) }), now);
  assert.match(reason, /no commit in 4 d/);
  assert.match(reason, /3 commit\(s\) ahead/);
});

test("a dirty tree is never called merged — someone is still working", () => {
  const live = tree({ ahead: 0, dirty: 4, lastCommitAt: aged(0.2), merged: false });
  assert.equal(stallReason(live, now), null);
});

test("no commits and nothing dirty is an empty branch", () => {
  assert.match(stallReason(tree({ ahead: 0, dirty: 0 }), now), /nothing committed/);
});

test("the board lists only the trees with unfinished work", () => {
  const lines = flightLines(
    [
      { isRoot: true, branch: "main" },
      tree({ branch: "feat/live" }),
      tree({ branch: "feat/done", ahead: 0, merged: true }),
    ],
    now,
  ).join("\n");
  assert.match(lines, /`feat\/live` — 2 ahead, 0 behind, clean, last commit 1 h ago, gates green 1 h ago/);
  assert.doesNotMatch(lines, /feat\/done/, "a merged branch is not in flight");
});

test("when every tree is merged, the board says so instead of listing them", () => {
  const lines = flightLines([tree({ ahead: 0, merged: true })], now).join("\n");
  assert.match(lines, /Nothing in flight/);
  assert.match(lines, /make worktree-gc/);
});
