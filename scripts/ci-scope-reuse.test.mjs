import assert from "node:assert/strict";
import test from "node:test";
import { coveredRoots, decideReuse, packageDir } from "./ci-scope-reuse.mjs";

// ADR-0124: `make close` may skip the scoped gates only when the content the
// gates read is provably unchanged. Every row of that decision is here; the
// git plumbing around it is one `git ls-tree` call.
const stamp = {
  base: "aaaa1111",
  tree: "tree-a",
  at: "2026-09-12T10:00:00.000Z",
  count: 3,
  covered: "hash-a",
  roots: ["internal/server/", "go.mod"],
};

test("no recorded run means the gates run", () => {
  const v = decideReuse({ stamp: null, head: "tree-b", covered: "hash-b", dirty: false });
  assert.equal(v.reuse, false);
  assert.match(v.reason, /no green ci-scoped run/);
});

test("a dirty tree never reuses a green run", () => {
  const v = decideReuse({ stamp, head: "tree-a", covered: "hash-a", dirty: true });
  assert.equal(v.reuse, false);
  assert.match(v.reason, /dirty/);
});

test("the same tree is green, whatever the stamps say", () => {
  const v = decideReuse({ stamp, head: "tree-a", covered: "hash-a", dirty: false });
  assert.equal(v.reuse, true);
  assert.match(v.reason, /tree is unchanged/);
});

test("a catch-up merge that left the covered content alone reuses the run", () => {
  const v = decideReuse({ stamp, head: "tree-b", covered: "hash-a", dirty: false });
  assert.equal(v.reuse, true);
  assert.match(v.reason, /3 covered path/);
});

test("a covered path whose content changed re-runs, even at the same path", () => {
  const v = decideReuse({ stamp, head: "tree-b", covered: "hash-b", dirty: false });
  assert.equal(v.reuse, false);
  assert.match(v.reason, /covered changed content/);
});

test("covered roots follow the scope the run actually exercised", () => {
  const web = coveredRoots({ paths: ["web/browser/src/App.jsx"] });
  assert.ok(web.includes("web/"));
  assert.ok(web.includes("web/browser/src/App.jsx"));

  const go = coveredRoots({ paths: ["internal/server/routes.go"], packages: ["internal/store", "."] });
  assert.ok(go.includes("go.mod") && go.includes("go.sum"));
  assert.ok(go.includes("internal/store/"));
  assert.ok(go.includes("."), "the module root package depends on everything");
  assert.ok(!go.includes("internal/"), "the go gate covers the tested closure, not the whole tree");

  const docs = coveredRoots({ paths: ["docs-site/guide/settings.md"] });
  assert.ok(docs.includes("docs-site/") && docs.includes(".vale.ini") && docs.includes("cmd/"));
  assert.ok(!docs.includes("docs/"), "docs/ stays out of the covered roots (make close runs its pass every time) — a main merge that only moved a note must reuse");

  const metadata = coveredRoots({ paths: ["docs/handoff/2026-09-12-x.md"] });
  assert.deepEqual(metadata, ["docs/handoff/2026-09-12-x.md"], "a note covers itself, nothing else");

  const desktop = coveredRoots({ paths: ["desktop-shell/src/btab.rs"] });
  assert.ok(desktop.includes("desktop-shell/"), "both shell gates read the whole crate");
  assert.ok(desktop.includes("desktop-shell/src/btab.rs"));

  const webBranch = coveredRoots({ paths: ["web/browser/src/App.jsx"] });
  assert.ok(webBranch.includes("cmd/"), "`make build` compiles the binary whatever changed");

  const full = coveredRoots({ paths: ["Makefile"] });
  assert.deepEqual(full, ["."], "gate-shaping paths cover the whole tree");
});

test("packageDir strips the module prefix, and the root package stays conservative", () => {
  assert.equal(packageDir("github.com/cfpperche/picode/internal/server", "github.com/cfpperche/picode"), "internal/server");
  assert.equal(packageDir("github.com/cfpperche/picode", "github.com/cfpperche/picode"), ".");
  assert.equal(packageDir("example.com/other/pkg", "github.com/cfpperche/picode"), "example.com/other/pkg");
});
