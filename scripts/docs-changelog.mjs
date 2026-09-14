#!/usr/bin/env node
// Public changelog page (make docs). CHANGELOG.md stays assembled-only
// (ADR-0105). This preview folds current fragments into [Unreleased] without
// writing them, then writes docs-site/changelog.md for VitePress. The file
// is generated, not committed.
import { readFileSync, writeFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { assemble, parseFragment, listFragmentPaths } from "./changelog-assemble.mjs";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");

export function loadFragments(repoRoot = root) {
  return listFragmentPaths(repoRoot).map((f) => {
    try {
      return parseFragment(readFileSync(f, "utf8"));
    } catch (e) {
      throw new Error(`${f}: ${e.message}`);
    }
  });
}

export function renderPublicChangelog(changelog, fragments) {
  const folded = fragments.length ? assemble(changelog, fragments) : changelog;
  const start = folded.search(/^## \[/m);
  let rest = (start >= 0 ? folded.slice(start) : folded.replace(/^# Changelog\s*/, "")).trim();
  // VitePress compiles markdown as a Vue template. Bare `<agent>` / `<$0.01`
  // in CHANGELOG.md are tags, not prose, and the build dies.
  rest = rest.replace(/</g, "&lt;");
  return `---
description: What changed in PiCode, newest first.
editLink: false
---

# Changelog

What shipped. Newest first. **Unreleased** is on \`main\` and not yet in a versioned cut.

The format is [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

${rest}
`;
}

function main() {
  const changelog = readFileSync(join(root, "CHANGELOG.md"), "utf8");
  const fragments = loadFragments(root);
  const page = renderPublicChangelog(changelog, fragments);
  const out = join(root, "docs-site", "changelog.md");
  writeFileSync(out, page.endsWith("\n") ? page : page + "\n");
  console.log(`changelog page written: ${out} (${page.split("\n").length} lines)`);
}

if (process.argv[1] && fileURLToPath(import.meta.url) === resolve(process.argv[1])) {
  main();
}
