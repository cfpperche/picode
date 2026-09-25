import assert from "node:assert/strict";
import { test } from "node:test";
import { EditorState } from "@codemirror/state";
import { ensureSyntaxTree } from "@codemirror/language";
import { markdown, markdownLanguage } from "@codemirror/lang-markdown";
import { inlineTokens, planTables, tableSkip } from "./mdLiveTables.js";

function tables(doc, active = []) {
  const state = EditorState.create({ doc, extensions: [markdown({ base: markdownLanguage })] });
  ensureSyntaxTree(state, doc.length, 5000);
  return planTables(state, new Set(active));
}

const DOC = "Intro\n\n| Name | Role | Notes |\n|:-----|:----:|------:|\n| Pi | `rpc` | a \\| b |\n| Codex |\n\nAfter\n";

test("planTables reads rows, cells, positions and alignment", () => {
  const [t] = tables(DOC);
  assert.equal(DOC.slice(t.from, t.to), "| Name | Role | Notes |\n|:-----|:----:|------:|\n| Pi | `rpc` | a \\| b |\n| Codex |");
  assert.deepEqual(t.align, ["left", "center", "right"]);
  assert.deepEqual(t.rows.map((r) => r.header), [true, false, false]);
  assert.deepEqual(t.rows.map((r) => r.cells.map((c) => c.text)), [["Name", "Role", "Notes"], ["Pi", "`rpc`", "a | b"], ["Codex", "", ""]]);
  const pi = t.rows[1].cells[0];
  assert.equal(DOC.slice(pi.from, pi.from + 2), "Pi");
  assert.equal(t.rows[2].cells[2].from, t.rows[2].from);
});

test("planTables leaves a table the cursor is in as source", () => {
  assert.equal(tables(DOC, [4]).length, 0);
  assert.equal(tables(DOC, [1, 9]).length, 1);
});

test("planTables finds tables inside quotes and lists", () => {
  assert.deepEqual(tables("> | a |\n> |---|\n> | 1 |\n").map((t) => t.quoted), [true]);
  assert.equal(tables(DOC)[0].quoted, false);
  assert.equal(tables("text\n").length, 0);
});

test("inlineTokens", () => {
  const cases = [
    ["plain", [{ type: "text", text: "plain" }]],
    ["a `c|d` b", [{ type: "text", text: "a " }, { type: "code", text: "c|d" }, { type: "text", text: " b" }]],
    ["**b** *i* _j_ ~~s~~", [{ type: "strong", text: "b" }, { type: "text", text: " " }, { type: "em", text: "i" }, { type: "text", text: " " }, { type: "em", text: "j" }, { type: "text", text: " " }, { type: "strike", text: "s" }]],
    ["[doc](a/b.md) <https://x.dev>", [{ type: "link", text: "doc", href: "a/b.md" }, { type: "text", text: " " }, { type: "link", text: "https://x.dev", href: "https://x.dev" }]],
    ["![alt](i.png)", [{ type: "text", text: "alt" }]],
    ["2 * 3 * 4", [{ type: "text", text: "2 * 3 * 4" }]],
  ];
  for (const [src, want] of cases) assert.deepEqual(inlineTokens(src), want, src);
});

test("tableSkip stops vertical motion at a table instead of jumping it", () => {
  const doc = "above\n\n| a |\n|---|\n| 1 |\n\nbelow\n";
  const state = EditorState.create({ doc, extensions: [markdown({ base: markdownLanguage })] });
  ensureSyntaxTree(state, doc.length, 5000);
  const t = planTables(state, new Set());
  const blank = doc.indexOf("\n\n") + 1;
  const below = doc.indexOf("below");
  assert.equal(tableSkip(t, blank, below, true), doc.indexOf("| a |"));
  assert.equal(tableSkip(t, below, blank, false), doc.indexOf("| 1 |"));
  assert.equal(tableSkip(t, 0, blank, true), null);
});
