import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";

import { SUGGEST_LIMIT, rankVisits, splitAddress } from "./addressHistory.js";

const visit = (url, { typed = false, title = "", id = 0 } = {}) => ({
  id,
  url,
  title,
  host: splitAddress(url).host,
  typed,
});

test("a typed URL outranks a page you only landed on", () => {
  // The spec's sentence is literal: "typed URLs first". Newest-first inside
  // each group; nothing here invents a relevance score.
  const rows = rankVisits([
    visit("https://landed.example/", { id: 1 }),
    visit("https://typed.example/", { typed: true, id: 2 }),
    visit("https://landed2.example/", { id: 3 }),
    visit("https://typed2.example/", { typed: true, id: 4 }),
  ]);
  assert.deepEqual(
    rows.map((r) => r.url),
    ["https://typed.example/", "https://typed2.example/", "https://landed.example/", "https://landed2.example/"],
  );
});

test("the dropdown forgets nothing it should, and repeats nothing", () => {
  const rows = rankVisits([
    visit("https://a.example/page", { id: 1 }),
    visit("https://a.example/page", { typed: true, id: 2 }), // same page, re-visited
    visit("https://a.example/page/", { id: 3 }), // and once more with the slash
    visit("https://b.example/", { id: 4 }),
  ]);
  assert.deepEqual(rows.map((r) => r.url), ["https://a.example/page", "https://b.example/"], "one row per page");
  assert.equal(rows[0].typed, false, "the newest visit wins, and it was not typed");
});

test("the page you are standing on is not a suggestion", () => {
  const rows = rankVisits([visit("https://here.example/", { id: 1 }), visit("https://there.example/", { id: 2 })], {
    skipUrl: "https://here.example",
  });
  assert.deepEqual(rows.map((r) => r.url), ["https://there.example/"]);
});

test("a query narrows by url, title or host", () => {
  const rows = [
    visit("https://github.com/cfpperche/picode", { title: "PiCode", id: 1 }),
    visit("https://news.example/", { title: "The Picode Times", id: 2 }),
    visit("https://other.example/", { title: "nothing", id: 3 }),
  ];
  assert.deepEqual(rankVisits(rows, { query: "github" }).map((r) => r.url), [rows[0].url], "host and path match");
  assert.deepEqual(rankVisits(rows, { query: "picode" }).map((r) => r.url), [rows[0].url, rows[1].url], "titles match");
  assert.deepEqual(rankVisits(rows, { query: "  GITHUB  " }).map((r) => r.url), [rows[0].url], "case and spacing are noise");
  assert.deepEqual(rankVisits(rows, { query: "zzz" }), []);
});

test("the pane with no desktop shell only offers what it can open", () => {
  const rows = [
    visit("http://localhost:5173/", { id: 1 }),
    visit("https://github.com/", { id: 2 }),
    visit("http://127.0.0.1:8445/app", { id: 3 }),
  ];
  assert.deepEqual(
    rankVisits(rows, { localOnly: true }).map((r) => r.url),
    ["http://localhost:5173/", "http://127.0.0.1:8445/app"],
  );
  assert.equal(rankVisits(rows).length, 3, "the desktop pane keeps them all");
});

test("junk never becomes a row, and the limit holds", () => {
  const rows = rankVisits([null, {}, { url: "   " }, visit("https://a.example/", { id: 1 })]);
  assert.deepEqual(rows.map((r) => r.url), ["https://a.example/"]);
  const many = Array.from({ length: 20 }, (_, i) => visit(`https://s${i}.example/`, { id: i }));
  assert.equal(rankVisits(many).length, SUGGEST_LIMIT);
  assert.equal(rankVisits(many, { limit: 3 }).length, 3);
  assert.equal(rankVisits(many, { limit: 0 }).length, SUGGEST_LIMIT, "a nonsense limit falls back to the glance");
  assert.deepEqual(rankVisits("not an array"), []);
});

test("splitAddress keeps the host loud and the rest quiet", () => {
  assert.deepEqual(splitAddress("https://github.com/cfpperche/picode?tab=repos#top"), {
    host: "github.com",
    rest: "/cfpperche/picode?tab=repos#top",
  });
  assert.deepEqual(splitAddress("https://github.com/"), { host: "github.com", rest: "" });
  assert.deepEqual(splitAddress("not a url"), { host: "not a url", rest: "" });
});

test("the suggestions are a layer the app's own vocabulary can see", () => {
  // A WebView2 paints over any HTML in its region, so the dropdown must be a
  // floating layer in the shared vocabulary — and the vocabulary's rule is
  // role + state, not a class of the day. Losing this would make the list
  // invisible over a live page (the exact failure the annotate work paid for).
  const surface = readFileSync(new URL("../components/WebTabAddress.jsx", import.meta.url), "utf8");
  assert.ok(surface.includes('role="listbox"'), "the dropdown declares a listbox");
  assert.ok(surface.includes('data-state={open ? "open" : "closed"}'), "and its open state, which is what the vocabulary matches");
});

test("the address bar no longer promises a search it cannot do", () => {
  // "Search or enter a URL" was the placeholder while the shell only prefixes
  // https:// — typing a phrase produced an invalid-URL error. Search is a
  // decision of its own (which provider, whose eyes on the query); until it
  // exists, the copy must not promise it (owner 2026-09-19).
  for (const file of ["../components/WebTabAddress.jsx", "../components/WebTab.jsx"]) {
    const surface = readFileSync(new URL(file, import.meta.url), "utf8");
    assert.ok(!surface.includes("Search or enter a URL"), `${file} must not promise search`);
  }
});
