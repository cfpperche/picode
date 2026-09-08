import assert from "node:assert/strict";
import { test } from "node:test";
import { groupTurns } from "./turns.js";
import {
  agentFinishNotice,
  changeMeta,
  changeTotals,
  finishStatus,
  lastReplyLine,
  noticeDuration,
  normalizeNotice,
  plainNotice,
  suppressNotice,
} from "./notice.js";

test("plainNotice keeps the legacy three kinds", () => {
  assert.equal(plainNotice("Saved.", "ok").level, "ok");
  assert.equal(plainNotice("Heads up", "info").level, "info");
  assert.equal(plainNotice("Boom", "err").level, "error");
  // The default at every legacy call site is an error.
  assert.equal(plainNotice("Boom").level, "error");
  assert.equal(plainNotice("Saved.", "ok").title, "Saved.");
});

test("normalizeNotice clamps the payload", () => {
  const n = normalizeNotice({
    level: "nonsense",
    title: "x".repeat(500),
    meta: [{ text: "a" }, { text: "b" }, { text: "c" }, { text: "d" }, { text: "e" }],
    actions: [
      { label: "One", hash: "#/1" },
      { label: "Two", hash: "#/2" },
      { label: "Three", hash: "#/3" },
    ],
  });
  assert.equal(n.level, "info");
  assert.equal(n.title.length, 400);
  assert.equal(n.meta.length, 4);
  assert.equal(n.actions.length, 2);
  // An action with no way out is not an action.
  assert.deepEqual(normalizeNotice({ actions: [{ label: "Nowhere" }] }).actions, []);
  // An actor without a name cannot identify anything.
  assert.equal(normalizeNotice({ actor: { kind: "agent" } }).actor, null);
});

test("noticeDuration: alerts outlive confirmations, busy waits for its outcome", () => {
  const base = 4000;
  assert.equal(noticeDuration(plainNotice("Saved.", "ok"), base), base);
  assert.equal(noticeDuration(plainNotice("Boom", "err"), base), 12000);
  // Three times a long preference, capped so nothing walls the screen off.
  assert.equal(noticeDuration(plainNotice("Boom", "err"), 15000), 30000);
  assert.equal(noticeDuration(normalizeNotice({ level: "busy" }), base), Infinity);
  // A way out needs long enough to read the way out.
  const withAction = normalizeNotice({ level: "ok", actions: [{ label: "Open", hash: "#/a" }] });
  assert.equal(noticeDuration(withAction, base), 8000);
  assert.equal(noticeDuration(withAction, 20000), 20000);
  // An explicit duration always wins.
  assert.equal(noticeDuration(normalizeNotice({ level: "error", duration: 1500 }), base), 1500);
});

test("suppressNotice: never announce the focused surface, always announce an alert", () => {
  const ok = normalizeNotice({ level: "ok", title: "Done", target: "#/agent/a1" });
  assert.equal(suppressNotice(ok, { target: "#/agent/a1", focused: true }), true);
  assert.equal(suppressNotice(ok, { target: "#/agent/a2", focused: true }), false);
  // Window in the background: the user is not looking at anything.
  assert.equal(suppressNotice(ok, { target: "#/agent/a1", focused: false }), false);
  assert.equal(suppressNotice(ok, null), false);
  // Untargeted notices have no surface to be redundant with.
  assert.equal(suppressNotice(normalizeNotice({ level: "ok" }), { target: "#/", focused: true }), false);
  const bad = normalizeNotice({ level: "error", title: "Boom", target: "#/agent/a1" });
  assert.equal(suppressNotice(bad, { target: "#/agent/a1", focused: true }), false);
});

test("changeTotals sums the turn's edits per file", () => {
  const turn = {
    work: [
      { kind: "tool", name: "read" },
      { kind: "tool", name: "edit", change: { path: "a.js", add: 40, del: 1 } },
      { kind: "tool", name: "edit", change: { path: "a.js", add: 4, del: 0 } },
      { kind: "tool", name: "write", change: { path: "b.js", add: 2, del: 0 } },
    ],
  };
  assert.deepEqual(changeTotals(turn), { files: 2, add: 46, del: 1 });
  assert.deepEqual(changeTotals({ work: [] }), { files: 0, add: 0, del: 0 });
  assert.deepEqual(changeTotals(null), { files: 0, add: 0, del: 0 });
});

test("changeMeta renders the footer chips, and nothing when nothing changed", () => {
  assert.deepEqual(changeMeta({ files: 2, add: 46, del: 1 }), [
    { text: "2 files", tone: "muted" },
    { text: "+46", tone: "add" },
    { text: "−1", tone: "del" },
  ]);
  assert.deepEqual(changeMeta({ files: 1, add: 3, del: 0 }), [
    { text: "1 file", tone: "muted" },
    { text: "+3", tone: "add" },
  ]);
  assert.deepEqual(changeMeta({ files: 0, add: 0, del: 0 }), []);
});

test("finishStatus claims a duration only when there is one", () => {
  assert.equal(finishStatus(7000), "finished · worked for 7s");
  assert.equal(finishStatus(95000), "finished · worked for 1m 35s");
  assert.equal(finishStatus(120), "finished");
  assert.equal(finishStatus(0), "finished");
});

test("lastReplyLine takes the agent's own sentence, skipping headings and fences", () => {
  const turn = {
    replies: [
      { kind: "block", cls: "", text: "first" },
      { kind: "block", cls: "thinking", text: "not this" },
      { kind: "block", cls: "", text: "## Result\n```sh\nx\n```\nPushed and opened a draft PR." },
    ],
  };
  assert.equal(lastReplyLine(turn), "Pushed and opened a draft PR.");
  assert.equal(lastReplyLine({ replies: [] }), "");
  // An alert is not a sentence the agent said.
  assert.equal(lastReplyLine({ replies: [{ kind: "alert", text: "boom" }] }), "");
});

test("agentFinishNotice builds the card from the turn alone", () => {
  const [turn] = groupTurns([
    { kind: "block", cls: "user", text: "ship it", ts: 1000 },
    { kind: "tool", name: "edit", change: { path: "a.js", add: 46, del: 1 }, ts: 3000 },
    { kind: "block", cls: "", text: "Pushed and opened a draft PR.", ts: 8000 },
  ]);
  const n = agentFinishNotice({
    agent: { id: "a1", name: "claude", cli: "claude-code" },
    turn,
    target: "#/agent/a1",
  });
  assert.equal(n.level, "ok");
  assert.deepEqual(n.actor, { kind: "agent", id: "a1", name: "claude", cli: "claude-code" });
  assert.equal(n.status, "finished · worked for 7s");
  assert.equal(n.title, "Pushed and opened a draft PR.");
  assert.deepEqual(n.meta, [
    { text: "1 file", tone: "muted" },
    { text: "+46", tone: "add" },
    { text: "−1", tone: "del" },
  ]);
  assert.deepEqual(n.actions, [{ label: "Open", hash: "#/agent/a1", run: null, primary: false }]);
  // One live notice per agent: the key is what makes the next finish
  // replace this card instead of stacking a second one.
  assert.equal(n.key, "agent:a1");
  assert.equal(n.target, "#/agent/a1");
});

test("agentFinishNotice stays truthful when the turn said nothing", () => {
  const n = agentFinishNotice({ agent: { id: "a1" }, turn: { work: [], replies: [] }, target: "#/agent/a1" });
  assert.equal(n.title, "Finished this turn.");
  assert.equal(n.status, "finished");
  assert.deepEqual(n.meta, []);
  assert.equal(n.actor.name, "Agent");
  assert.equal(n.actor.cli, "pi");
});
