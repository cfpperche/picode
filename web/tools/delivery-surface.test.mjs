// The Git tab's Delivery view is a PAGE FRAME, not a canvas — since
// 2026-09-21 it draws the same chrome a system route draws (docs/benchmarks.md
// carries the rule, docs/architecture/delivery.md the surface paragraph), the
// Agent CLIs standard of ADR-0103. Asserted on the source, like the other
// web/tools tests: the shells have no DOM harness, and the failure this guards
// against is a silent return to the full-bleed grey list.
import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

const read = (p) => readFileSync(new URL(p, import.meta.url), "utf8");
const browser = read("../browser/src/components/Delivery.jsx");
const css = read("../browser/src/styles/delivery.css");
const gitSurface = read("../browser/src/components/GitGraphSurface.jsx");
const mobile = read("../mobile/src/screens/GitDelivery.jsx");

test("the Delivery view is a page frame", () => {
  assert.match(browser, /className="settings-wrap"/);
  assert.match(browser, /className="settings-head"/);
  assert.match(browser, /className="settings-card"/);
  // One page scrollbar, not a padded canvas that scrolls inside the tab.
  assert.match(css, /\.delivery-page \{[^}]*overflow-y: auto;/);
  assert.doesNotMatch(css, /^\.delivery \{/m);
});

test("the Git tab's shared strip stays outside the frame", () => {
  // History is still a canvas: the workspace picker and the History/Delivery
  // toggle must not move into Delivery's card, or the graph shifts with them.
  assert.match(gitSurface, /<header className="delivery-tabs">/);
  assert.doesNotMatch(browser, /delivery-tabs/);
  assert.match(css, /\.delivery-tabs \{[^}]*border-bottom: 1px solid var\(--border\);/);
});

test("empty and blocked states are one line + one action inside the card", () => {
  assert.match(browser, /className="mcp-empty"/);
  assert.doesNotMatch(browser, /delivery-empty/);
  assert.doesNotMatch(css, /\.delivery-empty/);
});

test("the phone keeps its own Delivery section", () => {
  // The page frame is desktop/browser only (ADR-0122's one bundle); mobile is
  // full-width by design (ADR-0072) and owns a separate component.
  assert.doesNotMatch(mobile, /settings-wrap|settings-card/);
});
