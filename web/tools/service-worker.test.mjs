import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { runInNewContext } from "node:vm";
import { test } from "node:test";

const source = readFileSync(new URL("../public/sw.js", import.meta.url), "utf8");
function worker({ hit, ok = true, windows = [], keys = [], storageError = "" } = {}) {
  const handlers = {}, calls = [];
  const response = { ok, clone() { return this; } };
  runInNewContext(source, {
    URL, location: { origin: "https://picode.test" },
    self: {
      addEventListener: (type, fn) => { handlers[type] = fn; },
      skipWaiting: () => calls.push(["skipWaiting"]),
      clients: {
        claim: () => calls.push(["claim"]), matchAll: async () => windows,
        openWindow: url => calls.push(["open", url]),
      },
      registration: { showNotification: (title, options) => calls.push(["notify", title, JSON.parse(JSON.stringify(options))]) },
    },
    fetch: async (request, options) => { calls.push(["fetch", request.url, options?.cache]); return response; },
    caches: {
      keys: async () => keys, delete: async key => calls.push(["delete", key]),
      open: async key => { if (storageError === "open") throw Error("Storage unavailable"); calls.push(["cache", key]); return {
        match: async () => { if (storageError === "match") throw Error("Read failed"); return hit; },
        put: async request => { if (storageError === "put") throw Error("QuotaExceededError"); calls.push(["put", request.url]); },
      }; },
    },
  });
  async function event(type, fields = {}) {
    let response;
    const background = [];
    handlers[type]({ ...fields, waitUntil: p => background.push(p), respondWith: p => { response = p; } });
    const result = await response;
    await Promise.all(background);
    return result;
  }
  return { calls, event };
}

for (const [path, app] of [["/desktop/assets/a.js", "desktop"], ["/mobile/assets/a.css", "mobile"], ["/assets/launcher.js", "launcher"]]) {
  for (const hit of [undefined, {}]) test(`${path}: ${hit ? "cached" : "network"} assets use their own cache`, async () => {
    const sw = worker({ hit });
    await sw.event("fetch", { request: { url: "https://picode.test" + path, method: "GET" } });
    assert.deepEqual(sw.calls.filter(c => c[0] === "cache"), [["cache", "picode-ui-v2-" + app]]);
    assert.equal(sw.calls.some(c => c[0] === "fetch"), !hit);
    assert.equal(sw.calls.some(c => c[0] === "put"), !hit);
  });
}
for (const path of ["/", "/desktop/", "/mobile/", "/sw.js", "/manifest.json"]) test(`${path} always uses fresh HTML/metadata`, async () => {
  const sw = worker();
  await sw.event("fetch", { request: { url: "https://picode.test" + path, method: "GET" } });
  assert.deepEqual(sw.calls, [["fetch", "https://picode.test" + path, "no-store"]]);
});
for (const [url, method] of [["https://picode.test/api/workspaces", "GET"], ["https://picode.test/ws/agent", "GET"], ["https://picode.test/mobile/assets/a.js", "POST"], ["https://other.test/mobile/assets/a.js", "GET"]]) test(`${method} ${url} bypasses caches`, async () => {
  const sw = worker();
  await sw.event("fetch", { request: { url, method } });
  assert.deepEqual(sw.calls, []);
});
test("failed asset responses are never persisted", async () => {
  const sw = worker({ ok: false });
  await sw.event("fetch", { request: { url: "https://picode.test/mobile/assets/a.js", method: "GET" } });
  assert.equal(sw.calls.some(c => c[0] === "put"), false);
});
test("upgrade removes owned obsolete caches and preserves active/unrelated caches", async () => {
  const sw = worker({ keys: ["picode-assets-v1", "picode-ui-v1-mobile", "picode-ui-v2-desktop", "picode-ui-v2-mobile", "another-app"] });
  await sw.event("activate");
  assert.deepEqual(sw.calls, [["delete", "picode-assets-v1"], ["delete", "picode-ui-v1-mobile"], ["claim"]]);
});
for (const [paths, expected] of [
  [["/desktop/", "/mobile/"], "/mobile/"], [["/desktop/"], "/desktop/"], [["/?mobile=1"], "/?mobile=1"], [["/unrelated"], null], [[], null],
]) test(`notification target with ${paths.join(",") || "no windows"}`, async () => {
  const actions = [];
  const windows = paths.map(path => ({ url: "https://picode.test" + path, focus: () => actions.push(["focus", path]), postMessage: msg => actions.push(["navigate", msg.hash]) }));
  const sw = worker({ windows });
  await sw.event("notificationclick", { notification: { close() {}, data: { hash: "#/agent/a" } } });
  if (expected) assert.deepEqual(actions, [["navigate", "#/agent/a"], ["focus", expected]]);
  else assert.deepEqual(sw.calls, [["open", "/mobile/#/agent/a"]]);
});
test("external notification URLs cannot navigate outside the app", async () => {
  const sw = worker();
  await sw.event("notificationclick", { notification: { close() {}, data: { hash: "https://other.test" } } });
  assert.deepEqual(sw.calls, [["open", "/mobile/#/"]]);
});
test("push keeps the existing icon, tag and app deep link", async () => {
  const sw = worker();
  await sw.event("push", { data: { json: () => ({ title: "Agent needs you", tag: "a", hash: "#/agent/a" }) } });
  assert.deepEqual(sw.calls, [["notify", "Agent needs you", { body: "", tag: "a", renotify: true, icon: "/icon-192.png", badge: "/icon-192.png", data: { hash: "#/agent/a" } }]]);
});
test("the installed app identity and registration scope survive the new start URL", () => {
  const manifest = JSON.parse(readFileSync(new URL("../public/manifest.json", import.meta.url), "utf8"));
  assert.equal(manifest.id, "/?mobile=1");
  assert.equal(manifest.scope, "/");
  assert.equal(manifest.start_url, "/mobile/");
});

for (const storageError of ["open", "match", "put"]) test(`storage ${storageError} failure preserves the network response`, async () => {
  const sw = worker({ storageError });
  const response = await sw.event("fetch", { request: { url: "https://picode.test/mobile/assets/a.js", method: "GET" } });
  assert.equal(response.ok, true);
  assert.equal(sw.calls.filter(call => call[0] === "fetch").length, 1);
});
