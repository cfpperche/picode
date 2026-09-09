import assert from "node:assert/strict";
import { test } from "node:test";
import {
  RECONNECT_BURST,
  RECONNECT_MAX_MS,
  connectTermSocket,
  dropTermSocket,
  kickTermSocket,
  suspendTermSocket,
  isTermSocketSuspended,
} from "./termSocket.js";

function makeDeps() {
  const deps = {
    WebSocketImpl: null,
    setTimer: (fn, ms) => {
      const id = deps.timers.length + 1;
      deps.timers.push({ id, fn, ms });
      return id;
    },
    clearTimer: (id) => {
      const i = deps.timers.findIndex((t) => t.id === id);
      if (i >= 0) deps.timers.splice(i, 1);
    },
    doc: { visibilityState: "visible", listeners: {}, addEventListener(type, fn) { this.listeners[type] = fn; }, removeEventListener(type) { delete this.listeners[type]; } },
    win: { listeners: {}, addEventListener(type, fn) { this.listeners[type] = fn; }, removeEventListener(type) { delete this.listeners[type]; } },
    timers: [],
    created: [],
  };
  deps.flush = () => {
    const batch = deps.timers.splice(0, deps.timers.length);
    for (const t of batch) t.fn();
  };
  deps.lastDelay = () => (deps.timers.length ? deps.timers[deps.timers.length - 1].ms : null);
  deps.visible = () => deps.doc.listeners.visibilitychange && deps.doc.listeners.visibilitychange();
  deps.online = () => deps.win.listeners.online && deps.win.listeners.online();
  return deps;
}

class FakeWS {
  constructor(url, deps) {
    this.url = url;
    this.readyState = 0; // CONNECTING
    deps.created.push(this);
  }
  open() {
    this.readyState = 1;
    if (this.onopen) this.onopen();
  }
  message(data) {
    if (this.onmessage) this.onmessage({ data });
  }
  close() {
    if (this.readyState === 3) return;
    this.readyState = 3;
    if (this.onclose) this.onclose();
  }
  fail() {
    // server refused the upgrade: never opens
    this.readyState = 3;
    if (this.onerror) this.onerror(new Error("upgrade failed"));
    if (this.onclose) this.onclose();
  }
}

function setup(deps) {
  // A plain function: `new deps.WebSocketImpl(url)` must construct.
  deps.WebSocketImpl = function (url) {
    return new FakeWS(url, deps);
  };
  const entry = { sock: null, closedByUser: false };
  const calls = { open: 0, state: [], messages: [], giveUp: 0 };
  const ctl = connectTermSocket(entry, "ws://x/ws/term", {
    onOpen: () => {
      calls.open += 1;
    },
    onMessage: (ev) => {
      calls.messages.push(ev.data);
    },
    onState: (c) => {
      calls.state.push(c);
    },
    onGiveUp: () => {
      calls.giveUp += 1;
    },
  }, deps);
  return { entry, ctl, calls };
}

test("connects, forwards messages, and reattaches after a drop keeping the same xterm", () => {
  const deps = makeDeps();
  const { entry, calls } = setup(deps);
  assert.equal(deps.created.length, 1);
  deps.created[0].open();
  assert.equal(calls.open, 1);
  entry.sock.message("hello");
  assert.deepEqual(calls.messages, ["hello"]);

  // Phone locked: the socket dies. One state(false), then a scheduled retry.
  entry.sock.close();
  assert.deepEqual(calls.state, [false]);
  assert.equal(deps.lastDelay(), 1000);
  deps.flush();
  assert.equal(deps.created.length, 2, "reattached");
  assert.equal(entry.sock, deps.created[1], "entry.sock points at the new socket");
  deps.created[1].open();
  assert.equal(calls.open, 2);
  assert.deepEqual(calls.state, [false], "no extra state write for a clean reattach");
});

test("backoff grows to the cap and resets after a successful reattach", () => {
  const deps = makeDeps();
  const { entry, calls } = setup(deps);
  deps.created[0].open();
  const seen = [];
  for (let i = 0; i < RECONNECT_BURST; i++) {
    entry.sock.close();
    seen.push(deps.lastDelay());
    deps.flush();
  }
  assert.deepEqual(seen, [1000, 2000, 4000, 8000, 10000, 10000, 10000, 10000, 10000, 10000, 10000, 10000]);
  // One more failure with no live connection exhausts the burst.
  entry.sock.close();
  assert.equal(deps.lastDelay(), null, "give-up: no retry scheduled");
  assert.equal(calls.giveUp, 1);
  // A kick revives the entry, and a connection that opens resets the burst.
  kickTermSocket(entry);
  deps.created[deps.created.length - 1].open();
  entry.sock.close();
  assert.equal(deps.lastDelay(), 1000, "burst reset after a live connection");
});

