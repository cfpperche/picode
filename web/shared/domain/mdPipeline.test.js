import assert from "node:assert/strict";
import { test } from "node:test";
import { unified } from "unified";
import remarkParse from "remark-parse";
import remarkRehype from "remark-rehype";
import { docPipeline } from "./mdPipeline.js";

// Same wiring react-markdown does: raw HTML reaches rehype as `raw` nodes.
function render(md) {
  const proc = unified().use(remarkParse).use(docPipeline.remarkPlugins)
    .use(remarkRehype, { ...docPipeline.remarkRehypeOptions, allowDangerousHtml: true })
    .use(docPipeline.rehypePlugins);
  return proc.runSync(proc.parse(md));
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
