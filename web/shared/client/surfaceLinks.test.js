import { test } from "node:test";
import assert from "node:assert/strict";
import { surfaceLinkPlan, surfaceLinkClick } from "./surfaceLinks.js";

// The origin a surface is served from; every decision row is relative to it.
const ORIGIN = "http://127.0.0.1:7171";

test("the incident row: an external http(s) link never becomes navigation", () => {
  // In the shell it is a work-browser tab; outside it a new browser tab.
  assert.deepEqual(
    surfaceLinkPlan("https://github.com/picode/picode/unblock/1", { origin: ORIGIN, shell: true }),
    { action: "tab", url: "https://github.com/picode/picode/unblock/1" },
  );
  assert.deepEqual(
    surfaceLinkPlan("https://github.com/picode/picode/unblock/1", { origin: ORIGIN, shell: false }),
    { action: "new-tab", url: "https://github.com/picode/picode/unblock/1" },
  );
  assert.deepEqual(
    surfaceLinkPlan("http://example.com/a?b=c", { origin: ORIGIN, shell: true }),
    { action: "tab", url: "http://example.com/a?b=c" },
  );
  // Scheme case and padding never slip past the table.
  assert.deepEqual(
    surfaceLinkPlan("  HTTPS://Example.COM/x  ", { origin: ORIGIN, shell: true }),
    { action: "tab", url: "https://example.com/x" },
  );
  // Protocol-relative resolves against the page and lands outside it.
  assert.deepEqual(
    surfaceLinkPlan("//example.com/x", { origin: ORIGIN, shell: false }),
    { action: "new-tab", url: "http://example.com/x" },
  );
  // With no origin to compare against, an absolute web URL is still external.
  assert.deepEqual(surfaceLinkPlan("https://example.com/", { shell: true }), { action: "tab", url: "https://example.com/" });
});

test("in-app routes and non-web schemes keep native behavior", () => {
  for (const href of [
    "#/inbox/i-1", // hash route
    "inbox/i-1", // relative path
    "/api/apps/inbox/view", // same-origin absolute
    ORIGIN + "/browser/", // same-origin spelled out
    "mailto:owner@example.com",
    "javascript:void(0)",
    "tel:+15550100",
  ]) {
    assert.equal(surfaceLinkPlan(href, { origin: ORIGIN, shell: true }), null, href);
    assert.equal(surfaceLinkPlan(href, { origin: ORIGIN, shell: false }), null, href);
  }
  // Unparseable or missing hrefs are nobody's external link.
  for (const href of ["::::", "", "   ", null, undefined, 17]) {
    assert.equal(surfaceLinkPlan(href, { origin: ORIGIN, shell: true }), null, String(href));
  }
});

// clickOn mints the smallest DOM stand-in the guard touches.
function clickOn(href) {
  const attrs = href === undefined ? {} : { href: String(href) };
  const anchor = {
    target: "",
    rel: "",
    getAttribute: (name) => (name in attrs ? attrs[name] : null),
    closest: (selector) => (selector === "a[href]" && "href" in attrs ? anchor : null),
  };
  const event = {
    defaultPrevented: false,
    button: 0,
    target: anchor,
    preventDefault() { this.defaultPrevented = true; },
  };
  return { anchor, event };
}

test("in the shell the guard preventDefaults and hands the URL to the work browser", () => {
  const opened = [];
  const open = surfaceLinkClick({ origin: ORIGIN, shell: true, onOpenUrl: (u) => opened.push(u) });
  const { anchor, event } = clickOn("https://github.com/picode/picode/unblock/1");
  open(event);
  assert.equal(event.defaultPrevented, true); // the hosting document never moves
  assert.deepEqual(opened, ["https://github.com/picode/picode/unblock/1"]);
  assert.equal(anchor.target, ""); // the anchor itself stays out of it
});

test("outside the shell the guard sets _blank + noopener and lets the default run", () => {
  const opened = [];
  const open = surfaceLinkClick({ origin: ORIGIN, shell: false, onOpenUrl: (u) => opened.push(u) });
  const { anchor, event } = clickOn("https://github.com/picode/picode/unblock/1");
  open(event);
  assert.equal(event.defaultPrevented, false);
  assert.deepEqual(opened, []);
  assert.equal(anchor.target, "_blank");
  assert.equal(anchor.rel, "noopener noreferrer");
});

test("the guard steps aside for clicks that are not plain external-link activations", () => {
  const opened = [];
  const open = surfaceLinkClick({ origin: ORIGIN, shell: true, onOpenUrl: (u) => opened.push(u) });
  // Another door already took it (the shell's system-browser bridge answers
  // explicit target=_blank at document capture, before any surface).
  const taken = clickOn("https://example.com/x");
  taken.event.defaultPrevented = true;
  open(taken.event);
  // Middle-button and non-anchor targets are not surface-link activations.
  const middle = clickOn("https://example.com/x");
  middle.event.button = 1;
  open(middle.event);
  const stray = clickOn("https://example.com/x");
  stray.event.target = {};
  open(stray.event);
  // In-app targets stay native.
  for (const href of ["#/inbox/i-1", "mailto:owner@example.com", "/api/apps/inbox/view"]) {
    const kept = clickOn(href);
    open(kept.event);
    assert.equal(kept.event.defaultPrevented, false, href);
  }
  assert.deepEqual(opened, []);
});
