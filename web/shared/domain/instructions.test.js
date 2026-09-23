import { test } from "node:test";
import assert from "node:assert/strict";
import { cellTitle, createLine, groupFiles, rowMatters, sizeLabel, visibleClis } from "./instructions.js";

const clis = [
  { id: "claude-code", name: "Claude Code", installed: true },
  { id: "codex", name: "Codex", installed: true },
  { id: "grok", name: "Grok", installed: false },
];

function file(path, cells, extra = {}) {
  return { path, scope: "project", folder: "", cells, ...extra };
}

test("cellTitle names the winner, the on-demand rule and the cut", () => {
  assert.equal(cellTitle({ status: "shadowed", by: "CLAUDE.md", why: "Claude Code reads AGENTS.md only where no CLAUDE.md exists" }), "CLAUDE.md wins. Claude Code reads AGENTS.md only where no CLAUDE.md exists");
  assert.match(cellTitle({ status: "on-demand", why: "x" }), /^Read once the agent works in that folder\. x$/);
  assert.equal(cellTitle({ status: "reads", why: "y", cut: "Hermes cuts it on models with a window under 88k tokens" }), "y Hermes cuts it on models with a window under 88k tokens.");
  assert.equal(cellTitle(undefined), "");
});

test("groupFiles keeps the order and drops empty groups", () => {
  const groups = groupFiles([file("AGENTS.md", {}), { path: "~/.claude/CLAUDE.md", scope: "personal", cells: {} }]);
  assert.deepEqual(groups.map((g) => g.id), ["project", "personal"]);
  assert.equal(groups[0].title, "This workspace");
});

test("visibleClis shows the installed ones, all on request, all when none is installed", () => {
  assert.deepEqual(visibleClis(clis).map((c) => c.id), ["claude-code", "codex"]);
  assert.equal(visibleClis(clis, true).length, 3);
  assert.equal(visibleClis([{ id: "grok", installed: false }]).length, 1);
});

test("rowMatters hides a row every visible CLI ignores", () => {
  const cols = visibleClis(clis);
  assert.equal(rowMatters(file("GEMINI.md", { "claude-code": { status: "not-read" }, codex: { status: "not-read" }, grok: { status: "reads" } }), cols), false);
  assert.equal(rowMatters(file("AGENTS.md", { "claude-code": { status: "shadowed" } }), cols), true);
});

test("sizeLabel", () => {
  assert.equal(sizeLabel(900), "900 B");
  assert.equal(sizeLabel(21149), "21 KB");
  assert.equal(sizeLabel(5000), "4.9 KB");
});

// One row per case the Create dialog can meet (decision table).
const report = (files) => ({ clis, files });
const lineCases = [
  {
    name: "reads AGENTS.md",
    files: [file("AGENTS.md", { codex: { status: "reads" } })],
    cli: "codex",
    want: "Codex reads AGENTS.md.",
  },
  {
    name: "CLAUDE.md wins and AGENTS.md is left out",
    files: [file("AGENTS.md", { "claude-code": { status: "shadowed" } }), file("CLAUDE.md", { "claude-code": { status: "reads" } })],
    cli: "claude-code",
    want: "Claude Code reads CLAUDE.md; AGENTS.md is left out.",
  },
  {
    name: "an untrusted folder says so first",
    files: [file("AGENTS.md", { grok: { status: "untrusted" } })],
    cli: "grok",
    want: "Grok ignores this folder's instructions until you trust it in Grok.",
  },
  { name: "no file at all", files: [], cli: "codex", want: "No instruction file in this workspace." },
  {
    name: "files, but none this CLI reads",
    files: [file("GEMINI.md", { codex: { status: "not-read" } })],
    cli: "codex",
    want: "Codex reads none of this workspace's instruction files.",
  },
  {
    name: "a subfolder's file is not the start folder's",
    files: [file("pkg/AGENTS.md", { codex: { status: "reads" } }, { folder: "pkg" })],
    cli: "codex",
    want: "No instruction file in this workspace.",
  },
  { name: "an unknown CLI says nothing", files: [file("AGENTS.md", {})], cli: "nope", want: "" },
];
for (const c of lineCases) {
  test("createLine: " + c.name, () => {
    assert.equal(createLine(report(c.files), c.cli), c.want);
  });
}
