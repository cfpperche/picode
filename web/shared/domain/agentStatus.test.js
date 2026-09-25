import assert from "node:assert/strict";
import { test } from "node:test";
import { agentRowStatus, agentStatusLabel, agentStatusStamp, agentTerm, bucketAgentsByState } from "./agentStatus.js";

test("bound Pi uses terminal activity, while RPC ignores it", () => {
  const agent = { id:"pi", cli:"pi", terminalId:"t", mode:"interactive", streaming:true };
  for (const [state,want] of [["working","working"],["compacting","compacting"],["needs-you","needs-you"],["idle","ready"],["","open"]]) {
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
  assert.equal(agentStatusLabel("compacting"), "Compacting");
});

// The "Working first" view without a stamp: buckets rank by who needs the
// reader, rows keep their stored position inside a bucket, and the view may
// reorder but never lose a row — a status outside the vocabulary still
// renders.
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

// The "Working first" view with the park-time ranking (ADR-0173 amendment,
// 2026-09-24): every bucket whose rows are not working sorts by how long an
// agent has been parked in its status, shortest park first — the same stamp
// the pill shows. The working bucket keeps the stored order, rows without a
// truthful stamp keep theirs after the stamped ones, and equal stamps stay
// in the stored order.
test("bucketAgentsByState sorts the not-working buckets by park time", () => {
  const agents = [
    { id: "old", mode: "managed", lastStatusAt: "2026-09-23T09:00:00Z" },
    { id: "w1", mode: "managed", streaming: true, lastStartedAt: "2026-09-23T10:00:00Z" },
    { id: "fresh", mode: "managed", lastStatusAt: "2026-09-23T10:05:00Z" },
    { id: "w2", mode: "managed", streaming: true, lastStartedAt: "2026-09-23T09:30:00Z" },
    { id: "nostamp", mode: "managed" },
    { id: "stopped", mode: "stopped", lastStatusAt: "2026-09-23T10:00:00Z" },
  ];
  const stampOf = (a) => agentStatusStamp(agentRowStatus(a), a, null);
  const buckets = bucketAgentsByState(agents, (a) => agentRowStatus(a), stampOf);
  assert.deepEqual(buckets.map((b) => b.status), ["working", "ready", "stopped"]);
  assert.deepEqual(buckets[0].agents.map((a) => a.id), ["w1", "w2"], "working keeps the stored order, not park time");
  assert.deepEqual(buckets[1].agents.map((a) => a.id), ["fresh", "old", "nostamp"], "shortest park first, unstamped keeps its place last");
  assert.deepEqual(buckets[2].agents.map((a) => a.id), ["stopped"]);
  const tied = [
    { id: "a", mode: "managed", lastStatusAt: "2026-09-23T10:00:00Z" },
    { id: "b", mode: "managed", lastStatusAt: "2026-09-23T10:00:00Z" },
  ];
  assert.deepEqual(
    bucketAgentsByState(tied, (a) => agentRowStatus(a), stampOf)[0].agents.map((a) => a.id),
    ["a", "b"],
    "equal parks keep the stored order",
  );
});

// The pill's age, one stamp per status: the terminal hook's stateAt for
// live states, the store's writes for managed ones, and no invented clock —
// never createdAt, and stopped never reads the pre-stop terminal state.
test("agentStatusStamp: the age is when the current state began", () => {
  const term = (state, extra = {}) => ({ id: "t", state, stateAt: "2026-09-23T10:05:00Z", running: true, ...extra });
  const ag = { lastStatusAt: "2026-09-23T09:00:00Z", lastStartedAt: "2026-09-23T08:59:00Z", createdAt: "2026-09-01T00:00:00Z" };
  assert.equal(agentStatusStamp("ready", ag, term("idle")), "2026-09-23T10:05:00Z");
  assert.equal(agentStatusStamp("working", ag, term("working")), "2026-09-23T10:05:00Z");
  assert.equal(agentStatusStamp("needs-you", ag, term("needs-you")), "2026-09-23T10:05:00Z");
  assert.equal(agentStatusStamp("ready", ag, null), "2026-09-23T09:00:00Z", "managed ready reads the settle/stop write");
  assert.equal(agentStatusStamp("working", ag, null), "2026-09-23T08:59:00Z", "managed working reads the runtime start");
  assert.equal(agentStatusStamp("needs-you", ag, null), "", "the managed ask has no timestamp on the fleet row yet");
  assert.equal(agentStatusStamp("stopped", ag, term("idle", { running: false })), "2026-09-23T09:00:00Z", "stopped reads the stop write, not the stale state");
  assert.equal(agentStatusStamp("stopped", null, null), "");
  assert.equal(agentStatusStamp("interactive", ag, term("idle")), "");
  assert.equal(agentStatusStamp("ready", { createdAt: "2026-09-01T00:00:00Z" }, null), "", "createdAt is not a state age");
  assert.equal(agentStatusStamp("ready", null, null), "");
});

test("agentTerm reads the bound terminal or the fleet copy", () => {
  const live = [{ id: "t1", state: "working" }];
  assert.equal(agentTerm({ terminalId: "t1" }, live), live[0]);
  assert.equal(agentTerm({ terminalId: "t9", terminal: { id: "t9" } }, live).id, "t9");
  assert.equal(agentTerm({ terminalId: "t1", mode: "managed" }, live), null, "managed pi has no pill terminal");
  assert.equal(agentTerm(null, live), null);
});
