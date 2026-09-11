import test from "node:test";
import assert from "node:assert/strict";
import {
  SUPPORTED_API,
  normalizeManifests,
  supportedApp,
  nativeApp,
  normalizeView,
  aggregateBadge,
  activeAppTab,
} from "./appPrimitives.js";

test("group controls remain separate from container counts and preserve confirmation", () => {
  const view = normalizeView({ apiVersion: 1, blocks: [{ type: "list", id: "project:demo", title: "demo", collapsible: true, items: [{ id: "qa", title: "database" }], actions: [{ id: "review", label: "Review", confirm: "Review exact targets?", args: { project: "demo" } }] }] });
  assert.equal(view.blocks[0].items.length, 1);
  assert.equal(view.blocks[0].actions[0].confirm, "Review exact targets?");
  assert.deepEqual(view.blocks[0].actions[0].args, { project: "demo" });
});

test("nested App routes keep their parent tab selected", () => {
  const tabs = [{ id: "containers", path: "" }, { id: "resources", path: "resources" }, { id: "health", path: "health" }, { id: "history", path: "history" }];
  for (const [path, expected] of [["", "containers"], ["project/qa", "containers"], ["health/diagnosis/qa", "health"], ["resources/image/qa", "resources"], ["history/job/qa", "history"]]) assert.equal(activeAppTab(tabs, path), expected);
});

test("only identified, named list blocks become collapsible groups", () => {
  const v = normalizeView({ apiVersion: 1, blocks: [
    { type: "list", id: "one", title: "Project", collapsible: true },
    { type: "list", title: "No identity", collapsible: true },
    { type: "list", id: "two", collapsible: true },
    { type: "detail", id: "three", title: "Details", text: "output", collapsible: true },
  ] });
  assert.equal(v.blocks[0].id, "one");
  assert.equal(v.blocks[0].collapsible, true);
  assert.equal(v.blocks[1].collapsible, false);
  assert.equal(v.blocks[2].collapsible, false);
  assert.equal(v.blocks[3].collapsible, undefined);
});

test("runtime output remains literal and empty lists retain their state text", () => {
  const v = normalizeView({ apiVersion: 1, blocks: [
    { type: "detail", text: "<script>bad()</script> **not Markdown**", busy: true },
    { type: "detail", text: "" },
    { type: "list", empty: "No containers.", items: [{ id: "a", title: "A", busy: true }] },
  ] });
  assert.equal(v.blocks[0].text, "<script>bad()</script> **not Markdown**");
  assert.equal(v.blocks[0].markdown, undefined);
  assert.equal(v.blocks[0].busy, true);
  assert.equal(v.blocks[1].text, "");
  assert.equal(v.blocks[2].empty, "No containers.");
  assert.equal(v.blocks[2].items[0].busy, true);
});

test("normalizeManifests keeps valid rows, drops junk", () => {
  const out = normalizeManifests({
    apps: [
      { id: "demo", name: "Demo", icon: "flask", apiVersion: 1, badge: { count: 3 } },
      { id: "", name: "NoId" },
      { name: "NoIdAtAll" },
      null,
      { id: "future", name: "Future", apiVersion: 9, badge: { dot: true } },
    ],
  });
  assert.equal(out.length, 2);
  assert.deepEqual(out[0], {
    id: "demo", name: "Demo", icon: "flask", apiVersion: 1, surface: "", badge: { count: 3, dot: false },
  });
  assert.equal(out[1].apiVersion, 9);
  assert.equal(out[1].badge.dot, true);
});

test("normalizeManifests keeps the surface kind verbatim (ADR-0109)", () => {
  const out = normalizeManifests({ apps: [
    { id: "inbox", name: "Inbox", apiVersion: 1 },
    { id: "canvas", name: "Canvas", apiVersion: 1, surface: "native" },
    { id: "odd", name: "Odd", apiVersion: 1, surface: "hologram" },
    { id: "num", name: "Num", apiVersion: 1, surface: 7 },
  ] });
  assert.deepEqual(out.map((m) => m.surface), ["", "native", "hologram", "7"]);
  assert.equal(nativeApp(out[1]), true);
  assert.equal(nativeApp(out[0]), false);
  assert.equal(nativeApp(null), false);
});

