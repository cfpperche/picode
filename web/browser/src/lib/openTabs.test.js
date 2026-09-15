import assert from "node:assert/strict";
import { test } from "node:test";
import { filterAgentSplits, filterOpenTabs, moveTab, readAgentSplits, readGitOwners, readOpenTabs, readTermWanted, readWebTabUrls, writeAgentSplitUrls, writeAgentSplits, writeGitOwners, writeOpenTabs, writeTermWanted, writeWebTabUrls } from "./openTabs.js";

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

test("the agent split survives a relaunch: layout and urls round-trip", () => {
  const store = {};
  globalThis.localStorage = {
    getItem: (k) => (k in store ? store[k] : null),
    setItem: (k, v) => { store[k] = String(v); },
  };
  writeAgentSplits({ panes: { opus: "1", "term:t7": "2" }, ratios: { opus: 33 }, max: { "term:t7": true } });
  writeAgentSplitUrls({ 1: "https://globo.com/", 2: "" });
  const got = readAgentSplits();
  assert.deepEqual(got.panes, { opus: "1", "term:t7": "2" });
  assert.deepEqual(got.ratios, { opus: 33 });
  assert.deepEqual(got.max, { "term:t7": true });
  // An empty url is not a page to come back to.
  assert.deepEqual(got.urls, { 1: "https://globo.com/" });

  // Junk is dropped field by field: a pane whose id is not a webview id, a
  // ratio out of the drag range, a max flag on a pane that is gone.
  store["picode-agent-splits"] = JSON.stringify({
    panes: { opus: "1", gone: "x", "": "2" },
    ratios: { opus: 99, gone: 40 },
    max: { opus: "yes", gone: true },
  });
  const clean = readAgentSplits();
  assert.deepEqual(clean.panes, { opus: "1" });
  assert.deepEqual(clean.ratios, { opus: 80 });
  assert.deepEqual(clean.max, {});

  store["picode-agent-splits"] = "{ broken";
  store["picode-agent-split-urls"] = "{ broken";
  assert.deepEqual(readAgentSplits(), { panes: {}, ratios: {}, max: {}, urls: {} });
});

test("filterAgentSplits drops the panes whose host tab is gone", () => {
  const splits = {
    panes: { opus: "1", "term:t7": "2", dead: "3" },
    ratios: { opus: 25, dead: 60 },
    max: { opus: false, "term:t7": true },
    urls: { 1: "https://a/", 2: "https://b/", 3: "https://c/" },
  };
  const got = filterAgentSplits(splits, (key) => key !== "dead");
  assert.deepEqual(got.panes, { opus: "1", "term:t7": "2" });
  assert.deepEqual(got.ratios, { opus: 25 });
  assert.deepEqual(got.max, { "term:t7": true });
  // The dead pane's url goes with it; the live panes keep theirs.
  assert.deepEqual(got.urls, { 1: "https://a/", 2: "https://b/" });
  // A missing map is an empty split, not a crash.
  assert.deepEqual(filterAgentSplits(undefined, () => true), { panes: {}, ratios: {}, max: {}, urls: {} });
});

test("web tab addresses round-trip for shells without a webview", () => {
  const store = new Map();
  const original = globalThis.localStorage;
  globalThis.localStorage = {
    getItem: (k) => (store.has(k) ? store.get(k) : null),
    setItem: (k, v) => store.set(k, String(v)),
  };
  try {
    writeWebTabUrls({ 1: { url: "http://localhost:5173/", title: "Acme" }, 2: { url: "", title: "x" }, 3: "junk" });
    assert.deepEqual(readWebTabUrls(), { 1: { url: "http://localhost:5173/", title: "Acme" } });
    // Corrupt storage is not a crash: the strip opens empty tabs instead.
    localStorage.setItem("picode-webtab-urls", "{not json");
    assert.deepEqual(readWebTabUrls(), {});
  } finally {
    globalThis.localStorage = original;
  }
});
