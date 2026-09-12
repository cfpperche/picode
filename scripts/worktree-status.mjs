#!/usr/bin/env node
// worktree-status — what is actually in flight, from disk (ADR-0123).
//
// The handoff board used to report in-flight work as prose, and drifted: it
// named a worktree removed days earlier and missed one that was dirty on
// disk. In-flight state is git state, so it is derived here and rendered in
// two places: the board's `## In flight` section (`--markdown`) and the
// owner's `make worktree-status` (`--json` for scripts).
//
//   node scripts/worktree-status.mjs [--markdown|--json] [--idle-hours 24]

import { execFileSync } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";
import { dirname, join, relative } from "node:path";
import { fileURLToPath } from "node:url";

const here = join(dirname(fileURLToPath(import.meta.url)), "..");

// The reference is the MAIN worktree (the common git dir's parent), not the
// tree the script happens to run in: `git worktree list` reports absolute
// paths for every tree, including the root one.
let cachedRoot;
function repoRoot() {
  if (cachedRoot) return cachedRoot;
  try {
    const common = execFileSync("git", ["rev-parse", "--path-format=absolute", "--git-common-dir"], {
      cwd: here,
      encoding: "utf8",
      stdio: ["ignore", "pipe", "pipe"],
    }).trim();
    cachedRoot = dirname(common);
  } catch {
    cachedRoot = here;
  }
  return cachedRoot;
}
const root = here;

function git(args, cwd = root) {
  return execFileSync("git", args, { cwd, encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] }).trim();
}

function gitRaw(args, cwd = root) {
  try {
    return execFileSync("git", args, { cwd, encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] });
  } catch {
    return "";
  }
}

export function formatAge(ms) {
  if (ms == null || Number.isNaN(ms)) return "—";
  const min = Math.round(ms / 60000);
  if (min < 1) return "just now";
  if (min < 60) return `${min} min ago`;
  const h = Math.round(min / 60);
  if (h < 24) return `${h} h ago`;
  return `${Math.round(h / 24)} d ago`;
}

// A worktree that is behind nobody but produces no commits is the shape of an
// abandoned session: it holds a branch, a handoff note and a stale view of
// main. Report it; never delete it for the owner (AGENTS.md §5).
export function stallReason({ ahead, behind, dirty, lastCommitAt }, now = Date.now(), idleHours = 24) {
  if (!lastCommitAt) return "no commits yet";
  const idle = now - lastCommitAt; // ms, everywhere
  if (ahead === 0 && dirty === 0) return "nothing committed — empty branch";
  if (idle > idleHours * 3600_000) {
    const where = ahead > 0 ? `${ahead} commit(s) ahead` : "nothing ahead";
    return `no commit in ${Math.round(idle / 3600_000 / 24 * 10) / 10} d (${where})`;
  }
  return null;
}

export function flightLines(worktrees, now = Date.now(), idleHours = 24) {
  const live = worktrees.filter((w) => !w.isRoot);
  if (!live.length) return ["No worktree is open: `main` is the only tree."];
  return live.map((w) => {
    const bits = [`${w.ahead} ahead`, `${w.behind} behind`];
    bits.push(w.dirty ? `${w.dirty} dirty file(s)` : "clean");
    bits.push(`last commit ${formatAge(now - w.lastCommitAt)}`);
    if (w.greenAt) {
      bits.push(w.greenNow ? `gates green ${formatAge(now - w.greenAt)}` : "gates stale");
    } else {
      bits.push("no green run recorded");
    }
    const stall = stallReason(w, now, idleHours);
    return `- \`${w.branch}\` — ${bits.join(", ")}${stall ? ` — **stalled: ${stall}**` : ""}`;
  });
}

