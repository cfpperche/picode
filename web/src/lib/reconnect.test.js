import assert from "node:assert/strict";
import { test } from "node:test";
import { pingHealth, startReconnectWatch } from "./reconnect.js";

globalThis.window = {
  __picodeKickHealth: undefined,
  addEventListener() {},
  removeEventListener() {},
};
globalThis.document = {
  addEventListener() {},
  removeEventListener() {},
};

function wait(ms) {
  return new Promise((r) => setTimeout(r, ms));
}

test("pingHealth returns bootId string or null", async () => {
  assert.equal(await pingHealth(async () => ({ ok: true, json: async () => ({ bootId: "abc" }) })), "abc");
  assert.equal(await pingHealth(async () => ({ ok: false, json: async () => ({}) })), null);
  assert.equal(await pingHealth(async () => { throw new Error("offline"); }), null);
});

test("watch goes down after misses and reloads on recovery", async () => {
  const seq = [null, null, "b1"];
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
  await wait(60);
  stop();
  assert.ok(states.includes("down"), "states " + states.join(","));
  assert.ok(states.includes("up"), "states " + states.join(","));
  assert.equal(reloads, 1);
});

test("watch stays up when health is ok", async () => {
  const states = [];
  let reloads = 0;
  const stop = startReconnectWatch({
    ping: async () => "b1",
    reload: () => { reloads += 1; },
    onState: (s) => states.push(s),
    okMs: 8,
    downMs: 8,
  });
  await wait(30);
  stop();
  assert.equal(reloads, 0);
  assert.ok(!states.includes("down"));
});

test("fast restart: bootId change reloads without downtime", async () => {
  let boot = "b1";
  const states = [];
  let reloads = 0;
  const stop = startReconnectWatch({
    ping: async () => boot,
    reload: () => { reloads += 1; },
    onState: (s) => states.push(s),
    okMs: 10,
    downMs: 10,
  });
  await wait(25);
  boot = "b2"; // server restarted between polls
  await wait(30);
  stop();
  assert.equal(reloads, 1);
  assert.ok(!states.includes("down"));
  assert.ok(states.includes("up"));
});

test("kick via window event helper is exposed", async () => {
  const stop = startReconnectWatch({ ping: async () => "b1", reload: () => {}, okMs: 10, downMs: 10 });
  assert.equal(typeof window.__picodeKickHealth, "function");
  stop();
  assert.equal(window.__picodeKickHealth, undefined);
});