test("normalizeManifests survives junk payloads", () => {
  assert.deepEqual(normalizeManifests(null), []);
  assert.deepEqual(normalizeManifests({}), []);
  assert.deepEqual(normalizeManifests({ apps: "nope" }), []);
});

test("supportedApp gates on apiVersion", () => {
  assert.equal(supportedApp({ apiVersion: SUPPORTED_API }), true);
  assert.equal(supportedApp({ apiVersion: 99 }), false);
  assert.equal(supportedApp(null), false);
});

// The decision table of ADR-0109, one assertion per row: the registry a
// shell passes is a Set of the app ids it compiled in.
test("supportedApp gates on the surface a shell can draw (ADR-0109)", () => {
  const desktop = new Set(["canvas"]);
  const primitives = { id: "inbox", name: "Inbox", apiVersion: SUPPORTED_API, surface: "" };
  const native = { id: "canvas", name: "Canvas", apiVersion: SUPPORTED_API, surface: "native" };
  // surface "" (primitives): enabled on any shell, registry or not.
  assert.equal(supportedApp(primitives, desktop), true);
  assert.equal(supportedApp(primitives), true);
  assert.equal(supportedApp(primitives, new Set()), true);
  // surface "native", id registered (desktop): enabled.
  assert.equal(supportedApp(native, desktop), true);
  // surface "native", id not registered (an older desktop bundle): unsupported.
  assert.equal(supportedApp(native, new Set(["other"])), false);
  assert.equal(supportedApp(native, new Set()), false);
  // surface "native" on a shell with no registry at all (mobile): unsupported.
  assert.equal(supportedApp(native), false);
  assert.equal(supportedApp(native, null), false);
  // surface other value: unsupported everywhere, even when the id is registered.
  assert.equal(supportedApp({ ...native, surface: "hologram" }, desktop), false);
  assert.equal(supportedApp({ ...primitives, surface: "7" }), false);
  // apiVersion ≠ 1: unsupported as today, registry or not.
  assert.equal(supportedApp({ ...native, apiVersion: 2 }, desktop), false);
  assert.equal(supportedApp({ ...primitives, apiVersion: 0 }), false);
  // A manifest that predates the field (no surface key) is a primitives app.
  assert.equal(supportedApp({ id: "old", apiVersion: SUPPORTED_API }, desktop), true);
});

test("normalizeView refuses unsupported versions", () => {
  assert.equal(normalizeView(null), null);
  assert.equal(normalizeView({ apiVersion: 2, blocks: [] }), null);
});

test("normalizeView cleans blocks", () => {
  const v = normalizeView({
    apiVersion: 1,
    title: "T",
    blocks: [
      { type: "video", src: "x" },
      { type: "detail", markdown: "hi" },
      { type: "detail" },
      { type: "list", items: [
        { id: "a", title: "A", path: "item/a", actions: [{ id: "go", label: "Go" }, { id: "" }] },
        { id: "", title: "junk" },
      ] },
      { type: "form", form: { id: "f", fields: [
        { name: "x", method: "input" },
        { name: "y", method: "slider" },
        { method: "input" },
      ] } },
      { type: "form", form: { fields: [] } },
      { type: "actions", actions: [{ id: "ok", label: "OK", danger: true }] },
      { type: "actions", actions: [] },
    ],
  });
  assert.equal(v.title, "T");
  assert.deepEqual(v.blocks.map((b) => b.type), ["detail", "list", "form", "actions"]);
  assert.equal(v.blocks[1].items.length, 1);
  assert.equal(v.blocks[1].items[0].actions.length, 1);
  assert.equal(v.blocks[2].form.fields.length, 1);
  assert.equal(v.blocks[3].actions[0].danger, true);
});

