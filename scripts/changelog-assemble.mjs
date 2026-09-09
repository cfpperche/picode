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

// Insert each section's lines at the top of that section under [Unreleased],
// creating the section (in canonical order) when it is missing.
export function assemble(changelog, fragments) {
  const lines = changelog.split("\n");
  const start = lines.findIndex((l) => /^## \[Unreleased\]/.test(l));
  if (start < 0) throw new Error("CHANGELOG.md has no [Unreleased] section");
  let end = lines.findIndex((l, i) => i > start && /^## \[/.test(l));
  if (end < 0) end = lines.length;
  const block = lines.slice(start + 1, end);

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
  while (block.length && !block[block.length - 1].trim()) block.pop();
  return [...lines.slice(0, start + 1), ...block, "", ...lines.slice(end)].join("\n");
}

function main() {
  const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
  const dir = join(root, "docs", "changelog.d");
  const files = readdirSync(dir)
    .filter((f) => f.endsWith(".md") && f !== "README.md")
    .map((f) => join(dir, f))
    .sort((a, b) => statSync(a).mtimeMs - statSync(b).mtimeMs); // oldest first; newest ends on top
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