test("gives up after the burst when the session is gone, kick restarts it", () => {
  const deps = makeDeps();
  const { entry, calls } = setup(deps);
  for (let i = 0; i < RECONNECT_BURST; i++) {
    assert.equal(deps.created.length, i + 1);
    deps.created[i].fail(); // 404 / refused upgrade: never opens
    deps.flush();
  }
  assert.equal(calls.giveUp, 0, "burst not exhausted yet");
  deps.created[RECONNECT_BURST].fail();
  assert.equal(calls.giveUp, 1, "gives up after RECONNECT_BURST failures");
  assert.equal(deps.timers.length, 0, "no retries scheduled after give-up");

  // Failed attempts after give-up neither retry nor write more markers.
  deps.created[deps.created.length - 1].fail();
  assert.equal(calls.giveUp, 1);
  assert.equal(deps.created.length, RECONNECT_BURST + 1);

  // A kick (visibility regained, network back, remount) restarts the burst.
  kickTermSocket(entry);
  assert.equal(deps.created.length, RECONNECT_BURST + 2);
  deps.created[deps.created.length - 1].open();
  assert.equal(calls.open, 1, "revived");
});

test("visibility and online events kick an immediate reconnect", () => {
  const deps = makeDeps();
  const { entry } = setup(deps);
  deps.created[0].open();
  entry.sock.close();
  assert.equal(deps.created.length, 1);
  deps.visible(); // page visible again while only a timer is pending
  assert.equal(deps.created.length, 2, "immediate attempt, no timer wait");
  assert.equal(deps.timers.length, 0, "scheduled retry cancelled");
  deps.created[1].open();
  assert.equal(entry.sock.readyState, 1);
  const count = deps.created.length;
  deps.online(); // healthy socket: kick is a no-op
  assert.equal(deps.created.length, count);
});

test("kick clears give-up so a recreated session heals on the next wake", () => {
  const deps = makeDeps();
  const { entry, calls } = setup(deps);
  for (let i = 0; i <= RECONNECT_BURST; i++) {
    deps.created[i].fail();
    deps.flush();
  }
  assert.equal(calls.giveUp, 1);
  deps.visible();
  assert.ok(deps.created.length > RECONNECT_BURST + 1, "fresh burst on visibility");
});

test("dropTermSocket stops reattach and unwires the window listeners", () => {
  const deps = makeDeps();
  const { entry, calls } = setup(deps);
  deps.created[0].open();
  dropTermSocket(entry);
  assert.equal(deps.timers.length, 0);
  assert.ok(!entry.__sockCtl, "control state removed");
  assert.ok(!deps.doc.listeners.visibilitychange && !deps.win.listeners.online, "window listeners unwired");
  entry.sock.close();
  assert.deepEqual(calls.state, [], "no state write after a user close");
  deps.flush();
  deps.visible();
  deps.online();
  assert.equal(deps.created.length, 1, "no reattach after a user close");
});

test("entries that never connected still close cleanly (no __sockCtl)", () => {
  const deps = makeDeps();
  const entry = { sock: null, closedByUser: false };
  dropTermSocket(entry); // must not throw
  kickTermSocket(entry);
  assert.equal(entry.closedByUser, false, "kick on a bare entry is a no-op");
});

// The reversible stop (ADR-0109, matrix plan §4.5): a panel scrolled out
// of view closes its attach without losing the xterm.
test("suspend closes the socket and never reconnects: no timer, no state write", () => {
  const deps = makeDeps();
  const { entry, calls } = setup(deps);
  deps.created[0].open();
  suspendTermSocket(entry);
  assert.equal(isTermSocketSuspended(entry), true);
  assert.equal(entry.sock.readyState, 3, "socket closed");
  assert.deepEqual(calls.state, [], "no '— detached —' line while suspended");
  assert.equal(deps.timers.length, 0, "no retry scheduled");
  deps.flush();
  assert.equal(deps.created.length, 1, "nothing reconnected");
  assert.ok(entry.__sockCtl, "control block kept for the resume");
  // A suspension taken while a retry was pending cancels that retry too.
  const again = makeDeps();
  const second = setup(again);
  again.created[0].open();
  second.entry.sock.close(); // dropped: a retry is now pending
  assert.equal(again.timers.length, 1);
  suspendTermSocket(second.entry);
  assert.equal(again.timers.length, 0, "pending retry cancelled");
  again.flush();
  assert.equal(again.created.length, 1);
});

test("kick after suspend reattaches the same entry and resumes normal retries", () => {
  const deps = makeDeps();
  const { entry, calls } = setup(deps);
  deps.created[0].open();
  suspendTermSocket(entry);
  kickTermSocket(entry);
  assert.equal(isTermSocketSuspended(entry), false);
  assert.equal(deps.created.length, 2, "a new socket for the same entry");
  assert.equal(entry.sock, deps.created[1]);
  deps.created[1].open();
  assert.equal(calls.open, 2, "onOpen fires for the resumed attach");
  assert.deepEqual(calls.state, [], "still no detached line: the suspend was silent");
  // Back to normal: a real drop after the resume writes state and retries.
  entry.sock.close();
  assert.deepEqual(calls.state, [false]);
  assert.equal(deps.lastDelay(), 1000);
});

test("dropTermSocket after suspend wins: closedByUser, no reconnect ever", () => {
  const deps = makeDeps();
  const { entry, calls } = setup(deps);
  deps.created[0].open();
  suspendTermSocket(entry);
  dropTermSocket(entry);
  assert.equal(entry.closedByUser, true);
  assert.ok(!entry.__sockCtl, "control state removed");
  assert.equal(isTermSocketSuspended(entry), false, "no control block, no suspension to report");
  kickTermSocket(entry);
  deps.visible();
  deps.online();
  deps.flush();
  assert.equal(deps.created.length, 1, "no reattach after a user close");
  assert.deepEqual(calls.state, []);
});
