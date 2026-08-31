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
