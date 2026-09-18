import assert from "node:assert/strict";
import { test } from "node:test";
import { createComputerRunner, enabledKeys } from "./computerChannel.js";

test("a computer command reaches the bridge with its principal, action and params", async () => {
  const calls = [];
  const run = createComputerRunner({
    invoke: async (cmd, args) => {
      calls.push([cmd, args]);
      return { ok: true, image: "abc" };
    },
  });
  const result = await run({ id: "c1", kind: "computer", method: "left_click", params: { coordinate: [1, 2] }, principal: "agent-1" });
  assert.deepEqual(calls, [["computer_call", { principal: "agent-1", action: "left_click", paramsJson: '{"coordinate":[1,2]}' }]]);
  assert.deepEqual(result, { output: { ok: true, image: "abc" } });
});

test("a shell refusal travels verbatim, and missing pieces are named", async () => {
  const run = createComputerRunner({
    invoke: async () => {
      throw new Error("disabled: this agent may not use the computer");
    },
  });
  assert.deepEqual(await run({ method: "screenshot", principal: "agent-1" }), { error: "disabled: this agent may not use the computer" });
  assert.match((await run({ method: "screenshot", principal: " " })).error, /^disabled:/);
  assert.match((await createComputerRunner({ invoke: null })({ method: "screenshot", principal: "a" })).error, /^not_connected/);
});

test("enabledKeys mirrors only the switches that are on, in the daemon's namespace", () => {
  assert.deepEqual(
    enabledKeys([
      { kind: "agent", agentId: "a1", enabled: true },
      { kind: "agent", agentId: "a2", enabled: false },
      { kind: "terminal", termId: "t9", enabled: true },
      { kind: "agent", agentId: "a1", enabled: true },
      null,
    ]),
    ["a1", "term:t9"],
  );
  assert.deepEqual(enabledKeys(undefined), []);
});
