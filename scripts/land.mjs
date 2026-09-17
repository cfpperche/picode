#!/usr/bin/env node
// land — put a worktree branch on main, then gate it (ADR-0145 follow-up).
//
// The closing rite used to be two commands typed at the root:
//
//   git merge --ff-only <branch> && make ci
//
// and the root checkout is *shared*. Measured 2026-09-16: one session ran
// `git add -A` for its own docs and swept another agent's two benchmarks and an
// untracked plan into the index — the pre-commit hook refused the commit, which
// was luck, not design. The dangerous verb was the `commit`, and this script has
// none: it only fast-forwards and gates. (Measured too: a fast-forward leaves a
// foreign *staged* change intact — `git merge --ff-only` updates the paths the
// branch changes and touches nothing else — so another agent's work in flight
// is not a reason to refuse.)
//
// What it does check, before touching anything:
//   - you are in the root checkout, on main (a merge from a worktree is a
//     mistake that has happened);
//   - the branch exists;
//   - main can fast-forward to it — otherwise the branch needs `make close`,
//     and the message says so instead of "fatal: Not possible to fast-forward";
//   - no path the branch changes is dirty *here*: git would refuse the merge
//     mid-way, so name the file and the session that owns it instead.
//
// Then it runs the merge gate (`make ci`) on main. It never rebases, never
// resolves conflicts, never commits.

import { execFileSync, spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";

// land runs from the root checkout — the tree it merges into — so that is the
// tree it reads and writes. `make land` runs it there.
const root = process.cwd();

function git(args, { allowFail = false } = {}) {
  const run = spawnSync("git", args, { cwd: root, encoding: "utf8" });
  if (run.status !== 0) {
    if (allowFail) return "";
    throw new Error(`git ${args.join(" ")}: ${(run.stderr || "").trim()}`);
  }
  return run.stdout;
}

// The decision, in one place so the table can be tested without a repo.
// `collisions` are the root's dirty paths that the branch also changes — the
// only dirt that makes this merge unsafe; `ahead` is whether main is an
// ancestor of the branch (a fast-forward is possible).
export function plan({ branch, collisions = [], ahead = false }) {
  if (!branch) return { ok: false, reason: "usage: make land BRANCH=<name> (the branch to fast-forward main to)" };
  if (collisions.length) {
    return {
      ok: false,
      reason:
        `the root checkout has local changes to ${collisions.length} path(s) this branch also changes:\n` +
        collisions.map((p) => `  ${p}`).join("\n") +
        "\nCommit or discard them (in whatever session owns them), then land again.",
    };
  }
  if (!ahead) {
    return {
      ok: false,
      reason: `main cannot fast-forward to ${branch}: either it is merged already, or it needs \`make close\` (merge main into the branch) first.`,
    };
  }
  return { ok: true, reason: `fast-forwarding main to ${branch}` };
}

// dirtyPaths lists what the root has that is not committed (staged or
// modified); untracked files are not included — a merge overwrites none of
// them unless git says so, and it says so itself.
function dirtyPaths() {
  return git(["status", "--porcelain"])
    .split("\n")
    .filter((l) => l && !l.startsWith("??"))
    .map((l) => l.slice(3).trim());
}

function main() {
  const branch = (process.argv[2] || "").trim();
  const here = git(["rev-parse", "--abbrev-ref", "HEAD"]).trim();
  if (here !== "main") {
    console.error(`land: run this from the root checkout on main (now on ${here}).`);
    return 1;
  }
  const exists = branch ? git(["rev-parse", "--verify", "--quiet", `refs/heads/${branch}`], { allowFail: true }) : "";
  if (branch && !exists) {
    console.error(`land: no such branch ${branch} — \`git branch --list '${branch}'\` to see what exists.`);
    return 1;
  }
  // The collision set is the intersection, not the whole dirty tree: another
  // agent's work in flight is normal here and the merge leaves it alone.
  const changed = branch ? git(["diff", "--name-only", `main...${branch}`]).split("\n").filter(Boolean) : [];
  const collisions = dirtyPaths().filter((p) => changed.includes(p));
  const verdict = plan({ branch, collisions, ahead: !!exists && isAncestor("main", branch) });
  if (!verdict.ok) {
    console.error(`land: ${verdict.reason}`);
    return 1;
  }
  console.log(`land: ${verdict.reason}`);
  const commits = git(["log", "--oneline", `main..${branch}`]).trim().split("\n").filter(Boolean);
  for (const c of commits) console.log(`  ${c}`);
  try {
    execFileSync("git", ["merge", "--ff-only", branch], { cwd: root, stdio: "inherit" });
  } catch {
    console.error(`land: the fast-forward failed — nothing was changed. Run \`make close\` in that worktree.`);
    return 1;
  }
  const head = git(["rev-parse", "--short", "HEAD"]).trim();
  console.log(`land: main is at ${head}. Running the merge gate…`);
  const ci = spawnSync("make", ["ci"], { cwd: root, stdio: "inherit" });
  if (ci.status !== 0) {
    console.error(`land: make ci FAILED on main at ${head} — fix it (the branch is already merged; a follow-up branch is the honest fix).`);
    return 1;
  }
  console.log(`land: main is at ${head} and green. Cleanup:`);
  console.log(`  git worktree remove .worktrees/<name> && git branch -d ${branch}`);
  console.log("  make handoff");
  return 0;
}

function isAncestor(a, b) {
  const run = spawnSync("git", ["merge-base", "--is-ancestor", a, b], { cwd: root });
  return run.status === 0;
}

const invoked = process.argv[1] && fileURLToPath(import.meta.url) === process.argv[1];
if (invoked) process.exit(main());
