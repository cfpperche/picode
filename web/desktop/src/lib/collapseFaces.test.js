import assert from "node:assert/strict";
import { test } from "node:test";
import { collapseFaceItems } from "./collapseFaces.js";

test("agents come first, then terminals, with stable namespaced ids", () => {
  const items = collapseFaceItems(
    [{ id: "a1" }, { id: "a2" }],
    [{ id: "t1" }, { id: "t2" }],
  );
  assert.deepEqual(items.map((i) => i.id), ["a:a1", "a:a2", "t:t1", "t:t2"]);
  assert.equal(items[0].kind, "agent");
  assert.equal(items[2].kind, "term");
});

test("a workspace with only terminals is not empty", () => {
  const items = collapseFaceItems([], [{ id: "t1" }, { id: "t2" }]);
  assert.equal(items.length, 2);
  assert.ok(items.every((i) => i.kind === "term"));
});

test("blank entries are dropped; null lists behave like empty", () => {
  assert.deepEqual(collapseFaceItems(null, null), []);
  assert.deepEqual(
    collapseFaceItems([{ id: "a1" }, null, {}], [null, { id: "t1" }, {}]),
    [{ kind: "agent", id: "a:a1", ag: { id: "a1" } }, { kind: "term", id: "t:t1", term: { id: "t1" } }],
  );
});