test("aggregateBadge sums counts; dot only without counts", () => {
  assert.deepEqual(aggregateBadge([]), { count: 0, dot: false });
  assert.deepEqual(
    aggregateBadge([{ badge: { count: 2 } }, { badge: { count: 1, dot: true } }]),
    { count: 3, dot: false },
  );
  assert.deepEqual(
    aggregateBadge([{ badge: { count: 0, dot: true } }]),
    { count: 0, dot: true },
  );
  assert.deepEqual(aggregateBadge(null), { count: 0, dot: false });
});

test("normalizeView keeps the split layout and pane hints", () => {
  const v = normalizeView({
    apiVersion: 1,
    title: "Inbox",
    layout: "split",
    blocks: [
      { type: "list", pane: "list", title: "Needs you", items: [{ id: "a", title: "A" }] },
      { type: "detail", pane: "detail", title: "A", meta: ["agent", "needs input"], at: "2026-09-01T00:00:00Z", markdown: "body" },
    ],
  });
  assert.equal(v.layout, "split");
  assert.equal(v.blocks[0].pane, "list");
  assert.equal(v.blocks[0].title, "Needs you");
  assert.deepEqual(v.blocks[1].meta, ["agent", "needs input"]);
  assert.equal(v.blocks[1].at, "2026-09-01T00:00:00Z");
  // Unknown layouts and panes degrade to the stacked default.
  assert.equal(normalizeView({ apiVersion: 1, layout: "carousel", blocks: [] }).layout, "");
  const oddPane = normalizeView({ apiVersion: 1, blocks: [{ type: "detail", pane: "middle", markdown: "x" }] });
  assert.equal(oddPane.blocks[0].pane, "");
});

test("normalizeView preserves the row fields a dense list needs", () => {
  const v = normalizeView({
    apiVersion: 1,
    blocks: [{ type: "list", items: [{
      id: "a", title: "A",
      meta: ["helper", "needs input", 7],
      at: "2026-09-01T00:00:00Z",
      tone: "warn", unread: true, badge: "question",
      actions: [{ id: "done", label: "Done", icon: "check" }],
    }] }],
  });
  const row = v.blocks[0].items[0];
  assert.deepEqual(row.meta, ["helper", "needs input"]); // non-strings dropped
  assert.equal(row.at, "2026-09-01T00:00:00Z");
  assert.equal(row.tone, "warn");
  assert.equal(row.unread, true);
  assert.equal(row.badge, "question");
  assert.equal(row.actions[0].icon, "check");
  // An unknown tone falls back to neutral rather than leaking through.
  const odd = normalizeView({ apiVersion: 1, blocks: [{ type: "list", items: [{ id: "a", title: "A", tone: "chartreuse" }] }] });
  assert.equal(odd.blocks[0].items[0].tone, "");
  assert.equal(odd.blocks[0].items[0].unread, false);
});

test("normalizeView preserves tabs, drops junk ones, caps the count", () => {
  const v = normalizeView({
    apiVersion: 1,
    tabs: [
      { id: "active", label: "Active", path: "" },
      { id: "done", label: "Done", path: "done", badge: "3" },
      { id: "no-label" },
      { label: "no-id" },
      7,
      null,
    ],
    blocks: [],
  });
  assert.equal(v.tabs.length, 2);
  assert.deepEqual(v.tabs[0], { id: "active", label: "Active", path: "", badge: "" });
  assert.deepEqual(v.tabs[1], { id: "done", label: "Done", path: "done", badge: "3" });

  // Missing tabs field entirely degrades to an empty array, not a crash.
  assert.deepEqual(normalizeView({ apiVersion: 1, blocks: [] }).tabs, []);

  // Caps at a sane maximum even if a server sends more.
  const many = normalizeView({
    apiVersion: 1,
    tabs: Array.from({ length: 20 }, (_, i) => ({ id: "t" + i, label: "T" + i })),
    blocks: [],
  });
  assert.equal(many.tabs.length, 8);
});
