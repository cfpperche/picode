// openUrlFeed.test.js — the hand-off decision table (ADR-0180):
// visible + http(s) + first sighting → open; hidden, empty, duplicate or
// foreign → ignore; after the dedupe window the same URL opens again.
import test from "node:test";
import assert from "node:assert/strict";
import { installOpenUrlFeed, openUrlAllowed, OPEN_URL_EVENT } from "./openUrlFeed.js";

function stub() {
  const calls = [];
  let listener = null;
  const window = { visibility: "visible", now: 1000 };
  const sub = {
    subscribe: (fn) => { listener = fn; return () => { listener = null; }; },
    open: (url) => calls.push(url),
    hidden: () => window.visibility !== "visible",
    now: () => window.now,
  };
  return { calls, window, sub, fire: (ev) => listener && listener(ev) };
}

test("a visible client opens a CLI's URL", () => {
  const s = stub();
  installOpenUrlFeed(s.sub);
  s.fire({ type: OPEN_URL_EVENT, data: { termId: "t1", url: "https://auth.example/login" } });
  assert.deepEqual(s.calls, ["https://auth.example/login"]);
});

test("a hidden client stays out — the server keeps the fallback", () => {
  const s = stub();
  s.window.visibility = "hidden";
  installOpenUrlFeed(s.sub);
  s.fire({ type: OPEN_URL_EVENT, data: { termId: "t1", url: "https://a.example/" } });
  assert.deepEqual(s.calls, []);
});

test("unrelated and malformed events are ignored", () => {
  const s = stub();
  installOpenUrlFeed(s.sub);
  s.fire({ type: "agent.state", data: {} });
  s.fire({ type: OPEN_URL_EVENT, data: {} });
  s.fire({ type: OPEN_URL_EVENT, data: { url: "   " } });
  s.fire(null);
  assert.deepEqual(s.calls, []);
});

test("the same URL inside the window opens once, after it again", () => {
  const s = stub();
  installOpenUrlFeed(s.sub);
  s.fire({ type: OPEN_URL_EVENT, data: { termId: "t1", url: "https://x.example/" } });
  s.window.now += 400;
  s.fire({ type: OPEN_URL_EVENT, data: { termId: "t1", url: "https://x.example/" } });
  s.window.now += 2000;
  s.fire({ type: OPEN_URL_EVENT, data: { termId: "t1", url: "https://x.example/" } });
  assert.deepEqual(s.calls, ["https://x.example/", "https://x.example/"]);
});

test("different terminals and different URLs are not fused", () => {
  const s = stub();
  installOpenUrlFeed(s.sub);
  s.fire({ type: OPEN_URL_EVENT, data: { termId: "t1", url: "https://x.example/" } });
  s.fire({ type: OPEN_URL_EVENT, data: { termId: "t2", url: "https://x.example/" } });
  s.fire({ type: OPEN_URL_EVENT, data: { termId: "t1", url: "https://y.example/" } });
  assert.deepEqual(s.calls, ["https://x.example/", "https://x.example/", "https://y.example/"]);
});

test("uninstall detaches the listener", () => {
  const s = stub();
  const off = installOpenUrlFeed(s.sub);
  off();
  s.fire({ type: OPEN_URL_EVENT, data: { termId: "t1", url: "https://x.example/" } });
  assert.deepEqual(s.calls, []);
});

test("openUrlAllowed bounds the seen map", () => {
  const seen = new Map();
  for (let i = 0; i < 200; i++) openUrlAllowed(seen, "k" + i, i * 100);
  assert.ok(seen.size < 100);
});
