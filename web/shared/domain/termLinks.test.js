import assert from "node:assert/strict";
import { test } from "node:test";
import {
  hasOpenModifier, stripLineCol, classify, findLinks, linkAt, tokenAt, underCwd, relPath, wireTermLinks,
} from "./termLinks.js";

const cwd = "/home/goat/picode";

test("hasOpenModifier is ctrl or meta only", () => {
  assert.equal(hasOpenModifier({ ctrlKey: true }), true);
  assert.equal(hasOpenModifier({ metaKey: true }), true);
  assert.equal(hasOpenModifier({ altKey: true }), false);
  assert.equal(hasOpenModifier({}), false);
});

test("stripLineCol drops :line and :line:col", () => {
  assert.equal(stripLineCol("foo.go:12"), "foo.go");
  assert.equal(stripLineCol("foo.go:1-120"), "foo.go");
  assert.equal(stripLineCol("foo.go:12:3"), "foo.go");
  assert.equal(stripLineCol("foo.go."), "foo.go");
});

test("classify decision table", () => {
  const rows = [
    { raw: "https://example.com/a", want: { kind: "http", href: "https://example.com/a" } },
    { raw: "http://localhost:8445/", want: { kind: "http" } },
    { raw: "javascript:alert(1)", want: null },
    { raw: "/etc/passwd", want: null },
    { raw: cwd + "/web/src/a.js", want: { kind: "file", path: "web/src/a.js" } },
    { raw: "~/picode/web/a.js", want: { kind: "file", path: "~/picode/web/a.js" } },
    { raw: "web/src/a.js", want: { kind: "file", path: "web/src/a.js" } },
    { raw: "./web/a.js", want: { kind: "file", path: "web/a.js" } },
    { raw: "file://" + cwd + "/README.md", want: { kind: "file", path: "README.md" } },
    { raw: "web/src/a.js:12", want: { kind: "file", path: "web/src/a.js" } },
    { raw: "../secret", want: null },
    { raw: "/", want: null },
  ];
  for (const r of rows) {
    const got = classify(r.raw, cwd);
    if (r.want === null) {
      assert.equal(got, null, r.raw);
      continue;
    }
    assert.ok(got, r.raw);
    assert.equal(got.kind, r.want.kind, r.raw);
    if (r.want.path) assert.equal(got.path, r.want.path, r.raw);
    if (r.want.href) assert.equal(got.href, r.want.href, r.raw);
  }
});

test("linkAt hits the column of a jsx path", () => {
  const line = "read web/src/components/Composer.jsx";
  const hits = findLinks(line, cwd);
  assert.ok(hits[0]);
  const col = hits[0].start + 1;
  assert.equal(linkAt(line, col, cwd).path, "web/src/components/Composer.jsx");
  assert.equal(linkAt(line, 1, cwd), null);
});

test("findLinks picks bare js and jsx names", () => {
  const hits = findLinks("see Composer.jsx and termLinks.js here", cwd);
  assert.equal(hits.some((h) => h.path === "Composer.jsx"), true);
  assert.equal(hits.some((h) => h.path === "termLinks.js"), true);
});

test("findLinks skips http when scanning paths and keeps tool-call paths", () => {
  const line = "read ~/picode/web/src/a.js  see https://example.com/x  and /etc/passwd";
  const hits = findLinks(line, cwd);
  assert.equal(hits.some((h) => h.kind === "http"), true);
  assert.equal(hits.some((h) => h.kind === "file" && h.path.startsWith("~/")), true);
  assert.equal(hits.some((h) => String(h.raw).includes("/etc/passwd")), false);
});

test("printed URL boundaries exclude prose but keep balanced URL parentheses", () => {
  const rows = [
    ["(https://cfpperche.github.io/picode/guide/missions)", "https://cfpperche.github.io/picode/guide/missions"],
    ["https://example.com/wiki/Foo_(bar)", "https://example.com/wiki/Foo_(bar)"],
    ["https://example.com/a_(b)).", "https://example.com/a_(b)"],
    ["https://example.com/a?q=1).", "https://example.com/a?q=1"],
  ];
  for (const [line, want] of rows) {
    const links = findLinks(line, cwd);
    assert.equal(links.length, 1, line);
    assert.equal(links[0].raw, want, line);
    assert.equal(links[0].href, want, line);
    assert.equal(tokenAt(line, links[0].end + 1), null, line);
  }
});

test("underCwd / relPath", () => {
  assert.equal(underCwd(cwd, cwd + "/web"), true);
  assert.equal(underCwd(cwd, "/etc/passwd"), false);
  assert.equal(relPath(cwd, cwd + "/web/a.js"), "web/a.js");
});

test("classify uses live cwd, not the start folder", () => {
  const live = "/tmp";
  assert.equal(classify("ping.txt", live).path, "ping.txt");
  assert.equal(classify(live + "/ping.txt", live).path, "ping.txt");
  assert.equal(classify(cwd + "/README.md", live), null);
  assert.equal(classify("../secret", live), null);
});

test("tokenAt hits a path even when classify would reject it", () => {
  const line = "see /tmp/ping.txt and README.md";
  const tok = tokenAt(line, line.indexOf("ping") + 1);
  assert.ok(tok);
  assert.equal(tok.raw.includes("ping.txt"), true);
  assert.equal(linkAt(line, line.indexOf("ping") + 1, cwd), null);
});

// The app hands its own opener in: Ctrl+click on a link must land where the
// human's preference says (PiCode's browser surface by default), and a bare
// terminal — no opener — keeps the platform's own behavior.
test("wireTermLinks hands an http hit to the caller's opener", async () => {
  const calls = [];
  const win = { open: (...a) => { calls.push(a); return null; }, addEventListener() {}, removeEventListener() {} };
  globalThis.window = win;
  const term = {
    options: {},
    element: null,
    buffer: { active: { getLine: () => null } },
    registerLinkProvider: () => ({ dispose() {} }),
  };
  const opened = [];
  wireTermLinks(term, () => cwd, undefined, undefined, (href) => opened.push(href));
  const ev = { ctrlKey: true, preventDefault() {}, stopImmediatePropagation() {} };
  term.options.linkHandler.activate(ev, "https://example.com/x");
  await new Promise((r) => setTimeout(r, 0));
  assert.deepEqual(opened, ["https://example.com/x"]);
  assert.deepEqual(calls, []);

  // No opener: the platform's own window.open, as before.
  const bare = { ...term, options: {} };
  wireTermLinks(bare, () => cwd, undefined, undefined);
  bare.options.linkHandler.activate(ev, "https://example.com/y");
  await new Promise((r) => setTimeout(r, 0));
  assert.equal(calls.length, 1);
  assert.equal(calls[0][0], "https://example.com/y");
  delete globalThis.window;
});
