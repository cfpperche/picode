import test from "node:test";
import assert from "node:assert/strict";
import { connection, confirmOperation, reviewAndExecute } from "../src/client.ts";

test("connection uses the configured PiCode, token, and per-request TLS policy", () => {
  const local = connection({}, "/tmp/home", (path) => path.endsWith("server.json") ? '{"url":"https://localhost:8445"}' : "a".repeat(64));
  assert.equal(local.base, "https://localhost:8445");
  assert.equal(local.rejectUnauthorized, false);
  const remote = connection({ PICODE_URL: "https://server.example", PICODE_TOKEN: "b".repeat(64) }, "/tmp/home", () => { throw new Error("must not read local files"); });
  assert.equal(remote.rejectUnauthorized, true);
  assert.equal(remote.token, "b".repeat(64));
  assert.throws(() => connection({ PICODE_URL: "http://server.example" }, "/tmp/home", () => ""), /HTTPS/);
  assert.throws(() => connection({ PICODE_URL: "https://user:password@server.example" }, "/tmp/home", () => ""), /Invalid/);
});

test("maintenance confirmation: decline, unavailable UI, and approval", async () => {
  const plan = { id: "plan-qa", title: "Stop demo", impact: "Interrupt connections", steps: [{ action: "stop", name: "database", target: "a".repeat(64), state: "queued" }] };
  for (const mode of ["decline", "unavailable", "approve"]) {
    const calls: { path: string; body: any }[] = [];
    const ctx = mode === "unavailable" ? {} : { hasUI: true, ui: { confirm: async (title: string, message: string) => { assert.equal(title, plan.title); assert.ok(message.includes(plan.steps[0].target)); return mode === "approve"; } } };
    const response = await reviewAndExecute(plan, ctx, "request-qa", "agent-qa", async (_method, path, body) => { calls.push({ path, body }); return { inboxId: "review-qa", state: "running" }; });
    if (mode === "decline") { assert.equal(calls.length, 0); assert.equal(response.cancelled, true); }
    if (mode === "unavailable") { assert.deepEqual(calls, [{ path: "/api/docker/plans/plan-qa/review", body: {} }]); assert.equal(response.waitingForReview, true); }
    if (mode === "approve") { assert.deepEqual(calls, [{ path: "/api/docker/jobs", body: { planId: plan.id, requestKey: "request-qa", approved: true, agentId: "agent-qa" } }]); assert.equal(response.state, "running"); }
  }
});

test("Docker operation confirmation matrix", async () => {
  assert.equal(await confirmOperation("start", "qa", {}), true);
  for (const action of ["stop", "restart"]) {
    await assert.rejects(confirmOperation(action, "qa", {}), /Docker App/);
    let calls = 0;
    const ctx = { hasUI: true, ui: { confirm: async (title: string) => { calls++; assert.match(title, /qa/); return false; } } };
    assert.equal(await confirmOperation(action, "qa", ctx), false);
    assert.equal(calls, 1);
    ctx.ui.confirm = async () => true;
    assert.equal(await confirmOperation(action, "qa", ctx), true);
  }
  await assert.rejects(confirmOperation("delete", "qa", {}), /Unsupported/);
});
