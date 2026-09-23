import { test } from "node:test";
import assert from "node:assert/strict";
import { createLaunchAgent } from "./launchAgent.js";

function recorder(replies) {
  const calls = [];
  const request = async (url, opts) => {
    calls.push({ url, body: JSON.parse(opts.body) });
    const r = replies[url];
    if (r instanceof Error) throw r;
    return r;
  };
  return { calls, request };
}

test("free launch posts /api/agents with path and starts the terminal", async () => {
  const { calls, request } = recorder({ "/api/agents": { id: "a1", terminalId: "t1" }, "/api/terminals/t1/launch/start": {} });
  const out = await createLaunchAgent({ cli: "codex", name: "C", workspaceId: "", folder: "/w", overrides: { args: ["resume", "x"] } }, request);
  assert.deepEqual(calls[0], { url: "/api/agents", body: { cli: "codex", name: "C", overrides: { args: ["resume", "x"] }, path: "/w" } });
  assert.equal(calls[1].url, "/api/terminals/t1/launch/start");
  assert.deepEqual(out, { agent: { id: "a1", terminalId: "t1" }, terminalId: "t1", launchError: "" });
});

test("workspace launch uses workPath; a failed start is reported, not thrown", async () => {
  const { calls, request } = recorder({ "/api/workspaces/w%201/agents": { id: "a2", terminalId: "t2" }, "/api/terminals/t2/launch/start": new Error("boom") });
  const out = await createLaunchAgent({ cli: "claude-code", workspaceId: "w 1", folder: "/p/sub" }, request);
  assert.equal(calls[0].body.workPath, "/p/sub");
  assert.deepEqual(calls[0].body.overrides, {});
  assert.equal(out.launchError, "boom");
});

test("an agent without a terminal is not started", async () => {
  const { calls, request } = recorder({ "/api/agents": { id: "a3" } });
  const out = await createLaunchAgent({ cli: "pi", workspaceId: "ws_free" }, request);
  assert.equal(calls.length, 1);
  assert.equal(out.terminalId, "");
});

test("a plain Pi launch is the palette's Pi agent: no launch overrides, no start", async () => {
  const { calls, request } = recorder({ "/api/agents": { id: "p1", cli: "pi" } });
  const out = await createLaunchAgent({ cli: "pi", name: "P", workspaceId: "", overrides: {} }, request);
  assert.equal(calls.length, 1);
  assert.equal("overrides" in calls[0].body, false);
  assert.equal(out.terminalId, "");
});

test("a Pi launch with a profile keeps its overrides", async () => {
  const { calls, request } = recorder({ "/api/agents": { id: "p2", terminalId: "t2" }, "/api/terminals/t2/launch/start": {} });
  await createLaunchAgent({ cli: "pi", overrides: { args: ["--verbose"] } }, request);
  assert.deepEqual(calls[0].body.overrides, { args: ["--verbose"] });
});
