import assert from "node:assert/strict";
import { test } from "node:test";
import { langOf, highlightSource, escapeHtml } from "./highlight.js";

test("langOf", () => {
  assert.equal(langOf("language-go"), "go");
  assert.equal(langOf("language-JS"), "js");
  assert.equal(langOf(""), "");
});

test("highlight go keywords", () => {
  const html = highlightSource("go", "func main() {}");
  assert.match(html, /hljs-keyword/);
  assert.match(html, /func/);
});

test("unknown lang is escaped", () => {
  const html = highlightSource("nope", "<script>");
  assert.equal(html, escapeHtml("<script>"));
});
