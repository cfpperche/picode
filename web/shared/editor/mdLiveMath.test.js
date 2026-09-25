import assert from "node:assert/strict";
import { test } from "node:test";
import { EditorState } from "@codemirror/state";
import { ensureSyntaxTree } from "@codemirror/language";
import { markdown, markdownLanguage } from "@codemirror/lang-markdown";
import { planInlineMath, planMathBlocks, planMermaid } from "./mdLiveMath.js";

// syntaxTree(state) is the tree the state was created with; ensureSyntaxTree
// finishes the parse in the shared context but leaves that field alone. When
// creation ran out of its parse budget (a cold start under load) the code
// under test saw a partial tree. A no-op update carries the full parse in.
function parsed(doc) {
  const created = EditorState.create({ doc, extensions: [markdown({ base: markdownLanguage })] });
  ensureSyntaxTree(created, doc.length, 5000);
  return created.update({}).state;
}

function st(doc) {
  return parsed(doc);
}

test("planMathBlocks: multi-line, one-line, unclosed, code and cursor", () => {
  const doc = "a\n\n$$\n\\int_0^1 x\\,dx\n$$\n\n$$ E = mc^2 $$\n\n```\n$$\nnot math\n$$\n```\n\n$$\nopen\n";
  const blocks = planMathBlocks(st(doc), new Set());
  assert.deepEqual(blocks.map((b) => b.tex), ["\\int_0^1 x\\,dx", "E = mc^2"]);
  assert.equal(doc.slice(blocks[0].from, blocks[0].to), "$$\n\\int_0^1 x\\,dx\n$$");
  assert.equal(doc.slice(blocks[0].lastFrom, blocks[0].to), "$$");
  assert.deepEqual(planMathBlocks(st(doc), new Set([4])).map((b) => b.tex), ["E = mc^2"]);
});

test("planMermaid: only mermaid fences, closed, off the cursor", () => {
  const doc = "```mermaid\ngraph LR\n  A --> B\n```\n\n```go\nx\n```\n\n```mermaid\nopen\n";
  const [m, ...rest] = planMermaid(st(doc), new Set());
  assert.equal(rest.length, 0);
  assert.equal(m.src, "graph LR\n  A --> B");
  assert.equal(doc.slice(m.from, m.to), "```mermaid\ngraph LR\n  A --> B\n```");
  assert.equal(planMermaid(st(doc), new Set([2])).length, 0);
});

test("planInlineMath follows pandoc's dollar rules", () => {
  const cases = [
    ["Inline $E = mc^2$ here", ["E = mc^2"]],
    ["costs $5 and $10", []],
    ["a $x$5 b", []],
    ["a $ x$ b", []],
    ["a $x $ b", []],
    ["escaped \\$x$ no", []],
    ["two $a$ and $b_1$", ["a", "b_1"]],
    ["`$code$` and $m$", ["m"]],
    ["[$l$](u$x$)", ["l"]],
  ];
  for (const [line, want] of cases) {
    assert.deepEqual(planInlineMath(st(line), [{ from: 0, to: line.length }], new Set()).map((m) => m.tex), want, line);
  }
  const on = planInlineMath(st("x $a$\n"), [{ from: 0, to: 6 }], new Set([1]));
  assert.deepEqual(on.map((m) => m.replace), [false]);
});
