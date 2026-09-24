#!/usr/bin/env node
// docs-living — the living-docs pass of docs-check, runnable on its own.
//
// An adversarial pass (2026-09-21) found three things wrong in a way no gate
// could see: ADR index rows that disagreed with their files, a subsystem file
// docs/architecture.md did not link, and relative links that resolved to
// nowhere. All three are cheap to read from the tree. `docs-check` runs this
// pass inside the full parity gate; `ci-scoped` runs it alone for a change
// that touches only docs/ (scope `metadata`), and `make close` runs it on every
// close. On 2026-09-23 a broken link in a study passed a metadata-only close
// and failed `make ci` on main after the fast-forward. The dev-flow guide's
// quoted refusals ride along (docs-quotes.mjs): same cost, same reach.
//
//   node scripts/docs-living.mjs     # exit 0 ok, 1 failures

import { existsSync, readFileSync, readdirSync } from "node:fs";
import { dirname, join, relative, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { bendQuoteFailures } from "./docs-quotes.mjs";

// Code is not a link. A fenced block or an inline code span may quote link
// syntax (Antigravity's include form, a regex) without pointing anywhere;
// that false positive is what broke main on 2026-09-23. Spans are matched
// within one line, so a stray backtick cannot hide a link lines away.
export function stripCode(markdown) {
  const out = [];
  let fence = null;
  for (const line of markdown.split("\n")) {
    const open = /^ {0,3}(`{3,}|~{3,})/.exec(line);
    if (fence) {
      const close = /^ {0,3}(`{3,}|~{3,})\s*$/.exec(line);
      if (close && close[1][0] === fence.char && close[1].length >= fence.len) fence = null;
      out.push("");
      continue;
    }
    if (open) {
      fence = { char: open[1][0], len: open[1].length };
      out.push("");
      continue;
    }
    out.push(line.replace(/(`+)(.+?)\1(?!`)/g, ""));
  }
  return out.join("\n");
}

function markdownFiles(dir) {
  return readdirSync(dir, { withFileTypes: true }).flatMap((e) => {
    if (e.name === "node_modules" || e.name === ".vitepress") return [];
    const p = join(dir, e.name);
    return e.isDirectory() ? markdownFiles(p) : e.name.endsWith(".md") ? [p] : [];
  });
}

// The ADR index carries the CURRENT status, so "superseded by …" and
// "amended …" are richer than a file's header and stay. The one direction
// that is always wrong is an accepted decision the index still presents as
// a proposal.
function decisionFailures(root) {
  const fails = [];
  const decisionsDir = join(root, "docs", "decisions");
  const indexPath = join(decisionsDir, "README.md");
  if (!existsSync(indexPath)) return fails;
  const index = readFileSync(indexPath, "utf8");
  const rows = new Map();
  for (const line of index.split("\n")) {
    const m = /^\| \[(\d{4})\]\(([^)]+)\)/.exec(line);
    if (!m) continue;
    const cells = line.split("|");
    rows.set(m[1], { file: m[2], status: (cells[cells.length - 2] ?? "").trim() });
  }
  const files = readdirSync(decisionsDir).filter((f) => /^\d{4}-.*\.md$/.test(f)).sort();
  for (const f of files) {
    const n = f.slice(0, 4);
    const row = rows.get(n);
    if (!row) {
      fails.push(`ADR ${n} has no row in docs/decisions/README.md — \`make adr\` appends one`);
      continue;
    }
    if (row.file !== f) fails.push(`ADR ${n}: the index links ${row.file}, the file is ${f}`);
    // Both spellings are in the corpus: "- **Status:** x" (the colon inside
    // the bold) and "- **Status**: x". Dropping the asterisks first matches
    // either, and matching neither is a failure rather than a skip — the
    // first version of this check silently passed the ten files written the
    // first way, which is the whole failure mode it exists to stop.
    const own = /^\s*-\s*Status\s*:\s*(\w+)/im.exec(readFileSync(join(decisionsDir, f), "utf8").replaceAll("*", ""));
    if (!own) {
      fails.push(`ADR ${n}: no "- **Status:** …" line this check can read — it would go unchecked`);
    } else if (own[1].toLowerCase() === "accepted" && /^proposed\b/i.test(row.status)) {
      fails.push(`ADR ${n} is accepted in its own file but "proposed" in the index — update docs/decisions/README.md`);
    }
  }
  for (const n of rows.keys()) {
    if (!files.some((f) => f.startsWith(n))) fails.push(`docs/decisions/README.md has a row for ADR ${n} with no file`);
  }
  return fails;
}

// docs/architecture.md only links (ADR-0105). A subsystem file it does not
// name is documentation nobody can find from the index.
function architectureFailures(root) {
  const fails = [];
  const archDir = join(root, "docs", "architecture");
  const archIndexPath = join(root, "docs", "architecture.md");
  if (!existsSync(archDir) || !existsSync(archIndexPath)) return fails;
  const linked = new Set(
    [...readFileSync(archIndexPath, "utf8").matchAll(/architecture\/([a-z0-9-]+\.md)/g)].map((m) => m[1]),
  );
  for (const f of readdirSync(archDir).filter((f) => f.endsWith(".md")).sort()) {
    if (!linked.has(f)) fails.push(`docs/architecture/${f} is not linked from docs/architecture.md`);
  }
  for (const f of linked) {
    if (!existsSync(join(archDir, f))) fails.push(`docs/architecture.md links architecture/${f}, which does not exist`);
  }
  return fails;
}

// Relative Markdown links inside docs/ must resolve. docs/handoff.md is the
// exception: it is generated by `make handoff` and is not in git (ADR-0123).
function linkFailures(root) {
  const fails = [];
  const docsDir = join(root, "docs");
  if (!existsSync(docsDir)) return fails;
  for (const file of markdownFiles(docsDir)) {
    for (const m of stripCode(readFileSync(file, "utf8")).matchAll(/\]\(([^)\s#]+)(?:#[^)]*)?\)/g)) {
      const target = m[1];
      if (/^(?:[a-z]+:|\/|#)/i.test(target) || target.endsWith("handoff.md")) continue;
      if (!existsSync(resolve(dirname(file), target))) {
        fails.push(`${relative(root, file)}: broken relative link ${target}`);
      }
    }
  }
  return fails;
}

// quotes: the dev-flow cards against the scripts they quote (docs-quotes.mjs).
// Only the fixture trees of docs-living.test.mjs, which hold no guide, turn it off.
export function livingDocsFailures(root, { quotes = true } = {}) {
  return [
    ...decisionFailures(root),
    ...architectureFailures(root),
    ...linkFailures(root),
    ...(quotes ? bendQuoteFailures(root) : []),
  ];
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  const root = join(dirname(fileURLToPath(import.meta.url)), "..");
  const fails = livingDocsFailures(root);
  if (fails.length) {
    console.error("docs-living FAILED:");
    for (const f of fails) console.error("  - " + f);
    process.exit(1);
  }
  console.log("docs-living ok: ADR index, architecture index, relative links and dev-flow quotes");
}
