import assert from "node:assert/strict";
import { test, beforeEach } from "node:test";

const store = new Map();
globalThis.localStorage = {
  getItem: (k) => (store.has(k) ? store.get(k) : null),
  setItem: (k, v) => store.set(k, String(v)),
  removeItem: (k) => store.delete(k),
};

const { writeAskMemory, mergeAskMemory } = await import("./askMemory.js");

const slash = (text, ts) => ({ kind: "block", cls: "user", text, ts });
const ask = (id, status, ts) => ({
  kind: "ask", id, status, ts,
  steps: [{ id, status, answer: status === "answered" ? "default" : "" }],
});

beforeEach(() => store.clear());

test("row 8: an answered card and its slash bubble survive a reload merge", () => {
  const items = [slash("/roles", 1), ask("a", "answered", 2)];
  writeAskMemory("ag1", "s.jsonl", items);
  const merged = mergeAskMemory("ag1", "s.jsonl", []);
  assert.deepEqual(merged.map((it) => it.kind), ["block", "ask"]);
  assert.equal(merged[1].status, "answered");
});

test("cancelled and timed-out cards are not kept", () => {
  writeAskMemory("ag1", "s.jsonl", [slash("/roles", 1), ask("a", "cancelled", 2)]);
  const merged = mergeAskMemory("ag1", "s.jsonl", []);
  assert.deepEqual(merged.map((it) => it.kind), ["block"]);
});

test("row 9: the same command typed twice stays two bubbles", () => {
  const items = [slash("/roles", 1), ask("a", "answered", 2), slash("/roles", 3), ask("b", "answered", 4)];
  writeAskMemory("ag1", "s.jsonl", items);
  const merged = mergeAskMemory("ag1", "s.jsonl", []);
  assert.equal(merged.filter((it) => it.kind === "block").length, 2);
  // merging over already-merged items adds nothing
  const again = mergeAskMemory("ag1", "s.jsonl", merged);
  assert.equal(again.length, merged.length);
});

test("no session file yet: the live slot keeps the thread, then migrates", () => {
  const items = [slash("/roles", 1), ask("a", "answered", 2)];
  writeAskMemory("ag1", "", items);
  const merged = mergeAskMemory("ag1", "", []);
  assert.equal(merged.length, 2);
  // first real session appears: same items now write under its path
  writeAskMemory("ag1", "s.jsonl", items);
  assert.equal(mergeAskMemory("ag1", "s.jsonl", []).length, 2);
  assert.deepEqual(mergeAskMemory("ag1", "", []), []);
});

test("slots are per agent and session", () => {
  writeAskMemory("ag1", "s.jsonl", [slash("/roles", 1), ask("a", "answered", 2)]);
  assert.deepEqual(mergeAskMemory("ag2", "s.jsonl", []), []);
  assert.deepEqual(mergeAskMemory("ag1", "other.jsonl", []), []);
});
