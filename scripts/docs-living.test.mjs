import { test } from "node:test";
import assert from "node:assert/strict";
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { livingDocsFailures, stripCode } from "./docs-living.mjs";

// A throwaway repository root holding only the files a case names.
function tree(files) {
  const root = mkdtempSync(join(tmpdir(), "docs-living-"));
  for (const [path, body] of Object.entries(files)) {
    mkdirSync(dirname(join(root, path)), { recursive: true });
    writeFileSync(join(root, path), body);
  }
  return root;
}

function failures(files) {
  const root = tree(files);
  try {
    return livingDocsFailures(root, { quotes: false });
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
}

test("stripCode: inline spans and fenced blocks are not links", () => {
  assert.equal(stripCode("a `@[label](path)` b").includes("](path)"), false);
  assert.equal(stripCode("x ``[a](b) ` c`` y").includes("](b)"), false);
  assert.equal(stripCode("```md\n[a](nowhere.md)\n```\nafter").includes("nowhere"), false);
  assert.equal(stripCode("~~~\n[a](nowhere.md)\n~~~").includes("nowhere"), false);
  assert.match(stripCode("```\ncode\n```\n[kept](real.md)"), /\[kept\]\(real\.md\)/);
});

test("stripCode: a stray backtick hides nothing beyond its own line", () => {
  assert.match(stripCode("the ` key\n[kept](real.md)"), /\[kept\]\(real\.md\)/);
});

// Each row: the docs a case holds → the failures it must report (substrings).
const linkCases = [
  {
    name: "a broken relative link fails, naming the file and the target",
    files: { "docs/a.md": "see [b](b.md)" },
    want: ["docs/a.md: broken relative link b.md"],
  },
  { name: "a link that resolves passes", files: { "docs/a.md": "see [b](b.md)", "docs/b.md": "" }, want: [] },
  { name: "a link that climbs out of docs/ and resolves passes", files: { "docs/x/a.md": "[r](../../README.md)", "README.md": "" }, want: [] },
  { name: "link syntax inside inline code is ignored (2026-09-23)", files: { "docs/a.md": "expands `@[label](path)` here" }, want: [] },
  { name: "link syntax inside a fenced block is ignored", files: { "docs/a.md": "```\n[a](missing.md)\n```" }, want: [] },
  { name: "urls, absolute paths and anchors are not checked", files: { "docs/a.md": "[u](https://x.y/z) [p](/abs) [h](#top)" }, want: [] },
  { name: "the generated board is exempt", files: { "docs/a.md": "[board](handoff.md)" }, want: [] },
  { name: "an anchor on a missing file still fails", files: { "docs/a.md": "[b](b.md#part)" }, want: ["broken relative link b.md"] },
];
for (const c of linkCases) {
  test("links: " + c.name, () => {
    const got = failures(c.files);
    assert.equal(got.length, c.want.length, got.join("\n"));
    c.want.forEach((w, i) => assert.ok(got[i].includes(w), `${got[i]} should include ${w}`));
  });
}

test("architecture: an unlinked subsystem file and a dangling index link both fail", () => {
  const got = failures({
    "docs/architecture.md": "[a](architecture/a.md) [gone](architecture/gone.md)",
    "docs/architecture/a.md": "",
    "docs/architecture/b.md": "",
  });
  assert.ok(got.includes("docs/architecture/b.md is not linked from docs/architecture.md"), got.join("\n"));
  assert.ok(got.includes("docs/architecture.md links architecture/gone.md, which does not exist"), got.join("\n"));
});

test("decisions: a missing row and an accepted ADR shown as proposed both fail", () => {
  const got = failures({
    "docs/decisions/README.md": "| [0001](0001-a.md) | A | Proposed |\n",
    "docs/decisions/0001-a.md": "- **Status:** Accepted\n",
    "docs/decisions/0002-b.md": "- **Status**: Proposed\n",
  });
  assert.ok(got.some((f) => f.startsWith("ADR 0001 is accepted in its own file but \"proposed\"")), got.join("\n"));
  assert.ok(got.some((f) => f.startsWith("ADR 0002 has no row")), got.join("\n"));
});
