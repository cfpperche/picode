import assert from "node:assert/strict";
import { test } from "node:test";
import { readOpenTabs, writeOpenTabs, filterOpenTabs, moveTab, readTermWanted, writeTermWanted, readGitOwners, writeGitOwners } from "./openTabs.js";

test("filterOpenTabs drops missing agents", () => {
  const got = filterOpenTabs(
    { ids: ["a", "gone", "b"], selected: "gone" },
    (id) => id === "a" || id === "b",
  );
  assert.deepEqual(got.ids, ["a", "b"]);
  assert.equal(got.selected, "a");
});

test("moveTab reorders and no-ops on bad ids", () => {
  assert.deepEqual(moveTab(["a", "b", "c"], "a", "c"), ["b", "c", "a"]);
  assert.deepEqual(moveTab(["a", "b", "c"], "c", "a"), ["c", "a", "b"]);
  assert.deepEqual(moveTab(["a", "b"], "a", "a"), ["a", "b"]);
  assert.deepEqual(moveTab(["a", "b"], "z", "a"), ["a", "b"]);
});

test("roundtrip", () => {
  const store = {};
  globalThis.localStorage = {
    getItem: (k) => (k in store ? store[k] : null),
    setItem: (k, v) => { store[k] = String(v); },
  };
  writeOpenTabs(["x", "y"], "y");
  const got = readOpenTabs();
  assert.deepEqual(got.ids, ["x", "y"]);
  assert.equal(got.selected, "y");
});

test("a renamed app's tab id is rewritten on read (ADR-0118)", () => {
  const store = { "picode-tabs": JSON.stringify({ ids: ["a", "x:matrix", "b"], selected: "x:matrix" }) };
  globalThis.localStorage = {
    getItem: (k) => (k in store ? store[k] : null),
    setItem: (k, v) => { store[k] = String(v); },
  };
  const got = readOpenTabs();
  assert.deepEqual(got.ids, ["a", "x:canvas", "b"], "x:matrix opens the canvas app, not a dead tab");
  assert.equal(got.selected, "x:canvas", "the selection follows its tab");
  store["picode-tabs"] = JSON.stringify({ ids: ["x:canvas", "x:matrix"], selected: "x:canvas" });
  assert.deepEqual(readOpenTabs().ids, ["x:canvas"], "both ids are one tab, not two");
});

test("term view roundtrip and dedupe", () => {
  const store = {};
  globalThis.localStorage = {
    getItem: (k) => (k in store ? store[k] : null),
    setItem: (k, v) => { store[k] = String(v); },
  };
  writeTermWanted(["a", "b", "a", ""]);
  assert.deepEqual(readTermWanted(), ["a", "b"]);
  writeTermWanted([]);
  assert.deepEqual(readTermWanted(), []);
});

test("git owners survive a round trip and reject junk", () => {
  const store = {};
  globalThis.localStorage = {
    getItem: (k) => (k in store ? store[k] : null),
    setItem: (k, v) => { store[k] = String(v); },
  };

  writeGitOwners({ "g:/r/.git": { kind: "agent", id: "opus", name: "Opus" } });
  assert.deepEqual(readGitOwners(), { "g:/r/.git": { kind: "agent", id: "opus", name: "Opus" } });

  // A workspace owner survives the reload: downgrading it to an agent sent
  // the restored tab to /api/agents/<workspace id> and 404ed (ADR-0030).
  store["picode-git-owners"] = JSON.stringify({ "g:/w/.git": { kind: "workspace", id: "ws1", name: "picode" } });
  assert.deepEqual(readGitOwners(), { "g:/w/.git": { kind: "workspace", id: "ws1", name: "picode" } });

  // An unknown kind falls back to agent; an entry with no id is dropped.
  store["picode-git-owners"] = JSON.stringify({
    "g:/a": { kind: "wat", id: "x" },
    "g:/b": { kind: "term", id: "" },
    "g:/c": "not-an-object",
  });
  assert.deepEqual(readGitOwners(), { "g:/a": { kind: "agent", id: "x", name: "" } });

  store["picode-git-owners"] = "{ broken";
  assert.deepEqual(readGitOwners(), {});
  store["picode-git-owners"] = JSON.stringify([1, 2]);
  assert.deepEqual(readGitOwners(), {});
});
