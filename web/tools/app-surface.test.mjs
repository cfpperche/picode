// #/app/<id> is a PAGE FRAME, not a canvas — since 2026-09-14 the host's
// AppSurface draws the same chrome a system route draws (docs/plans/
// app-surface-parity.md; docs/benchmarks.md carries the rule, routes.md the
// host paragraph). Asserted on the source, like the other web/tools tests:
// the shells have no DOM harness, and the failure this guards against is a
// silent return to the full-bleed canvas.
import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

const read = (p) => readFileSync(new URL(p, import.meta.url), "utf8");
const browser = read("../browser/src/components/AppSurface.jsx");
const css = read("../browser/src/styles/app.css");
const mobile = read("../mobile/src/components/AppSurface.jsx");

test("the primitives app surface is a page frame", () => {
  assert.match(browser, /className="settings-wrap"/);
  assert.match(browser, /className="settings-head"/);
  assert.match(browser, /className="settings-card app-card"/);
  // .ft-head belongs to FileTreeSurface/NativeDemoSurface, which are still
  // canvases: the app frame must not borrow (or restyle) their classes.
  assert.doesNotMatch(browser, /"ft-head|ft-head"/);
});

test("the view's tabs are an underline nav inside the card, not the old pills", () => {
  assert.match(browser, /className="app-page-tabs"/);
  assert.match(browser, /className="app-page-tab"/);
  assert.match(browser, /aria-current=\{t\.id === active \? "page" : undefined\}/);
  assert.match(browser, /className="app-tab-count"/);
  // The segmented control the app canvas used to draw in its toolbar.
  assert.doesNotMatch(browser, /app-tabs-opt|app-tabs-badge/);
  assert.doesNotMatch(css, /\.app-tabs-opt|\.app-tabs-face/);
});

test("the card owns the filter, and the card owns no padding of its own", () => {
  assert.match(browser, /className="app-search"/);
  assert.match(browser, /className="app-card-toolbar"/);
  assert.match(css, /\.app-card \{ padding: 0; overflow: visible; \}/);
  // One page scrollbar (the route pages' behaviour), not a nested canvas one.
  assert.match(css, /\.app-surface \{[^}]*overflow-y: auto;/);
});

test("the phone keeps its own AppSurface", () => {
  // The parity refactor is desktop/browser only: mobile's copy is a separate
  // file with its own full-width layout and must not have gained the frame.
  assert.doesNotMatch(mobile, /settings-card app-card|app-page-tabs/);
});
