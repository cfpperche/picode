import assert from "node:assert/strict";
import { test } from "node:test";
import { EditorState, EditorSelection } from "@codemirror/state";
import { ensureSyntaxTree } from "@codemirror/language";
import { markdown, markdownLanguage } from "@codemirror/lang-markdown";
import { activeLines, headingPos, planLive } from "./mdLivePlan.js";

// syntaxTree(state) is the tree the state was created with; ensureSyntaxTree
// finishes the parse in the shared context but leaves that field alone. When
// creation ran out of its parse budget (a cold start under load) the code
// under test saw a partial tree. A no-op update carries the full parse in.
function parsed(doc) {
  const created = EditorState.create({ doc, extensions: [markdown({ base: markdownLanguage })] });
  ensureSyntaxTree(created, doc.length, 5000);
  return created.update({}).state;
}

function plan(doc, active = []) {
  const state = parsed(doc);
  const specs = planLive(state, [{ from: 0, to: doc.length }], new Set(active));
  const at = (s) => doc.slice(s.from, s.to);
  return {
    specs,
    hidden: specs.filter((s) => s.kind === "hide").map(at),
    marks: (cls) => specs.filter((s) => s.kind === "mark" && s.cls === cls).map(at),
    lines: (cls) => specs.filter((s) => s.kind === "line" && s.cls.split(" ").includes(cls)).map((s) => state.doc.lineAt(s.at).number),
    kind: (k) => specs.filter((s) => s.kind === k),
  };
}

test("headings: class per level, marks hidden off the cursor line only", () => {
  const off = plan("# Title\n\n## Sub ##\n");
  assert.deepEqual(off.lines("cm-md-h1"), [1]);
  assert.deepEqual(off.lines("cm-md-h2"), [3]);
  assert.deepEqual(off.hidden, ["# ", "## ", " ##"]);
  assert.deepEqual(plan("# Title\n", [1]).hidden, []);
});

test("inline: bold, italic, strike and code keep their text, lose their marks", () => {
  const p = plan("**b** *i* ~~s~~ `c`\n");
  assert.deepEqual(p.marks("cm-md-strong"), ["**b**"]);
  assert.deepEqual(p.marks("cm-md-em"), ["*i*"]);
  assert.deepEqual(p.marks("cm-md-strike"), ["~~s~~"]);
  assert.deepEqual(p.marks("cm-md-code"), ["`c`"]);
  assert.deepEqual(p.hidden, ["**", "**", "*", "*", "~~", "~~", "`", "`"]);
  assert.deepEqual(plan("**b**\n", [1]).hidden, []);
});

test("links show their text; the target hides off the cursor line", () => {
  const off = plan("See [the guide](docs/guide.md) and <https://x.dev> or https://y.dev\n");
  const link = off.specs.find((s) => s.kind === "mark" && s.cls === "cm-md-link");
  assert.equal(link.href, "docs/guide.md");
  assert.deepEqual(off.marks("cm-md-link"), ["the guide", "https://x.dev", "https://y.dev"]);
  assert.deepEqual(off.hidden, ["[", "](docs/guide.md)", "<", ">"]);
  const on = plan("[t](u)\n", [1]);
  assert.deepEqual(on.hidden, []);
  assert.deepEqual(on.marks("cm-md-syntax"), ["](u)"]);
  assert.deepEqual(plan("[plain]\n").marks("cm-md-link"), []);
});

test("images replace their syntax off the cursor line and sit after it on it", () => {
  const off = plan("![Logo](img/logo.svg)\n").kind("image")[0];
  assert.deepEqual([off.url, off.alt, off.replace, off.from], ["img/logo.svg", "Logo", true, 0]);
  const on = plan("![Logo](a.png)\n", [1]).kind("image")[0];
  assert.deepEqual([on.replace, on.from, on.to], [false, 14, 14]);
});

test("lists: bullets and checkboxes stand in for markers", () => {
  const p = plan("- a\n- [x] done\n- [ ] open\n1. first\n");
  assert.equal(p.kind("bullet").length, 1);
  assert.deepEqual(p.kind("task").map((t) => t.checked), [true, false]);
  assert.deepEqual(p.hidden, ["- ", "- "]);
  assert.deepEqual(p.marks("cm-md-listmark"), ["1."]);
  const on = plan("- [ ] open\n", [1]);
  assert.equal(on.kind("task").length, 0);
  assert.deepEqual(plan("- a\n  - b\n").marks("cm-md-indent"), ["  "]);
});

test("quotes and alerts", () => {
  const p = plan("> [!WARNING]\n> Careful.\n\n> quote\n");
  assert.deepEqual(p.lines("cm-md-alert-warning"), [1, 2]);
  assert.deepEqual(p.lines("cm-md-quote"), [1, 2, 4]);
  const label = p.kind("label")[0];
  assert.equal(label.text, "Warning");
  assert.equal("> [!WARNING]\n".slice(label.from, label.to), "[!WARNING]");
  assert.deepEqual(p.hidden, ["> ", "> ", "> "]);
  assert.equal(plan("> [!NOTE]\n> x\n", [1]).kind("label").length, 0);
});

test("fenced code: fences fold to a label unless the cursor is inside", () => {
  const doc = "```go\nx := 1\n```\n";
  const off = plan(doc);
  assert.deepEqual(off.lines("cm-md-codeblock"), [1, 2, 3]);
  assert.deepEqual(off.kind("label").map((l) => l.text), ["go"]);
  assert.deepEqual(off.hidden, ["```"]);
  const inside = plan(doc, [2]);
  assert.equal(inside.kind("label").length, 0);
  assert.deepEqual(inside.hidden, []);
});

test("frontmatter stays source; rules, tables and html get line classes", () => {
  const p = plan("---\ntitle: x\n---\n# Body\n\n---\n\n| a |\n|---|\n| 1 |\n\n<div>x</div>\n");
  assert.deepEqual(p.lines("cm-md-frontmatter"), [1, 2, 3]);
  assert.deepEqual(p.lines("cm-md-h1"), [4]);
  assert.deepEqual(p.lines("cm-md-hr"), [6]);
  assert.deepEqual(p.lines("cm-md-table"), [8, 9, 10]);
  assert.deepEqual(p.lines("cm-md-html"), [12]);
});

test("activeLines: every selected line, none without focus", () => {
  const state = EditorState.create({ doc: "a\nb\nc\nd", extensions: EditorState.allowMultipleSelections.of(true), selection: EditorSelection.create([EditorSelection.range(2, 5), EditorSelection.cursor(7)]) });
  assert.deepEqual([...activeLines(state, true)], [2, 3, 4]);
  assert.equal(activeLines(state, false).size, 0);
});

test("headingPos finds a #fragment with the preview's slugs", () => {
  const doc = "# Intro\n\n## **Bold** [link](x) title\n\n## Intro\n";
  const state = parsed(doc);
  assert.equal(headingPos(state, "intro"), 0);
  assert.equal(headingPos(state, "bold-link-title"), 9);
  assert.equal(headingPos(state, "#intro-1"), doc.indexOf("## Intro"));
  assert.equal(headingPos(state, "user-content-intro"), 0);
  assert.equal(headingPos(state, "missing"), -1);
});
