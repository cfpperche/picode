import { test } from "node:test";
import assert from "node:assert/strict";
import { checklistLine, applyChecklists, indexChecklists, checklistItems, checklistRefusal, currentStep } from "./checklist.js";
import { summarizeArgs } from "./toolArgs.js";
import { stepLabel } from "./turns.js";

test("line: in-progress step, else first pending, else n/n, else absent, else nothing", () => {
  assert.deepEqual(checklistLine({ items: [{ text: "a", status: "completed" }, { text: "b", status: "in-progress" }, { text: "c" }] }), { kind: "step", text: "b", position: 2, total: 3 });
  assert.deepEqual(checklistLine({ items: [{ text: "a", status: "completed" }, { text: "c", status: "pending" }] }), { kind: "step", text: "c", position: 2, total: 2 });
  assert.deepEqual(checklistLine({ items: [{ text: "a", status: "completed" }] }), { kind: "step", text: "a", position: 1, total: 1 });
  assert.deepEqual(checklistLine({ items: [], absent: true }), { kind: "absent" });
  assert.equal(checklistLine({ items: [] }), null);
  assert.equal(checklistLine(null), null);
  assert.equal(currentStep(undefined), null);
});

test("reset: the clear event lands as silence — no line until a real list arrives", () => {
  // The daemon answers a reset with the empty, non-absent checklist; the
  // row keeps its place in the map but renders nothing.
  const m0 = indexChecklists([{ agentId: "a", items: [{ text: "old task", status: "in-progress" }] }]);
  assert.equal(checklistLine(m0.a).kind, "step");
  const m1 = applyChecklists(m0, { type: "agent.checklist", data: { agentId: "a", items: [], absent: false } });
  assert.equal(checklistLine(m1.a), null);
  // And a boot fetch after the reset no longer lists the agent.
  assert.deepEqual(indexChecklists([]), {});
});

test("map: feed events replace, delete drops, others untouched", () => {
  const m0 = indexChecklists([{ agentId: "a", items: [{ text: "x", status: "pending" }] }, { nope: true }]);
  assert.deepEqual(Object.keys(m0), ["a"]);
  const m1 = applyChecklists(m0, { type: "agent.checklist", data: { agentId: "b", items: [] , absent: true } });
  assert.equal(Object.keys(m1).length, 2);
  const m2 = applyChecklists(m1, { type: "agent.deleted", data: { id: "a" } });
  assert.deepEqual(Object.keys(m2), ["b"]);
  assert.equal(applyChecklists(m2, { type: "agent.status", data: { id: "b" } }), m2);
  assert.deepEqual(applyChecklists(null, { type: "feed.open" }), {});
});

test("chat items: result details win over call args; bad rows dropped", () => {
  assert.deepEqual(checklistItems({ toolArgs: { items: [{ text: "a" }, { nope: 1 }, { text: "b", status: "weird" }] } }), [{ text: "a", status: "pending" }, { text: "b", status: "pending" }]);
  assert.deepEqual(checklistItems({ toolArgs: { items: [{ text: "a" }] }, result: { details: { items: [{ text: "z", status: "completed" }] } } }), [{ text: "z", status: "completed" }]);
  assert.deepEqual(checklistItems({}), []);
});

test("refusal: the failed call's line is visible in both live and replay shapes", () => {
  const line = "[pi-checklist] A checklist is required before the first change of this task.";
  // Live stream: the whole RPC result rides it.result; detail is JSON.
  const live = {
    status: "error",
    detail: JSON.stringify({ content: [{ type: "text", text: line }], isError: true }),
    result: { content: [{ type: "text", text: line }], details: { items: [] }, isError: true },
  };
  assert.equal(checklistRefusal(live), line);
  // Replayed transcript: the text is flattened into detail; result is details only.
  const replay = { status: "error", detail: line, result: { items: [{ text: "old", status: "completed" }] } };
  assert.equal(checklistRefusal(replay), line);
  // Healthy calls have no refusal, and a JSON detail never leaks as one.
  assert.equal(checklistRefusal({ status: "ok", detail: line }), "");
  assert.equal(checklistRefusal({ status: "error", detail: '{"content":[]}' }), "");
  assert.equal(checklistRefusal({}), "");
});

test("a checklist call summarizes as progress and step, and the turn step reads Checklist", () => {
  const args = { items: [{ text: "read", status: "completed" }, { text: "edit the file", status: "in-progress" }] };
  assert.equal(summarizeArgs(args), "1/2 · edit the file");
  assert.equal(summarizeArgs({ items: [] }), "");
  assert.equal(stepLabel({ kind: "tool", name: "checklist", args: summarizeArgs(args) }), "Checklist 1/2 · edit the file");
  assert.equal(stepLabel({ kind: "tool", name: "checklist", args: "" }), "Wrote the checklist");
});
