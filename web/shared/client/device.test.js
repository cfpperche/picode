import assert from "node:assert/strict";
import { test } from "node:test";
import { startPresence } from "./device.js";

for (const app of ["desktop", "mobile"]) test(`${app} presence reports its application identity at every heartbeat`, async t => {
  const calls = [];
  const previous = Object.getOwnPropertyDescriptor(globalThis, "localStorage");
  Object.defineProperty(globalThis, "localStorage", { configurable: true, value: { getItem: () => "device-a" } });
  t.after(() => {
    if (previous) Object.defineProperty(globalThis, "localStorage", previous);
    else delete globalThis.localStorage;
  });
  t.mock.method(globalThis, "fetch", async (url, options) => { calls.push([url, JSON.parse(options.body)]); return { ok: true, status: 204 }; });
  let tick;
  t.mock.method(globalThis, "setInterval", fn => { tick = fn; return 5; });
  const clear = t.mock.method(globalThis, "clearInterval", () => {});
  const stop = startPresence(app);
  tick();
  stop();
  assert.deepEqual(calls, Array.from({ length: 2 }, () => ["/api/devices/ping", { id: "device-a", host: app === "desktop" }]));
  assert.deepEqual(clear.mock.calls[0].arguments, [5]);
});
