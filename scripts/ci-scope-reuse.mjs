#!/usr/bin/env node
// ci-scope-reuse — what `make close` is allowed to skip (ADR-0124).
//
// `make ci-scoped` records the shape of the run it just made green: the base
// commit, the tree, the changed paths, and the content the gates actually read
// (the tested Go packages' directories, `web/`, `docs/`…). `make close` then
// asks this module whether a catch-up merge could have changed any of that.
// The tree hash answers "nothing happened"; the covered hash answers "nothing
// this run READ changed, however much of main moved" — by content, never by
// name, so an amended file in the same path still re-runs.
//
//   node scripts/ci-scope-reuse.mjs --write --base <commit> [--packages "internal/server …"]
//   node scripts/ci-scope-reuse.mjs --check     # exit 0 reuse, 1 rerun, 2 error
//
// Both sides expand the same recorded spec with `git ls-tree`, so the two
// hashes are comparable: the stamp stores a pointer, never a tree.

import { createHash } from "node:crypto";
import { execFileSync } from "node:child_process";
import { existsSync, readFileSync, unlinkSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { changedPaths, localScope } from "./ci-scope.mjs";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");

function git(args, opts = {}) {
  return execFileSync("git", args, { cwd: root, encoding: "utf8", ...opts });
}

function isClean() {
  return git(["status", "--porcelain"]).trim() === "";
}

function headTree() {
  return git(["rev-parse", "HEAD^{tree}"]).trim();
}

function stampPath() {
  return join(git(["rev-parse", "--git-dir"]).trim(), "picode-ci-scoped.json");
}

// The directories the gates of this change set read. Pure so the rule is
// testable without a repository: a gate may be skipped only when everything it
// looked at still has the same content.
export function coveredRoots({ paths = [], packages = [], full = false } = {}) {
  const scope = localScope(paths);
  // A gate-shaping path (Makefile, go.mod, the hooks, ci-scope itself) ran the
  // whole matrix: everything it read is the whole tree.
  if (full || scope.full) return ["."];
  const roots = new Set(paths);
  // The Go gate reads exactly the packages it tested and their dependency
  // closure (the caller passes both); `make build` and `docs-check` compile the
  // whole binary or regenerate the spec, so they read the tree.
  const wholeGoTree = () => {
    roots.add("go.mod");
    roots.add("go.sum");
    roots.add("cmd/");
    roots.add("internal/");
  };
  if (scope.go || packages.length) {
    roots.add("go.mod");
    roots.add("go.sum");
    for (const p of packages) roots.add(p === "." ? "." : p + "/");
  }
  if (scope.web) {
    roots.add("web/");
    wholeGoTree(); // `make build` compiles the embedded binary
  }
  if (scope.packages) {
    roots.add("web/");
    roots.add("packages/");
    roots.add(".pi/");
  }
  if (scope.desktop) {
    // desktop-test compiles the pure crate standalone and the xwin gate the
    // whole crate — both read everything under desktop-shell/.
    roots.add("desktop-shell/");
  }
  if (scope.docs) {
    // docs-check re-generates OpenAPI (`go run ./cmd/picode-openapi`) and runs
    // scripts/ as child processes. `docs/` feeds only the living-docs pass,
    // which `make close` runs every time, so it stays out of the covered
    // roots (ADR-0124): a note landing on main never re-runs anyone's tests.
    roots.add("docs-site/");
    roots.add("styles/");
    roots.add("scripts/");
    roots.add(".vale.ini");
    wholeGoTree(); // docs-check regenerates the spec via `cmd/picode-openapi`
  }
  return [...roots].filter(Boolean).sort();
}

function coveredHash(spec) {
  const entries = git(["ls-tree", "-r", "HEAD", "--", ...spec]);
  return createHash("sha256").update(entries).digest("hex");
}

// The whole verdict, in one place: what may be reused and why (or why not).
export function decideReuse({ stamp, head, covered, dirty }) {
  if (!stamp) return { reuse: false, reason: "no green ci-scoped run is recorded for this worktree" };
  if (dirty) return { reuse: false, reason: "the working tree is dirty" };
  if (stamp.tree === head) return { reuse: true, reason: "the tree is unchanged since the green run" };
  if (stamp.covered === covered) {
    return {
      reuse: true,
      reason: `a merge moved main, but none of the ${stamp.count ?? "recorded"} covered path(s) changed content`,
    };
  }
  return { reuse: false, reason: "a path this run covered changed content — re-testing" };
}

// The module directory of a resolved Go import path: `github.com/x/y/internal/z`
// → `internal/z`; the module root becomes `.` (the whole tree, deliberately
// conservative — the root package imports everything).
export function packageDir(importPath, modulePath = modulePrefix()) {
  if (!modulePath || !importPath.startsWith(modulePath)) return importPath;
  const rest = importPath.slice(modulePath.length).replace(/^\//, "");
  return rest === "" ? "." : rest;
}

let cachedModule;
function modulePrefix() {
  if (cachedModule !== undefined) return cachedModule;
  try {
    cachedModule = (readFileSync(join(root, "go.mod"), "utf8").match(/^module\s+(\S+)/m) ?? [])[1] ?? "";
  } catch {
    cachedModule = "";
  }
  return cachedModule;
}

function write({ base, packages, full }) {
  if (!isClean()) {
    console.log("ci-scope-reuse: working tree is dirty — no stamp written (the gates cannot be reused)");
    return;
  }
  const paths = changedPaths(base, "HEAD");
  const spec = coveredRoots({ paths, packages, full });
  const stamp = {
    base,
    tree: headTree(),
    at: new Date().toISOString(),
    count: paths.length,
    covered: coveredHash(spec),
    roots: spec,
  };
  writeFileSync(stampPath(), JSON.stringify(stamp, null, 2) + "\n");
  console.log(`ci-scope-reuse: recorded ${spec.length} covered path(s) over ${paths.length} changed`);
}

function check() {
  const p = stampPath();
  let stamp = null;
  if (existsSync(p)) {
    try {
      stamp = JSON.parse(readFileSync(p, "utf8"));
    } catch {
      stamp = null;
    }
  }
  const head = headTree();
  const spec = stamp?.roots ?? [];
  const covered = stamp && spec.length ? coveredHash(spec) : null;
  const verdict = decideReuse({ stamp, head, covered, dirty: !isClean() });
  console.log(`${verdict.reuse ? "reuse" : "rerun"}: ${verdict.reason}`);
  return verdict.reuse ? 0 : 1;
}

function main() {
  const argv = process.argv.slice(2);
  if (argv.includes("--write")) {
    const baseIdx = argv.indexOf("--base");
    const base = baseIdx >= 0 ? argv[baseIdx + 1] : "main";
    const pkgIdx = argv.indexOf("--packages");
    const packages = pkgIdx >= 0 ? String(argv[pkgIdx + 1] ?? "").split(/\s+/).filter(Boolean).map((p) => packageDir(p)) : [];
    write({ base, packages, full: argv.includes("--full") });
    return 0;
  }
  if (argv.includes("--check")) return check();
  if (argv.includes("--forget")) {
    const p = stampPath();
    if (existsSync(p)) unlinkSync(p);
    console.log("ci-scope-reuse: stamp forgotten");
    return 0;
  }
  console.error("usage: ci-scope-reuse.mjs --write --base <commit> [--packages \"…\"] | --check | --forget");
  return 2;
}

const invoked = process.argv[1] && fileURLToPath(import.meta.url) === process.argv[1];
if (invoked) {
  try {
    process.exit(main());
  } catch (e) {
    console.error(`ci-scope-reuse: ${e.message}`);
    process.exit(2);
  }
}
