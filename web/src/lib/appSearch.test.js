import assert from "node:assert/strict";
import { test } from "node:test";
import { matchesItem, filterListBlocks, countListItems } from "./appSearch.js";

const item = { id: "1", title: "Should inbox rows support j/k", subtitle: "g.", meta: ["agent needs you"], badge: "QUESTION" };

test("empty or blank query always matches", () => {
  assert.equal(matchesItem(item, ""), true);
  assert.equal(matchesItem(item, "   "), true);
  assert.equal(matchesItem({}, ""), true);
});

test("matches by title, subtitle, badge and meta, case-insensitively", () => {
  assert.equal(matchesItem(item, "ROWS"), true);
  assert.equal(matchesItem(item, "g."), true);
  assert.equal(matchesItem(item, "question"), true);
  assert.equal(matchesItem(item, "needs you"), true);
});

test("no match when the query is nowhere in the item", () => {
  assert.equal(matchesItem(item, "snooze"), false);
});

const blocks = [
  { type: "list", title: "Needs you", items: [item, { id: "2", title: "Aprovar janela de manutenção", meta: [] }] },
  { type: "actions", actions: [{ id: "clear", label: "Clear all done" }] },
];

test("filterListBlocks keeps only matching items in list blocks", () => {
  const out = filterListBlocks(blocks, "manutenção");
  assert.equal(out.length, 2);
  assert.equal(out[0].items.length, 1);
  assert.equal(out[0].items[0].id, "2");
});

test("filterListBlocks drops a list block left with zero items", () => {
  const out = filterListBlocks(blocks, "nothing matches this");
  assert.equal(out.some((b) => b.type === "list"), false);
});

test("filterListBlocks never touches non-list blocks", () => {
  const out = filterListBlocks(blocks, "nothing matches this");
  const actions = out.find((b) => b.type === "actions");
  assert.deepEqual(actions, blocks[1]);
});

test("filterListBlocks is a no-op for a blank query", () => {
  assert.deepEqual(filterListBlocks(blocks, ""), blocks);
  assert.deepEqual(filterListBlocks(blocks, "   "), blocks);
});

test("countListItems sums only list blocks, and is safe on empty input", () => {
  assert.equal(countListItems(blocks), 2);
  assert.equal(countListItems([]), 0);
  assert.equal(countListItems(null), 0);
  assert.equal(countListItems(undefined), 0);
});
