import assert from "node:assert/strict";
import { test } from "node:test";
import { groupTurns } from "./turns.js";
import {
  agentFinishNotice, askNoticeKey, asksOnSurface, changeMeta, changeTotals, finishStatus, lastReplyLine, needsYouNotice, needsYouPlan, normalizeNotice, noticeDuration, noticeMuted, plainNotice, reminderNotice, reminderPlan, remindersCollapsedNotice, suppressNotice,
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

test("suppressNotice: never announce the focused surface, always announce an error", () => {
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
  // A needs-you for the conversation already showing its ask card IS
  // redundant, so warn is suppressed like the rest.
  const warn = normalizeNotice({ level: "warn", title: "Needs a choice", target: "#/agent/a1" });
  assert.equal(suppressNotice(warn, { target: "#/agent/a1", focused: true }), true);
});

test("noticeMuted: the announce switches silence a class, not the app", () => {
  const finished = agentFinishNotice({ agent: { id: "a1" }, turn: null, target: "#/agent/a1" });
  const ask = needsYouNotice({ agentId: "a1", agentName: "atlas", title: "Q?" }, "#/agent/a1");
  assert.equal(finished.channel, "finished");
  assert.equal(ask.channel, "needsYou");
  assert.equal(noticeMuted(finished, { announceFinished: false }), true);
  assert.equal(noticeMuted(finished, { announceFinished: true }), false);
  assert.equal(noticeMuted(ask, { announceNeedsYou: false }), true);
  assert.equal(noticeMuted(finished, { announceNeedsYou: false }), false);
  // Feedback for something the user just did carries no channel.
  assert.equal(noticeMuted(plainNotice("Saved.", "ok"), { announceFinished: false, announceNeedsYou: false }), false);
  assert.equal(noticeMuted(finished, {}), false);
});

test("needsYouNotice carries the question and outlives the clock", () => {
  const n = needsYouNotice(
    { agentId: "a1", agentName: "atlas", where: "picode", title: "Run the migration?", message: "drops a table", dialogId: "d1", key: "ask:a1" },
    "#/agent/a1",
  );
  assert.equal(n.level, "warn");
  assert.equal(n.status, "needs you · picode");
  assert.equal(n.title, "Run the migration?");
  assert.equal(n.body, "drops a table");
  assert.deepEqual(n.actions, [{ label: "Answer", hash: "#/agent/a1", run: null, primary: true }]);
  assert.equal(n.duration, Infinity);
  assert.equal(noticeDuration(n, 4000), Infinity);
  // Keyed by the question, so the next one replaces this card.
  assert.equal(n.key, "ask:a1:d1");
  assert.equal(askNoticeKey({ agentId: "a1" }), "ask:a1");
  assert.equal(askNoticeKey({}), "");
});

test("needsYouPlan announces arrivals, withdraws answers, and seeds silently", () => {
  const target = (id) => "#/agent/" + id;
  const q1 = { kind: "ask", agentId: "a1", agentName: "atlas", title: "Q1", dialogId: "d1" };
  const q2 = { kind: "ask", agentId: "a1", agentName: "atlas", title: "Q2", dialogId: "d2" };
  const other = { kind: "inbox", itemId: "i1", title: "not a dialog" };

  // A page load must not toast a backlog the badge already shows.
  const seeded = needsYouPlan([q1, other], null, target);
  assert.deepEqual(seeded.fresh, []);
  assert.deepEqual(seeded.gone, []);
  assert.deepEqual([...seeded.keys], ["ask:a1:d1"]);

  // Nothing changed.
  const idle = needsYouPlan([q1], seeded.keys, target);
  assert.deepEqual(idle.fresh, []);
  assert.deepEqual(idle.gone, []);

  // Answered, and the next question raised: one card out, one in.
  const rolled = needsYouPlan([q2], seeded.keys, target);
  assert.deepEqual(rolled.gone, ["ask:a1:d1"]);
  assert.equal(rolled.fresh.length, 1);
  assert.equal(rolled.fresh[0].title, "Q2");
  assert.equal(rolled.fresh[0].actions[0].hash, "#/agent/a1");

  // Answered and nothing follows.
  const cleared = needsYouPlan([], rolled.keys, target);
  assert.deepEqual(cleared.fresh, []);
  assert.deepEqual(cleared.gone, ["ask:a1:d2"]);
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

test("asksOnSurface names the cards the user is now looking at", () => {
  const target = (id) => "#/agent/" + id;
  const entries = [
    { kind: "ask", agentId: "a1", dialogId: "d1" },
    { kind: "ask", agentId: "a2" },
    { kind: "inbox", itemId: "i1" },
  ];
  assert.deepEqual(asksOnSurface(entries, "#/agent/a1", target), ["ask:a1:d1"]);
  assert.deepEqual(asksOnSurface(entries, "#/agent/a2", target), ["ask:a2"]);
  assert.deepEqual(asksOnSurface(entries, "#/settings", target), []);
  assert.deepEqual(asksOnSurface(entries, "", target), []);
});

test("reminder cards are sticky, keyed by the Inbox item, and close through the item", () => {
  let closed = 0;
  let snoozed = 0;
  const n = reminderNotice({ inboxId: "i1", pinId: "p1", title: "Water the plants", label: "every day at 09:00", body: "Balcony first.", catchUp: true }, {
    pinHash: "#/pins/p1", snooze: () => { snoozed++; }, close: () => { closed++; },
  });
  assert.equal(n.key, "reminder:i1");
  assert.equal(n.duration, Infinity);
  assert.equal(n.channel, "reminder");
  assert.equal(n.actor.kind, "pin");
  assert.equal(n.actor.name, "Water the plants");
  assert.equal(n.status, "reminder · every day at 09:00 · was due earlier");
  assert.equal(n.title, "Balcony first.");
  assert.deepEqual(n.actions.map((a) => a.label), ["Snooze", "Open"]);
  assert.equal(n.actions[1].hash, "#/pins/p1");
  n.actions[0].run(); n.onClose();
  assert.deepEqual([snoozed, closed], [1, 1]);
  assert.equal(noticeDuration(n, 4000), Infinity);
  assert.equal(suppressNotice(n, { target: "#/pins/p1", focused: true }), false, "never suppressed by the surface");
  assert.equal(noticeMuted(n, { announceReminders: false }), true);
  assert.equal(noticeMuted(n, {}), false);
  assert.equal(reminderNotice({ inboxId: "i2", title: "T" }).title, "T");
  assert.equal(reminderNotice({ inboxId: "i2", title: "T" }).actions.length, 0);
});

test("reminderPlan raises every open item on the first pass and collapses above the cap", () => {
  const notice = (f) => reminderNotice(f, { pinHash: "#/pins/" + f.pinId });
  const collapsed = (n) => remindersCollapsedNotice(n, "#/app/inbox");
  const items = [
    { id: "i1", sourceId: "p1", title: "A", reason: "at Tue 09:00", body: "first line\nsecond", state: "unread" },
    { id: "i2", sourceId: "p2", title: "B", reason: "every day at 09:00", body: "x\n\nWas due Mon 09:00.", state: "read" },
    { id: "i3", sourceId: "p3", title: "C", state: "done" },
  ];
  const first = reminderPlan(items, null, { notice, collapsed });
  assert.deepEqual([...first.keys], ["reminder:i1", "reminder:i2"]);
  assert.deepEqual(first.show.map((n) => n.title), ["first line", "x"]);
  assert.equal(first.show[1].status.includes("was due earlier"), true, "catch-up read from the body");
  assert.deepEqual(first.hide, []);
  // i1 closed elsewhere: its card goes; i2 stays and is re-raised (same key replaces).
  const second = reminderPlan(items.slice(1), first.keys, { notice, collapsed });
  assert.deepEqual(second.hide, ["reminder:i1"]);
  assert.deepEqual([...second.keys], ["reminder:i2"]);
  // Above the cap: one card, and every per-item card is withdrawn.
  const many = ["a", "b", "c", "d"].map((id) => ({ id, sourceId: "p" + id, title: id, state: "unread" }));
  const third = reminderPlan(many, second.keys, { notice, collapsed, cap: 3 });
  assert.deepEqual([...third.keys], ["reminders:all"]);
  assert.deepEqual(third.hide, ["reminder:i2"]);
  assert.equal(third.show.length, 1);
  assert.equal(third.show[0].title, "4 reminders are waiting for you.");
  assert.equal(third.show[0].actions[0].hash, "#/app/inbox");
  // Back under the cap: the collapsed card goes, per-item cards return.
  const fourth = reminderPlan(many.slice(0, 2), third.keys, { notice, collapsed, cap: 3 });
  assert.deepEqual(fourth.hide, ["reminders:all"]);
  assert.equal(fourth.show.length, 2);
  // Nothing open: everything shown is withdrawn.
  const fifth = reminderPlan([], fourth.keys, { notice, collapsed });
  assert.deepEqual(fifth.hide.sort(), ["reminder:a", "reminder:b"]);
});