export function readWorktrees() {
  const mainWorktree = repoRoot();
  const out = gitRaw(["worktree", "list", "--porcelain"]);
  const blocks = out.split("\n\n").filter((b) => b.trim());
  const worktrees = [];
  for (const block of blocks) {
    const lines = block.split("\n");
    const path = (lines.find((l) => l.startsWith("worktree ")) ?? "").slice("worktree ".length);
    if (!path) continue;
    const head = (lines.find((l) => l.startsWith("HEAD ")) ?? "").slice(5);
    const branchRef = (lines.find((l) => l.startsWith("branch ")) ?? "").slice(7);
    const branch = branchRef.replace(/^refs\/heads\//, "") || "(detached)";
    const isRoot = path === mainWorktree;

    let ahead = 0;
    let behind = 0;
    const counts = gitRaw(["rev-list", "--left-right", "--count", `main...HEAD`], path).trim().split(/\s+/);
    if (counts.length === 2) {
      behind = Number(counts[0]) || 0;
      ahead = Number(counts[1]) || 0;
    }
    const dirty = gitRaw(["status", "--porcelain"], path).split("\n").filter((l) => l.trim()).length;
    const lastCommitAt = (Number(gitRaw(["log", "-1", "--format=%ct"], path).trim()) || 0) * 1000;
    const subject = gitRaw(["log", "-1", "--format=%s"], path).trim();

    let greenAt = 0;
    let greenNow = false;
    try {
      const gitDir = git(["rev-parse", "--git-dir"], path);
      const stampFile = join(gitDir.startsWith("/") ? gitDir : join(path, gitDir), "picode-ci-scoped.json");
      if (existsSync(stampFile)) {
        const stamp = JSON.parse(readFileSync(stampFile, "utf8"));
        greenAt = Date.parse(stamp.at) || 0;
        greenNow = stamp.tree === git(["rev-parse", "HEAD^{tree}"], path);
      }
    } catch {
      /* no stamp: report "no green run recorded" */
    }

    worktrees.push({
      path,
      rel: isRoot ? "." : relative(mainWorktree, path),
      branch,
      head,
      ahead,
      behind,
      dirty,
      lastCommitAt,
      subject,
      greenAt,
      greenNow,
      isRoot,
    });
  }
  // The root first, then the worktrees that moved most recently.
  worktrees.sort((a, b) => (a.isRoot === b.isRoot ? b.lastCommitAt - a.lastCommitAt : a.isRoot ? -1 : 1));
  return worktrees;
}

function human(worktrees, now, idleHours) {
  const rows = worktrees.map((w) => {
    const gates = w.greenAt ? (w.greenNow ? `green ${formatAge(now - w.greenAt)}` : "stale") : "—";
    return {
      cell: [
        w.rel,
        w.branch,
        `${w.ahead}/${w.behind}`,
        w.dirty ? String(w.dirty) : "0",
        w.lastCommitAt ? formatAge(now - w.lastCommitAt) : "—",
        gates,
      ],
      stall: w.isRoot ? null : stallReason(w, now, idleHours),
    };
  });
  const head = ["worktree", "branch", "ahead/behind", "dirty", "last commit", "gates"];
  const widths = head.map((h, i) => Math.max(h.length, ...rows.map((r) => r.cell[i].length)));
  const line = (cells) => cells.map((c, i) => c.padEnd(widths[i])).join("  ").trimEnd();
  const out = [line(head), line(head.map((h) => "-".repeat(h.length)))];
  for (const r of rows) out.push(line(r.cell) + (r.stall ? `   ⚠ ${r.stall}` : ""));
  return out.join("\n");
}

function main() {
  const argv = process.argv.slice(2);
  const idleIdx = argv.indexOf("--idle-hours");
  const idleHours = idleIdx >= 0 ? Number(argv[idleIdx + 1]) || 24 : 24;
  const now = Date.now();
  const worktrees = readWorktrees();
  if (argv.includes("--json")) {
    console.log(JSON.stringify({ now: new Date(now).toISOString(), worktrees }, null, 2));
    return 0;
  }
  if (argv.includes("--markdown")) {
    console.log(flightLines(worktrees, now, idleHours).join("\n"));
    return 0;
  }
  console.log(human(worktrees, now, idleHours));
  const stalled = worktrees.filter((w) => !w.isRoot && stallReason(w, now, idleHours));
  if (stalled.length) {
    console.log(
      `\n${stalled.length} worktree(s) stalled — an idle branch holds a handoff note and a stale view of main.\n` +
        `Finish it, or record the gap in docs/handoff/open/process.md and remove the tree (make worktree-gc).`,
    );
  }
  return 0;
}

const invoked = process.argv[1] && fileURLToPath(import.meta.url) === process.argv[1];
if (invoked) process.exit(main());
