import { test } from "node:test";
import assert from "node:assert/strict";
import { bootReads } from "./bootReads.js";

const FAST = {
  "/api/workspaces": [{ id: "w1" }],
  "/api/agents?free=1": [{ id: "a1" }],
  "/api/terminals": { terminals: [{ id: "t1" }] },
  "/api/apps": [{ id: "inbox" }],
  "/api/system": { host: { name: "box" } },
  "/api/version": { version: "1" },
  "/api/clis": { clis: [] },
  "/api/catalog": { providers: [] },
};

// A fake api: each path answers after `delay[path]` ms (default 1), or
// rejects when listed in `fail`. `calls` counts every request.
function fakeApi({ delay = {}, fail = [] } = {}) {
  const calls = [];
  const api = (path) => {
    calls.push(path);
    return new Promise((resolve, reject) => setTimeout(() => (fail.includes(path) ? reject(new Error(path + " failed")) : resolve(FAST[path])), delay[path] ?? 1));
  };
  return { api, calls };
}

test("the fleet does not wait for a slow catalog", async () => {
  const { api } = fakeApi({ delay: { "/api/catalog": 400, "/api/clis": 300, "/api/system": 300 } });
  const t0 = Date.now();
  const { fleet, side } = bootReads(api);
  const f = await fleet;
  assert.ok(Date.now() - t0 < 150, "fleet waited " + (Date.now() - t0) + " ms");
  assert.deepEqual(f.workspaces, [{ id: "w1" }]);
  assert.deepEqual(f.terminals, [{ id: "t1" }]);
  assert.equal(f.appsOk, true);
  assert.deepEqual(await side.catalog, { providers: [] });
});

test("side reads failing never touch the fleet", async () => {
  const { api } = fakeApi({ fail: ["/api/catalog", "/api/clis", "/api/system"] });
  const { fleet, side } = bootReads(api);
  // Handlers go on at once, as the App's .then/.catch do.
  const outcomes = Promise.allSettled([side.catalog, side.clis, side.system]);
  assert.deepEqual((await fleet).freeAgents, [{ id: "a1" }]);
  assert.deepEqual((await outcomes).map((o) => o.status), ["rejected", "rejected", "rejected"]);
});

// | read failing        | fleet                          |
// | ------------------- | ------------------------------ |
// | free agents         | freeAgents: []                 |
// | terminals           | terminals: []                  |
// | apps                | apps: null, appsOk: false      |
// | workspaces          | rejects (the boot's catch)     |
test("fleet failures follow the boot's old rules", async () => {
  const quiet = (side) => Object.values(side).forEach((p) => p.catch(() => {}));
  let r = bootReads(fakeApi({ fail: ["/api/agents?free=1", "/api/terminals", "/api/apps"] }).api);
  quiet(r.side);
  const f = await r.fleet;
  assert.deepEqual([f.freeAgents, f.terminals, f.apps, f.appsOk], [[], [], null, false]);
  r = bootReads(fakeApi({ fail: ["/api/workspaces"] }).api);
  quiet(r.side);
  await assert.rejects(r.fleet, /workspaces failed/);
});

test("every read is asked once; the catalog only after the fleet", async () => {
  const { api, calls } = fakeApi();
  const r = bootReads(api);
  Object.values(r.side).forEach((p) => p.catch(() => {}));
  assert.ok(!calls.includes("/api/catalog"), "the catalog must not compete with the fleet's reads");
  assert.deepEqual([...calls].sort(), Object.keys(FAST).filter((k) => k !== "/api/catalog").sort());
  await r.fleet;
  await r.side.catalog;
  assert.equal(calls.filter((c) => c === "/api/catalog").length, 1);
  assert.equal(calls.filter((c) => c === "/api/agents?free=1").length, 1);
});

test("a failed fleet still asks for the catalog", async () => {
  const { api, calls } = fakeApi({ fail: ["/api/workspaces"] });
  const r = bootReads(api);
  await assert.rejects(r.fleet);
  await r.side.catalog;
  assert.ok(calls.includes("/api/catalog"));
  await r.side.system; await r.side.clis;
});
