import assert from "node:assert/strict";
import { test } from "node:test";
import { readDraft, writeDraft, clearDraft } from "./draft.js";

function memStore() {
  const store = {};
  globalThis.localStorage = {
    getItem: (k) => (k in store ? store[k] : null),
    setItem: (k, v) => { store[k] = String(v); },
    removeItem: (k) => { delete store[k]; },
  };
  return store;
}

test("write then read per agent", () => {
  memStore();
  writeDraft("a", "hello", "steer");
  writeDraft("b", "other", "follow_up");
  assert.deepEqual(readDraft("a"), { text: "hello", kind: "steer" });
  assert.deepEqual(readDraft("b"), { text: "other", kind: "follow_up" });
});

test("empty text deletes the slot", () => {
  memStore();
  writeDraft("a", "hello", "prompt");
  writeDraft("a", "  ", "steer");
  assert.deepEqual(readDraft("a"), { text: "", kind: "prompt" });
});

test("clearDraft deletes", () => {
  memStore();
  writeDraft("a", "x", "prompt");
  clearDraft("a");
  assert.equal(readDraft("a").text, "");
});

test("unknown kind becomes prompt", () => {
  memStore();
  writeDraft("a", "x", "nope");
  assert.equal(readDraft("a").kind, "prompt");
});

test("missing agent is empty", () => {
  memStore();
  assert.deepEqual(readDraft("gone"), { text: "", kind: "prompt" });
  assert.deepEqual(readDraft(""), { text: "", kind: "prompt" });
});
