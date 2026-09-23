import assert from "node:assert/strict";
import { test } from "node:test";
import { agentRowStatus, agentStatusLabel, agentTerm, bucketAgentsByState } from "./agentStatus.js";

test("bound Pi uses terminal activity, while RPC ignores it", () => {
  const agent = { id:"pi", cli:"pi", terminalId:"t", mode:"interactive", streaming:true };
  for (const [state,want] of [["working","working"],["needs-you","needs-you"],["idle","ready"],["","open"]]) {
    const terminal = { id:"t", running:true, cli:"pi", state };
    assert.equal(agentRowStatus({...agent,terminal}),want);
    assert.equal(agentRowStatus({...agent,terminal,mode:"managed"}),"working");
  }
  assert.equal(agentRowStatus({...agent,terminal:{id:"t",running:false}}),"stopped");
});

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

// The "Working first" view: buckets rank by who needs the reader, rows keep
// their stored position inside a bucket, and the view may reorder but never
// lose a row — a status outside the vocabulary still renders.
test("bucketAgentsByState ranks who needs the reader first", () => {
  const agents = [
    { id: "r", mode: "managed" },
    { id: "w1", mode: "managed", streaming: true },
    { id: "n", mode: "managed", waiting: true },
    { id: "w2", mode: "managed", streaming: true },
    { id: "s", mode: "stopped" },
    { id: "i", mode: "interactive" },
  ];
  const buckets = bucketAgentsByState(agents, (a) => agentRowStatus(a));
  assert.deepEqual(buckets.map((b) => b.status), ["needs-you", "working", "interactive", "ready", "stopped"]);
  assert.deepEqual(buckets[1].agents.map((a) => a.id), ["w1", "w2"], "stored position rules inside a bucket");
  assert.deepEqual(bucketAgentsByState([], () => "ready"), []);
  const odd = bucketAgentsByState([{ id: "o" }], () => "mystery");
  assert.deepEqual(odd.map((b) => b.status), ["mystery"]);
  assert.deepEqual(odd[0].agents.map((a) => a.id), ["o"]);
});

test("agentTerm reads the bound terminal or the fleet copy", () => {
  const live = [{ id: "t1", state: "working" }];
  assert.equal(agentTerm({ terminalId: "t1" }, live), live[0]);
  assert.equal(agentTerm({ terminalId: "t9", terminal: { id: "t9" } }, live).id, "t9");
  assert.equal(agentTerm({ terminalId: "t1", mode: "managed" }, live), null, "managed pi has no pill terminal");
  assert.equal(agentTerm({ legacyInteractive: true, terminalId: "t1" }, live), null);
  assert.equal(agentTerm(null, live), null);
});
