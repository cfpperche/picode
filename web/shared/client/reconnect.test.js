import { test } from "node:test";
import assert from "node:assert/strict";
import { startReconnectWatch, pageServing } from "./reconnect.js";

test("pageServing: true only for an ok answer", async () => {
  assert.equal(await pageServing(async () => ({ ok: true }), "/x/"), true);
  assert.equal(await pageServing(async () => ({ ok: false }), "/x/"), false);
  assert.equal(await pageServing(async () => { throw new Error("down"); }, "/x/"), false);
});

test("bootId change defers the reload until the page serves again", async () => {
  let pings = 0;
  let probes = 0;
  let reloaded = 0;
  const stop = startReconnectWatch({
    ping: async () => (pings += 1) === 1 ? "boot-1" : "boot-2",
    reload: () => { reloaded += 1; },
    probe: async () => (probes += 1) >= 3,
    waitMs: 1,
    okMs: 2,
    downMs: 2,
  });
  for (let i = 0; i < 200 && reloaded === 0; i++) {
    await new Promise((r) => setTimeout(r, 5));
  }
  stop();
  assert.ok(reloaded >= 1, "reload must eventually fire");
  assert.ok(probes >= 2, `the reload must wait for the page (probes=${probes})`);
});

test("a probe that never succeeds still reloads (bounded, fail-open)", async () => {
  let pings = 0;
  let reloaded = 0;
  const stop = startReconnectWatch({
    ping: async () => (pings += 1) === 1 ? "boot-1" : "boot-2",
    reload: () => { reloaded += 1; },
    probe: async () => false,
    waitMs: 1,
    okMs: 2,
    downMs: 2,
  });
  for (let i = 0; i < 400 && reloaded === 0; i++) {
    await new Promise((r) => setTimeout(r, 5));
  }
  stop();
  assert.equal(reloaded, 1, "fail-open reload fires after the bounded wait");
});

// The production callers (browser and mobile App.jsx) pass only onState: the
// probe is a default, not a duty of the call site. Without it the reload path
// threw, the tick loop died, and the Reconnecting overlay stayed up until a
// manual reload — the deploy screen that never left (2026-09-21).
test("a watch started without a probe reloads after a deploy", async () => {
  let pings = 0;
  let reloaded = 0;
  const states = [];
  const stop = startReconnectWatch({
    ping: async () => (pings += 1) === 1 ? "boot-1" : pings <= 3 ? null : "boot-2",
    reload: () => { reloaded += 1; },
    waitMs: 1,
    okMs: 2,
    downMs: 2,
    onState: (s) => states.push(s),
  });
  for (let i = 0; i < 400 && reloaded === 0; i++) {
    await new Promise((r) => setTimeout(r, 5));
  }
  stop();
  assert.deepEqual(states, ["ok", "down", "up"]);
  assert.equal(reloaded, 1, "the reload must fire with no injected probe");
});

test("a probe that throws is not a reason to strand the page", async () => {
  let pings = 0;
  let reloaded = 0;
  const stop = startReconnectWatch({
    ping: async () => (pings += 1) === 1 ? "boot-1" : "boot-2",
    reload: () => { reloaded += 1; },
    probe: async () => { throw new TypeError("probe is not a function"); },
    waitMs: 1,
    okMs: 2,
    downMs: 2,
  });
  for (let i = 0; i < 400 && reloaded === 0; i++) {
    await new Promise((r) => setTimeout(r, 5));
  }
  stop();
  assert.equal(reloaded, 1, "a broken probe still fails open into the reload");
});
