import assert from "node:assert/strict";
import { test } from "node:test";
import { isSearchTool, searchQuery, hitsFromResult, hitsFromTool } from "./searchCards.js";

test("isSearchTool", () => {
  assert.equal(isSearchTool("web_search"), true);
  assert.equal(isSearchTool("url_context"), true);
  assert.equal(isSearchTool("bash"), false);
});

test("searchQuery", () => {
  assert.equal(searchQuery({ query: "pi coding agent" }), "pi coding agent");
  assert.equal(searchQuery({ path: "x" }), "");
});

test("hits from details.sources", () => {
  const hits = hitsFromResult({
    content: [{ type: "text", text: "ok" }],
    details: { sources: [{ title: "Pi", url: "https://pi.dev" }, { title: "Pi", url: "https://pi.dev" }] },
  });
  assert.equal(hits.length, 1);
  assert.equal(hits[0].url, "https://pi.dev");
});

test("hits from markdown fallback", () => {
  const hits = hitsFromResult({ content: [{ type: "text", text: "## Sources\n1. [Docs](https://example.com/a)" }] });
  assert.equal(hits[0].title, "Docs");
});

test("hitsFromTool reads result then detail json", () => {
  const a = hitsFromTool({ result: { details: { sources: [{ url: "https://a.example", title: "A" }] } } });
  assert.equal(a[0].title, "A");
  const b = hitsFromTool({ detail: JSON.stringify({ details: { sources: [{ url: "https://b.example", title: "B" }] } }) });
  assert.equal(b[0].title, "B");
});
