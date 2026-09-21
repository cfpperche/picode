#!/usr/bin/env node
// The pi key map drifts under us: the pane shipped 89 of pi's 90 documented
// actions and nine defaults that are wrong on Windows and WSL, and nothing
// noticed because the catalog is hand-written from someone else's docs.
//
// This probe re-reads the installed pi's own docs/keybindings.md and compares
// it with internal/pikeys/catalog.go — ids, defaults, and the per-platform
// alternates. It is a probe, not a gate: a machine without pi says so loudly and
// exits 0, so `make ci` never depends on a vendor package being installed.
import { execFileSync } from "node:child_process";
import { readFileSync, existsSync } from "node:fs";
import { dirname, join, resolve } from "node:path";

const skip = (why) => { console.log("keys-drift: " + why + " — skipped"); process.exit(0); };

let bin = "";
try {
  bin = execFileSync("command", ["-v", "pi"], { encoding: "utf8", shell: "/bin/bash" }).trim();
} catch {
  skip("pi is not installed on this machine");
}
if (!bin) skip("pi is not installed on this machine");

// …/lib/node_modules/<pkg>/dist/bundle/cli.js -> …/lib/node_modules/<pkg>
const real = execFileSync("readlink", ["-f", bin], { encoding: "utf8" }).trim();
const pkgRoot = dirname(dirname(dirname(real)));
const docsPath = join(pkgRoot, "docs", "keybindings.md");
if (!existsSync(docsPath)) skip("the installed pi has no docs/keybindings.md (" + docsPath + ")");

const docs = readFileSync(docsPath, "utf8");
const catalogPath = resolve("internal/pikeys/catalog.go");
if (!existsSync(catalogPath)) skip("run this from the repository root");
const catalog = readFileSync(catalogPath, "utf8");

// Documented: | `tui.editor.cursorUp` | `up`, `ctrl+p` (`alt+up` on WSL) | … |
// The parenthetical names the OTHER platforms, so it is stripped: the base
// default is what pi uses on this machine unless the catalog declares an
// alternate for the platform (pikeys.Catalog.Alt).
const documented = new Map();
const rowRe = /^\| `((?:tui|app)\.[A-Za-z.]+)` \| ([^|]*?) \|/gm;
for (const [, id, raw] of docs.matchAll(rowRe)) {
  const base = raw.replace(/\([^)]*\)/g, "");
  const keys = base.trim() === "*(none)*" || base.trim() === ""
    ? []
    : [...base.matchAll(/`([^`]+)`/g)].map((m) => m[1]);
  documented.set(id, keys);
}

// Catalog: {"id", "Group", "Label", []string{"a", "b"}, alt(...)}
const declared = new Map();
const entryRe = /\{"((?:tui|app)\.[A-Za-z.]+)",\s*"([^"]*)",\s*"([^"]*)",\s*(nil|\[\]string\{[^}]*\})/g;
for (const [, id, , , defs] of catalog.matchAll(entryRe)) {
  declared.set(id, defs === "nil" ? [] : [...defs.matchAll(/"([^"]*)"/g)].map((m) => m[1]));
}

const missing = [...documented.keys()].filter((id) => !declared.has(id));
const extra = [...declared.keys()].filter((id) => !documented.has(id));
const drift = [...documented.keys()]
  .filter((id) => declared.has(id))
  .filter((id) => documented.get(id).join(",") !== declared.get(id).join(","))
  .map((id) => ({ id, docs: documented.get(id), catalog: declared.get(id) }));

const problems = missing.length + extra.length + drift.length;
if (!problems) {
  console.log(`keys-drift: catalog matches pi ${JSON.parse(readFileSync(join(pkgRoot, "package.json"), "utf8")).version} ` +
    `(${documented.size} actions, ${[...documented.values()].filter((k) => k.length === 0).length} unbound)`);
  process.exit(0);
}
console.error("keys-drift: the catalog and pi's own docs disagree");
for (const id of missing) console.error("  documented, not in the pane: " + id);
for (const id of extra) console.error("  in the pane, not documented: " + id);
for (const d of drift) {
  console.error(`  ${d.id}: docs=${JSON.stringify(d.docs)} catalog=${JSON.stringify(d.catalog)}`);
}
console.error("  Fix internal/pikeys/catalog.go (and, for an alternate, its Alt map) — the pane prints these.");
process.exit(1);
