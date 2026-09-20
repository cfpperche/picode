import { test } from "node:test";
import assert from "node:assert/strict";
import { saveAgentConfig } from "./agentConfig.js";

for (const [mode, patch, actions] of [
  ["stopped", { opMode: "readonly" }, [""]],
  ["managed", { model: "new" }, [""]],
  ["interactive", { checklist: "always" }, [""]],
  ["managed", { opMode: "full" }, [""]],
  ["managed", { opMode: "readonly" }, ["", "/managed/stop", "/managed/start"]],
  ["interactive", { opMode: "readonly" }, ["", "/close", "/open"]],
]) test(`${mode} configuration ${JSON.stringify(patch)}`, async () => {
  const calls = [], closed = [];
  await saveAgentConfig({ id: "a/b", mode }, patch, async (url, options) => calls.push({ url, ...options }), id => closed.push(id));
  assert.deepEqual(calls.map(c => c.url), actions.map(action => "/api/agents/a%2Fb" + action));
  assert.deepEqual(JSON.parse(calls[0].body), patch);
  assert.equal(calls[0].method, "PATCH");
  assert.ok(calls.slice(1).every(c => c.method === "POST"));
  assert.deepEqual(closed, mode === "interactive" && actions.length > 1 ? ["sh:a/b"] : []);
});
for (const failure of ["", "/managed/stop", "/managed/start"]) test(`failure at ${failure || "save"} stops the sequence and reports partial success honestly`, async () => {
  const calls = [];
  await assert.rejects(() => saveAgentConfig({ id: "a", mode: "managed" }, { opMode: "readonly" }, async url => {
    calls.push(url);
    if (url === "/api/agents/a" + failure) throw Error("Offline");
  }), failure ? /Settings saved, but/ : /Offline/);
  assert.equal(calls.at(-1), "/api/agents/a" + failure);
});
test("missing agent never writes", async () => {
  await assert.rejects(() => saveAgentConfig(null, {}, () => assert.fail("unexpected write")), /Select an agent/);
});

for (const failure of ["", "/close", "/open"]) test("legacy configuration migration cleanup: " + (failure || "success"), async () => {
  const closed = [], calls = [];
  const run = () => saveAgentConfig({ id: "a", terminalId: "t", mode: "interactive", legacyInteractive: true }, { opMode: "readonly" }, async url => {
    calls.push(url);
    if (failure && url.endsWith(failure)) throw Error("Offline");
  }, key => closed.push(key));
  if (failure) await assert.rejects(run, /Settings saved, but/); else await run();
  assert.deepEqual(closed, failure === "/close" ? [] : ["sh:a", "sh:t"]);
  if (failure === "/close") assert.equal(calls.some(url => url.endsWith("/open")), false);
});
