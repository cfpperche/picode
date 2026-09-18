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
