import test from "node:test";
import assert from "node:assert/strict";
import {
  SUPPORTED_API,
  normalizeManifests,
  supportedApp,
  normalizeView,
  aggregateBadge,
} from "./appPrimitives.js";

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
    id: "demo", name: "Demo", icon: "flask", apiVersion: 1, badge: { count: 3, dot: false },
  });
  assert.equal(out[1].apiVersion, 9);
  assert.equal(out[1].badge.dot, true);
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
