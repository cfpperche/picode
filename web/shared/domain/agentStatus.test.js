import assert from "node:assert/strict";
import { test } from "node:test";
import { agentRowStatus, agentStatusLabel } from "./agentStatus.js";

// The sidebar's derivation, one row per word: a dialog beats streaming,
// streaming beats the mode, and the mode decides the rest.
test("agentRowStatus: needs-you > working > mode", () => {
  const a = { id: "a1", mode: "managed" };
  assert.equal(agentRowStatus({ ...a, waiting: true, streaming: true }), "needs-you");
  assert.equal(agentRowStatus(a, { waitingId: "a1" }), "needs-you");
  assert.equal(agentRowStatus({ ...a, streaming: true }), "working");
  assert.equal(agentRowStatus(a, { workingId: "a1" }), "working");
  assert.equal(agentRowStatus({ ...a, mode: "interactive" }, { workingIds: ["a1"] }), "working");
  assert.equal(agentRowStatus({ ...a, mode: "interactive" }), "interactive");
  assert.equal(agentRowStatus({ ...a, mode: "stopped" }), "stopped");
  assert.equal(agentRowStatus({ id: "a1" }), "stopped", "no mode reads as stopped");
  assert.equal(agentRowStatus(a), "ready");
  assert.equal(agentRowStatus(a, { workingIds: ["other"], waitingId: "other" }), "ready");
  assert.equal(agentRowStatus(null), "stopped");
});

test("agentStatusLabel: the chip's five words", () => {
  assert.deepEqual(["needs-you", "working", "interactive", "stopped", "ready"].map(agentStatusLabel), ["Needs you", "Working", "In terminal", "Stopped", "Ready"]);
  assert.equal(agentStatusLabel("anything"), "Ready");
});
