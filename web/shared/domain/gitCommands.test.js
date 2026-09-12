import test from "node:test";
import assert from "node:assert/strict";
import { askedNote, branchChip } from "./gitCommands.js";

// The git graph composes thirty-odd actions the rail's own verb table never
// knew (ADR-0096). When the server named one, that name is what the toast
// says; the table stays the fallback for the six it does know.
test("a composed verb wins over the built-in table", () => {
  assert.equal(askedNote("Atlas", "merge", {}, "merge feat/x"), "Asked Atlas to merge feat/x.");
  assert.equal(askedNote("Atlas", "commit", {}), "Asked Atlas to commit.");
  assert.equal(askedNote("Atlas", "reset-hard", {}), "Asked Atlas to do it.", "an unknown action still reads as a sentence");
  assert.equal(askedNote("Atlas", "reset-hard", {}, "reset to 0123456, discarding the changes"),
    "Asked Atlas to reset to 0123456, discarding the changes.");
});

test("the channel the prompt took still decides the tense", () => {
  assert.equal(askedNote("Atlas", "merge", { mode: "interactive" }, "merge feat/x"), "Asked Atlas to merge feat/x in its terminal.");
  assert.equal(askedNote("Atlas", "merge", { busy: true }, "merge feat/x"), "Queued for Atlas: merge feat/x after its current turn.");
});

// The Inspector's row draws one string, the branch: two ellipsised spans
// sharing a chip's worth of width read as "· fe... ·...". The worktree is
// still an answer someone wants, so the tooltip carries it.
test("the branch chip's title names the worktree the row no longer draws", () => {
  const chip = branchChip({ git: true, branch: "feat/canvas-resize", worktree: "canvas-resize", upstream: "origin/feat/canvas-resize", ahead: 2 });
  assert.equal(chip.name, "feat/canvas-resize");
  assert.equal(chip.worktree, "canvas-resize");
  assert.equal(chip.title, "Branch feat/canvas-resize, 2 ahead origin/feat/canvas-resize, worktree canvas-resize");
  const plain = branchChip({ git: true, branch: "main", upstream: "origin/main" });
  assert.equal(plain.title, "Branch main is level with origin/main", "no worktree, no clause");
  const detached = branchChip({ git: true, branch: "0123456", detached: true, worktree: "wt" });
  assert.equal(detached.title, "Detached at 0123456, worktree wt");
});
