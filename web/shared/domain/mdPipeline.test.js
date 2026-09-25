import assert from "node:assert/strict";
import { test } from "node:test";
import { unified } from "unified";
import remarkParse from "remark-parse";
import remarkRehype from "remark-rehype";
import { docPipeline, withSourceLines } from "./mdPipeline.js";
import { splitFrontmatter } from "./mdDocument.js";

// Same wiring react-markdown does: raw HTML reaches rehype as `raw` nodes.
function render(md, pipeline = docPipeline) {
  const proc = unified().use(remarkParse).use(pipeline.remarkPlugins)
    .use(remarkRehype, { ...pipeline.remarkRehypeOptions, allowDangerousHtml: true })
    .use(pipeline.rehypePlugins);
  return proc.runSync(proc.parse(md), md);
}

function all(node, pred, out = []) {
  if (node.type === "element" && pred(node)) out.push(node);
  for (const c of node.children || []) all(c, pred, out);
  return out;
}
const byTag = (tree, tag) => all(tree, (n) => n.tagName === tag);
const cls = (n) => n.properties.className || [];

test("raw HTML survives on GitHub's allow-list", () => {
  const tree = render('<p align="center"><img src="logo.svg" width="80" alt="Logo"></p>\n\n<h1 align="center">PiCode</h1>\n\n<details><summary>More</summary>\n\nHidden\n\n</details>\n');
  const img = byTag(tree, "img")[0];
  assert.equal(img.properties.src, "logo.svg");
  assert.equal(img.properties.width, 80);
  assert.equal(byTag(tree, "p")[0].properties.align, "center");
  assert.equal(byTag(tree, "h1")[0].properties.id, "user-content-picode");
  assert.equal(byTag(tree, "summary").length, 1);
});

test("dangerous HTML is dropped", () => {
  const tree = render('<script>alert(1)</script>\n\n<img src="x" onerror="alert(1)">\n\n<a href="javascript:alert(1)">x</a>\n\n<iframe src="https://x"></iframe>\n\n<div id="app">clobber</div>\n');
  assert.equal(byTag(tree, "script").length, 0);
  assert.equal(byTag(tree, "iframe").length, 0);
  assert.equal(byTag(tree, "img")[0].properties.onError, undefined);
  assert.equal(byTag(tree, "a")[0].properties.href, undefined);
  assert.equal(all(tree, (n) => n.properties.id === "app").length, 0);
});

test("alerts, math, footnotes and headings after sanitizing", () => {
  const tree = render("# Intro\n\n> [!TIP]\n> Use it.\n\nInline $a^2$ and a note[^1].\n\n$$\nx\n$$\n\n## Intro\n\n[^1]: The note.\n");
  const alert = all(tree, (n) => cls(n).includes("md-alert"))[0];
  assert.deepEqual(cls(alert), ["md-alert", "md-alert-tip"]);
  assert.equal(all(tree, (n) => cls(n).includes("katex")).length >= 1, true);
  assert.equal(all(tree, (n) => cls(n).includes("katex-display")).length, 1);
  const ids = all(tree, (n) => /^h[1-6]$/.test(n.tagName)).map((n) => n.properties.id);
  assert.deepEqual(ids.slice(0, 2), ["user-content-intro", "user-content-intro-1"]);
  const ref = all(tree, (n) => n.tagName === "a" && n.properties.dataFootnoteRef !== undefined)[0];
  const target = ref.properties.href.slice(1);
  assert.equal(all(tree, (n) => n.properties.id === "user-content-" + target).length, 1);
});

test("withSourceLines stamps blocks with the file's own line numbers", () => {
  const file = "---\ntitle: x\n---\n# Title\n\ntext\n\n> [!NOTE]\n> hi\n\n- a\n- b\n\n```go\nx\n```\n\n| a |\n|---|\n| 1 |\n";
  const { body, bodyLine } = splitFrontmatter(file);
  assert.equal(bodyLine, 4);
  const tree = render(body, withSourceLines(bodyLine - 1));
  const line = (pred) => all(tree, pred)[0].properties.dataLine;
  assert.equal(line((n) => n.tagName === "h1"), 4);
  assert.equal(line((n) => n.tagName === "p"), 6);
  assert.equal(line((n) => cls(n).includes("md-alert")), 8);
  assert.deepEqual(byTag(tree, "li").map((n) => n.properties.dataLine), [11, 12]);
  assert.equal(line((n) => n.tagName === "pre"), 14);
  assert.deepEqual(byTag(tree, "tr").map((n) => n.properties.dataLine), [18, 20]);
  assert.equal(all(render(body), (n) => n.properties.dataLine !== undefined).length, 0);
});

test("prices stay text; real inline math still renders", () => {
  const texts = (md) => all(render(md), (n) => (n.properties.className || []).includes("katex")).length;
  assert.equal(texts("it costs $5 and $10."), 0);
  assert.equal(texts("a $ x$ b and $y $ c"), 0);
  assert.equal(texts("Inline $E = mc^2$ here"), 1);
  assert.equal(texts("two $a$ and $b_1$"), 2);
});
