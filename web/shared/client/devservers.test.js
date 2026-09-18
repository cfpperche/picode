import assert from "node:assert/strict";
import { test } from "node:test";
import { hideDevServer, isLoopbackUrl, listDevServers, stopDevServer, unhideDevServer } from "./devservers.js";

test("isLoopbackUrl answers this machine, and only this machine", () => {
  const yes = [
    "http://localhost:5173/",
    "http://localhost/",
    "http://127.0.0.1:3000",
    "https://127.0.0.1:8443/app",
    "http://[::1]:8080/",
    "http://127.1.2.3:9000/",
    "http://myapp.localhost:5173/",
  ];
  const no = [
    "https://example.com/",
    "http://192.168.1.5:5173/",
    "http://10.0.0.7/",
    "ftp://localhost/",
    "javascript:alert(1)",
    "not a url",
    "",
    null,
  ];
  for (const url of yes) assert.equal(isLoopbackUrl(url), true, `want true: ${url}`);
  for (const url of no) assert.equal(isLoopbackUrl(url), false, `want false: ${url}`);
});

test("listDevServers reads the guarded route, and asks for a refresh when told", async () => {
  const calls = [];
  const original = globalThis.fetch;
  globalThis.fetch = async (url, opts) => {
    calls.push({ url, opts });
    return { ok: true, status: 200, statusText: "OK", json: async () => ({ servers: [{ port: 5173 }] }) };
  };
  try {
    const page = await listDevServers();
    assert.equal(page.servers[0].port, 5173);
    assert.equal(calls[0].url, "/api/devservers");
    await listDevServers({ refresh: true });
    assert.equal(calls[1].url, "/api/devservers?refresh=1");
  } finally {
    globalThis.fetch = original;
  }
});

// The panel's three verbs travel with the row's identity — the port, the pid
// and the process's start token — because that is what the server re-derives
// before it signals or hides anything.
test("the panel's writes carry the identity of the process", async () => {
  const calls = [];
  const original = globalThis.fetch;
  globalThis.fetch = async (url, opts) => {
    calls.push({ url, opts, body: opts && opts.body ? JSON.parse(opts.body) : null });
    return { ok: true, status: 200, statusText: "OK", json: async () => ({}) };
  };
  try {
    await stopDevServer({ port: 45683, pid: 3257366, startKey: "987654" });
    assert.equal(calls[0].url, "/api/devservers/stop");
    assert.equal(calls[0].opts.method, "POST");
    assert.deepEqual(calls[0].body, { port: 45683, pid: 3257366, startKey: "987654", force: false });
    await stopDevServer({ port: 45683, pid: 3257366, startKey: "987654", force: true });
    assert.equal(calls[1].body.force, true);
    await hideDevServer({ port: 45683, pid: 3257366, startKey: "987654" });
    assert.equal(calls[2].url, "/api/devservers/hide");
    assert.deepEqual(calls[2].body, { port: 45683, pid: 3257366, startKey: "987654" });
    await unhideDevServer("12");
    assert.equal(calls[3].url, "/api/devservers/unhide");
    assert.deepEqual(calls[3].body, { hideId: 12 });
  } finally {
    globalThis.fetch = original;
  }
});
