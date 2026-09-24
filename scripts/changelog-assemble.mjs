#!/usr/bin/env node
// make changelog — fold docs/changelog.d/*.md fragments into the [Unreleased]
// section of CHANGELOG.md (ADR-0105). Branches never edit CHANGELOG.md: two
// branches inserting at the top of [Unreleased] conflicted in 31 of 75
// catch-up merges in three days. Each branch leaves one fragment; this folds
// them, newest first inside each Keep a Changelog section, removes them and
// stages both.
import { readFileSync, writeFileSync, readdirSync, rmSync, statSync } from "node:fs";
import { execFileSync } from "node:child_process";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

export const SECTIONS = ["Added", "Changed", "Deprecated", "Removed", "Fixed", "Security"];

// A fragment is `### Section` headings followed by bullet blocks.
export function parseFragment(text) {
  const out = {};
  let cur = null;
  for (const line of text.split("\n")) {
    const m = /^###\s+(\w+)\s*$/.exec(line);
    if (m) {
      cur = SECTIONS.find((s) => s.toLowerCase() === m[1].toLowerCase());
      if (!cur) throw new Error(`unknown section "${m[1]}" (use ${SECTIONS.join(", ")})`);
      out[cur] ??= [];
      continue;
    }
    if (cur === null) {
      if (line.trim()) throw new Error(`text before the first "### Section" heading: ${line.slice(0, 60)}`);
      continue;
    }
    out[cur].push(line);
  }
  for (const k of Object.keys(out)) {
    while (out[k].length && !out[k][0].trim()) out[k].shift();
    while (out[k].length && !out[k][out[k].length - 1].trim()) out[k].pop();
    if (!out[k].length) delete out[k];
  }
  if (!Object.keys(out).length) throw new Error("fragment has no entries");
  return out;
}

// Fold every repeated `### Section` in one release block into the first of
// its name, in canonical order, keeping each entry where its heading put it.
// A block drifts into duplicates the moment anything writes a heading that is
// already there — years of direct edits did, before fragments (ADR-0105) —
// and nothing healed it: `assemble` inserts into the *first* match and walks
// past the rest. The cut then publishes that block as the GitHub release
// body, so 0.2.0 was one fold away from announcing six "Fixed" headings.
// Exported for the test; `assemble` runs it on every fold.
export function normalizeBlock(block) {
  const heading = /^###\s+(\w+)\s*$/;
  const buckets = new Map();
  const before = [];
  let cur = null;
  for (const line of block) {
    const m = heading.exec(line);
    if (m) {
      const s = SECTIONS.find((x) => x.toLowerCase() === m[1].toLowerCase());
      if (!s) throw new Error(`unknown section "${m[1]}" in [Unreleased] (use ${SECTIONS.join(", ")})`);
      cur = s;
      if (!buckets.has(s)) buckets.set(s, []);
      continue;
    }
    (cur === null ? before : buckets.get(cur)).push(line);
  }
  const trim = (ls) => {
    const out = ls.slice();
    while (out.length && !out[0].trim()) out.shift();
    while (out.length && !out[out.length - 1].trim()) out.pop();
    return out;
  };
  const head = trim(before);
  const out = head.length ? [...head, ""] : [];
  for (const s of SECTIONS) {
    const body = trim(buckets.get(s) || []);
    if (!body.length) continue;
    out.push(`### ${s}`, "", ...body, "");
  }
  while (out.length && !out[out.length - 1].trim()) out.pop();
  return out;
}

// Insert each section's lines at the top of that section under [Unreleased],
// creating the section (in canonical order) when it is missing.
export function assemble(changelog, fragments) {
  const lines = changelog.split("\n");
  let start = lines.findIndex((l) => /^## \[Unreleased\]/.test(l));
  if (start < 0) {
    const firstRelease = lines.findIndex((l) => /^## \[[^\]]+\]/.test(l));
    if (firstRelease < 0) throw new Error("CHANGELOG.md has no release section");
    // A release cut may consume the entire Unreleased block. Recreate it
    // before the newest release when the next fragment arrives.
    lines.splice(firstRelease, 0, "## [Unreleased]", "");
    start = firstRelease;
  }
  let end = lines.findIndex((l, i) => i > start && /^## \[/.test(l));
  if (end < 0) end = lines.length;
  // Heal first: an inherited duplicate would otherwise swallow this fold's
  // entries into whichever copy happened to come first.
  const block = normalizeBlock(lines.slice(start + 1, end));

  const merged = {};
  for (const f of fragments) {
    for (const [s, ls] of Object.entries(f)) (merged[s] ??= []).unshift(...ls, "");
  }

  for (const s of SECTIONS) {
    if (!merged[s]) continue;
    const entries = merged[s];
    const heading = (t) => new RegExp(`^### ${t}\\s*$`);
    const h = block.findIndex((l) => heading(s).test(l));
    if (h < 0) {
      // Canonical order: right before the first section that follows it.
      // `entries` ends with a blank; the slot before a following heading is
      // already blank, so drop it there to keep one empty line between them.
      const later = SECTIONS.slice(SECTIONS.indexOf(s) + 1);
      let at = block.findIndex((l) => later.some((t) => heading(t).test(l)));
      const body = at < 0 ? entries : entries.slice(0, -1);
      if (at < 0) at = block.length;
      while (at > 0 && !block[at - 1].trim()) at--;
      block.splice(at, 0, "", `### ${s}`, "", ...body);
      continue;
    }
    let i = h + 1;
    while (i < block.length && !block[i].trim()) i++;
    block.splice(i, 0, ...entries);
  }
  const healed = normalizeBlock(block);
  return [...lines.slice(0, start + 1), "", ...healed, "", ...lines.slice(end)].join("\n");
}

// Commit time decides the order (a fresh clone gives every file the same
// mtime); an uncommitted fragment falls back to its mtime.
export function listFragmentPaths(root) {
  const dir = join(root, "docs", "changelog.d");
  return readdirSync(dir)
    .filter((f) => f.endsWith(".md") && f !== "README.md")
    .map((f) => join(dir, f))
    .sort((a, b) => when(root, a) - when(root, b));
}

function when(root, f) {
  try {
    const t = execFileSync("git", ["log", "-1", "--format=%ct", "--", f], { cwd: root, stdio: ["ignore", "pipe", "ignore"] })
      .toString()
      .trim();
    if (t) return Number(t) * 1000;
  } catch {}
  return statSync(f).mtimeMs;
}

function main() {
  // --check: parse one fragment from stdin (the pre-commit hook feeds it the
  // staged blob) and exit non-zero with the reason.
  if (process.argv.includes("--check")) {
    try {
      parseFragment(readFileSync(0, "utf8"));
    } catch (e) {
      console.error(e.message);
      process.exit(1);
    }
    return;
  }
  const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
  const files = listFragmentPaths(root);
  if (!files.length) {
    console.log("changelog: no fragments in docs/changelog.d/");
    return;
  }
  const fragments = files.map((f) => {
    try {
      return parseFragment(readFileSync(f, "utf8"));
    } catch (e) {
      throw new Error(`${f}: ${e.message}`);
    }
  });
  const path = join(root, "CHANGELOG.md");
  writeFileSync(path, assemble(readFileSync(path, "utf8"), fragments));
  for (const f of files) rmSync(f);
  execFileSync("git", ["add", "-A", "--", "CHANGELOG.md", "docs/changelog.d"], { cwd: root });
  console.log(`changelog: folded ${files.length} fragment(s) into [Unreleased] and staged the result — commit it on main`);
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) main();
