import assert from "node:assert/strict";
import { test } from "node:test";
import { pingHealth, startReconnectWatch } from "./reconnect.js";

function wait(ms) {
  return new Promise((r) => setTimeout(r, ms));
}

test("pingHealth is true only on ok", async () => {
  assert.equal(await pingHealth(async () => ({ ok: true })), true);
  assert.equal(await pingHealth(async () => ({ ok: false })), false);
  assert.equal(await pingHealth(async () => { throw new Error("offline"); }), false);
});

test("watch goes down after misses and reloads on recovery", async () => {
  const seq = [false, false, true];
  let i = 0;
  const states = [];
  let reloads = 0;
  const stop = startReconnectWatch({
    ping: async () => seq[Math.min(i++, seq.length - 1)],
    reload: () => { reloads += 1; },
    onState: (s) => states.push(s),
    downAfter: 2,
    okMs: 8,
    downMs: 8,
  });
  await wait(50);
  stop();
  assert.ok(states.includes("down"), "states " + states.join(","));
  assert.ok(states.includes("up"), "states " + states.join(","));
  assert.equal(reloads, 1);
});

test("watch stays up when health is ok", async () => {
  const states = [];
  let reloads = 0;
  const stop = startReconnectWatch({
    ping: async () => true,
    reload: () => { reloads += 1; },
    onState: (s) => states.push(s),
    downAfter: 2,
    okMs: 8,
    downMs: 8,
  });
  await wait(30);
  stop();
  assert.equal(reloads, 0);
  assert.ok(!states.includes("down"));
});
