import test from "node:test";
import assert from "node:assert/strict";
import { startFeed, stopFeed, subscribeFeed, feedConnected, feedCursor, parseChange, _resetFeedForTests } from "./feed.js";

// A fake EventSource: the test drives open / events / errors by hand.
class FakeES {
  constructor(url) { this.url = url; this.listeners = {}; FakeES.last = this; }
  addEventListener(name, fn) { this.listeners[name] = fn; }
  fire(name, data, lastEventId) { this.listeners[name]({ data, lastEventId }); }
  close() { this.closed = true; }
}

const store = new Map();
globalThis.sessionStorage = {
  getItem: (k) => (store.has(k) ? store.get(k) : null),
  setItem: (k, v) => store.set(k, String(v)),
  removeItem: (k) => store.delete(k),
};
globalThis.window = globalThis.window || {};

test("parseChange takes the id from Last-Event-ID and rejects junk", () => {
  assert.equal(parseChange("nope", "3"), null);
  assert.equal(parseChange('{"nope":1}', "3"), null);
  assert.deepEqual(parseChange('{"type":"inbox.created","data":{"id":"a"}}', "7"), { type: "inbox.created", data: { id: "a" }, id: 7 });
  assert.equal(parseChange('{"type":"device.online","data":{}}', "").id, 0);
});

test("feed: open, change with cursor, down, reset, and bootId change", () => {
  _resetFeedForTests();
  store.clear();
  const seen = [];
  const states = [];
  let kicked = 0;
  window.__picodeKickHealth = () => { kicked++; };
  subscribeFeed((ev) => seen.push(ev.type + (ev.id ? "#" + ev.id : "")));
  startFeed({ EventSourceImpl: FakeES, onState: (s) => states.push(s) });
  const es = FakeES.last;
  assert.equal(es.url, "/api/events");
  assert.equal(feedConnected(), false);
  es.onopen();
  assert.equal(feedConnected(), true);
  assert.deepEqual(seen, ["feed.open"]);
  es.fire("hello", JSON.stringify({ bootId: "b1", latest: 4 }));
  es.fire("change", JSON.stringify({ type: "inbox.created", data: { id: "a" } }), "5");
  assert.equal(feedCursor(), 5);
  assert.equal(store.get("picode-feed-seq"), "5");
  es.fire("change", JSON.stringify({ type: "device.online", data: {} }), "");
  assert.equal(feedCursor(), 5, "ephemeral events do not move the cursor");
  es.onerror();
  assert.equal(feedConnected(), false);
  assert.equal(kicked, 1);
  es.onopen();
  es.fire("reset", "{}");
  assert.equal(feedCursor(), 0);
  es.fire("hello", JSON.stringify({ bootId: "b2", latest: 9 }));
  assert.equal(kicked, 2, "a new bootId kicks the health watch");
  assert.deepEqual(seen, ["feed.open", "inbox.created#5", "device.online", "feed.down", "feed.open", "feed.reset"]);
  assert.deepEqual(states, ["open", "down", "open"]);
  stopFeed();
  assert.equal(es.closed, true);

  // A reload resumes from the stored cursor.
  store.set("picode-feed-seq", "42");
  startFeed({ EventSourceImpl: FakeES });
  assert.equal(FakeES.last.url, "/api/events?after=42");
  _resetFeedForTests();
});
