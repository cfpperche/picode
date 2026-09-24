import test from "node:test";
import assert from "node:assert/strict";
import { applyFleet, applyInbox, applyAutomations, applyRuns, applySnips, applyTui, applyUsage, touches } from "./feedReducers.js";

test("terminal events project onto the bound Pi without changing its mode", () => {
  let s = {workspaces:[],freeAgents:[{id:"a",cli:"pi",terminalId:"t",mode:"managed"}],terminals:[{id:"t",running:true,cli:"pi"}]};
  s = applyFleet(s,{type:"terminal.state",data:{termId:"t",state:"needs-you",cli:"pi"}});
  assert.equal(s.freeAgents[0].terminal.state,"needs-you");
  assert.equal(s.freeAgents[0].mode,"managed");
});

const ws = (id, agents = []) => ({ id, name: id, path: "/" + id, agents });
const ag = (id, workspaceId, extra = {}) => ({ id, workspaceId, name: id, lastStatus: "never_started", running: false, mode: "stopped", streaming: false, waiting: false, ...extra });

test("fleet: workspaces and agents are added, updated and removed in place", () => {
  let s = { workspaces: [ws("b")], freeAgents: [], terminals: [] };
  s = applyFleet(s, { type: "workspace.added", data: { id: "a", name: "a", path: "/a" } });
  assert.deepEqual(s.workspaces.map((w) => w.id), ["b", "a"], "a new workspace appends");
  s = applyFleet(s, { type: "workspace.updated", data: { id: "a", name: "Renamed", path: "/a" } });
  assert.equal(s.workspaces.find((w) => w.id === "a").name, "Renamed", "a rename patches the card");
  assert.equal(applyFleet(s, { type: "workspace.updated", data: { id: "zz", name: "x" } }), null, "an unknown workspace refetches");
  s = applyFleet(s, { type: "agent.added", data: { id: "x", workspaceId: "a", name: "x" } });
  assert.equal(s.workspaces.find((w) => w.id === "a").agents[0].mode, "stopped");
  s = applyFleet(s, { type: "agent.added", data: { id: "f", workspaceId: "ws_free", name: "f" } });
  assert.equal(s.freeAgents.length, 1);
  s = applyFleet(s, { type: "agent.updated", data: { id: "x", workspaceId: "a", name: "renamed", model: "m" } });
  assert.equal(s.workspaces.find((w) => w.id === "a").agents[0].name, "renamed");
  assert.equal(s.workspaces.find((w) => w.id === "a").agents[0].mode, "stopped", "view fields survive a patch");
  assert.equal(applyFleet(s, { type: "agent.updated", data: { id: "nope", workspaceId: "a" } }), null, "unknown agent → refetch");
  assert.equal(applyFleet(s, { type: "agent.added", data: { id: "y", workspaceId: "missing" } }), null, "unknown workspace → refetch");
  s = applyFleet(s, { type: "agent.deleted", data: { id: "x" } });
  assert.equal(s.workspaces.find((w) => w.id === "a").agents.length, 0);
  s = applyFleet(s, { type: "workspace.deleted", data: { id: "a" } });
  assert.deepEqual(s.workspaces.map((w) => w.id), ["b"]);
  assert.equal(applyFleet(s, { type: "pin.created", data: {} }), s, "unrelated events leave state untouched");
});

test("fleet: managed CLI principals bind and unbind without touching agents", () => {
  let s = { workspaces: [ws("a")], freeAgents: [], terminals: [{ id: "t1", workspaceId: "a" }] };
  s = applyFleet(s, {
    type: "managed_cli.added",
    data: { id: "m1", workspaceId: "a", cli: "claude-code", terminalId: "t1", name: "Claude" },
  });
  assert.equal(s.workspaces[0].managedClis.length, 1);
  assert.equal(s.workspaces[0].agents.length, 0);
  assert.equal(applyFleet(s, { type: "managed_cli.added", data: { id: "m2", workspaceId: "missing" } }), null);
  s = applyFleet(s, { type: "managed_cli.removed", data: { id: "m1", terminalId: "t1", workspaceId: "a" } });
  assert.equal(s.workspaces[0].managedClis.length, 0);
});

test("fleet: status and live state", () => {
  const s = { workspaces: [ws("a", [ag("x", "a", { running: true, mode: "managed", streaming: true })])], freeAgents: [], terminals: [] };
  assert.equal(applyFleet(s, { type: "agent.status", data: { id: "x", lastStatus: "running" } }), null, "a start without mode → refetch");
  const started = applyFleet(s, { type: "agent.status", data: { id: "x", lastStatus: "running", mode: "interactive", lastStatusAt: "2026-09-23T09:30:00Z" } });
  assert.equal(started.workspaces[0].agents[0].mode, "interactive");
  assert.equal(started.workspaces[0].agents[0].running, true);
  assert.equal(started.workspaces[0].agents[0].lastStatusAt, "2026-09-23T09:30:00Z", "a start stamps the row's state age");
  const restamped = applyFleet(started, { type: "agent.status", data: { id: "x", lastStatus: "running", mode: "interactive" } });
  assert.equal(restamped.workspaces[0].agents[0].lastStatusAt, "2026-09-23T09:30:00Z", "an event without a stamp keeps the row's own");
  assert.equal(applyFleet(s, { type: "agent.status", data: { id: "ghost", lastStatus: "running" } }), s, "unknown agent stays untouched");
  const stopped = applyFleet(s, { type: "agent.status", data: { id: "x", lastStatus: "stopped" } });
  assert.equal(stopped.workspaces[0].agents[0].mode, "stopped");
  assert.equal(stopped.workspaces[0].agents[0].streaming, false);
  const live = applyFleet(stopped, { type: "agent.state", data: { agentId: "x", streaming: true, waiting: false } });
  assert.equal(live.workspaces[0].agents[0].mode, "managed");
  assert.equal(live.workspaces[0].agents[0].streaming, true);
  const waiting = applyFleet(live, { type: "agent.state", data: { agentId: "x", streaming: false, waiting: true, dialog: { id: "d" } } });
  assert.equal(waiting.workspaces[0].agents[0].waiting, true);
  assert.equal(waiting.workspaces[0].agents[0].dialog.id, "d");
  assert.equal(applyFleet(s, { type: "agent.state", data: { agentId: "ghost", streaming: true } }), s);
});

test("fleet: agent.settled stamps the ready age and clears streaming", () => {
  const s = { workspaces: [ws("a", [ag("x", "a", { running: true, mode: "managed", streaming: true, lastStatusAt: "2026-09-23T09:00:00Z" })])], freeAgents: [], terminals: [] };
  const next = applyFleet(s, { type: "agent.settled", data: { id: "x", lastStatusAt: "2026-09-23T10:00:00Z" } });
  const row = next.workspaces[0].agents[0];
  assert.equal(row.streaming, false, "a settle implies not streaming");
  assert.equal(row.lastStatusAt, "2026-09-23T10:00:00Z");
  assert.equal(row.mode, "managed", "the runtime stays running");
  assert.equal(applyFleet(s, { type: "agent.settled", data: { id: "ghost", lastStatusAt: "2026-09-23T10:00:00Z" } }), s, "unknown agent stays untouched");
});


test("fleet: terminals", () => {
  let s = { workspaces: [], freeAgents: [], terminals: [] };
  s = applyFleet(s, { type: "terminal.created", data: { id: "t1", name: "T", workspaceId: "ws_free" } });
  assert.equal(s.terminals.length, 1);
  s = applyFleet(s, { type: "terminal.updated", data: { id: "t1", name: "U", workspaceId: "ws_free" } });
  assert.equal(s.terminals[0].name, "U");
  assert.equal(applyFleet(s, { type: "terminal.updated", data: { id: "zz" } }), null);
  s = applyFleet(s, { type: "terminal.deleted", data: { id: "t1" } });
  assert.equal(s.terminals.length, 0);
});

test("fleet: reorder applies a permutation and refetches on a mismatch", () => {
  let s = {
    workspaces: [ws("b", [ag("b1", "b"), ag("b2", "b")]), ws("a", [ag("a1", "a")])],
    freeAgents: [ag("f1", "ws_free"), ag("f2", "ws_free")],
    terminals: [
      { id: "t1", workspaceId: "a" },
      { id: "t2", workspaceId: "ws_free" },
      { id: "t3", workspaceId: "a" },
    ],
  };
  s = applyFleet(s, { type: "workspace.reordered", data: { ids: ["a", "b"] } });
  assert.deepEqual(s.workspaces.map((w) => w.id), ["a", "b"]);
  assert.equal(applyFleet(s, { type: "workspace.reordered", data: { ids: ["a"] } }), null);
  s = applyFleet(s, { type: "agent.reordered", data: { workspaceId: "b", ids: ["b2", "b1"] } });
  assert.deepEqual(s.workspaces.find((w) => w.id === "b").agents.map((a) => a.id), ["b2", "b1"]);
  s = applyFleet(s, { type: "agent.reordered", data: { workspaceId: "ws_free", ids: ["f2", "f1"] } });
  assert.deepEqual(s.freeAgents.map((a) => a.id), ["f2", "f1"]);
  s = applyFleet(s, { type: "terminal.reordered", data: { workspaceId: "a", ids: ["t3", "t1"] } });
  assert.deepEqual(s.terminals.map((t) => t.id), ["t3", "t2", "t1"]);
  assert.equal(applyFleet(s, { type: "terminal.reordered", data: { ids: ["missing"] } }), null);
});

test("fleet: terminal.state (CLI, ADR-0056 tier 1)", () => {
  let s = { workspaces: [], freeAgents: [], terminals: [{ id: "t1", name: "T" }] };
  s = applyFleet(s, { type: "terminal.state", data: { termId: "t1", state: "working", cli: "claude-code", at: "2026-09-04T10:00:00Z" } });
  assert.equal(s.terminals[0].state, "working");
  assert.equal(s.terminals[0].cli, "claude-code");
  assert.equal(s.terminals[0].stateAt, "2026-09-04T10:00:00Z");
  s = applyFleet(s, { type: "terminal.state", data: { termId: "t1", state: "needs-you", cli: "claude-code" } });
  assert.equal(s.terminals[0].state, "needs-you");
  // Clearing (stale sweep) removes the fields instead of keeping a lie.
  s = applyFleet(s, { type: "terminal.state", data: { termId: "t1", state: null } });
  assert.equal(s.terminals[0].state, undefined);
  assert.equal(s.terminals[0].cli, undefined);
  // Unknown terminals stay untouched — durable events reconcile the list.
  assert.equal(applyFleet(s, { type: "terminal.state", data: { termId: "zz", state: "working" } }), s);
  assert.equal(applyFleet(s, { type: "terminal.state", data: {} }), s);
});

test("fleet: terminal.checklist (the plan of the pi inside the terminal, ADR-0055)", () => {
  let s = { workspaces: [], freeAgents: [], terminals: [{ id: "t1", name: "T" }] };
  s = applyFleet(s, { type: "terminal.checklist", data: { termId: "t1", items: [{ text: "explore", status: "completed" }, { text: "edit", status: "in-progress" }], updatedAt: "2026-09-06T10:00:00Z" } });
  assert.deepEqual(s.terminals[0].checklist, { items: [{ text: "explore", status: "completed" }, { text: "edit", status: "in-progress" }], absent: false, updatedAt: "2026-09-06T10:00:00Z" });
  // An absent marker is a row of its own (renders as silence, ADR-0092).
  s = applyFleet(s, { type: "terminal.checklist", data: { termId: "t1", items: [], absent: true } });
  assert.deepEqual(s.terminals[0].checklist, { items: [], absent: true, updatedAt: undefined });
  // The reset/cleared empty state is silence: the line goes away.
  s = applyFleet(s, { type: "terminal.checklist", data: { termId: "t1", items: [] } });
  assert.equal(s.terminals[0].checklist, undefined);
  // Unknown terminals and shapeless events stay untouched.
  assert.equal(applyFleet(s, { type: "terminal.checklist", data: { termId: "zz", items: [{ text: "x" }] } }), s);
  assert.equal(applyFleet(s, { type: "terminal.checklist", data: {} }), s);
});

test("fleet: terminal lifecycle clears stale presence; launch defaults invalidate", () => {
  const state = { workspaces: [], freeAgents: [], terminals: [{ id: "t", running: true, cli: "pi", tui: { runId: "old" }, state: "working" }] };
  const stopped = { id: "t", running: false, launchCli: "pi" };
  const next = applyFleet(state, { type: "terminal.changed", data: stopped });
  assert.deepEqual(next.terminals, [stopped]);
  assert.equal(applyFleet(next, { type: "terminal.launch", data: { id: "t" } }), null);
  assert.equal(applyFleet(next, { type: "cli.updated", data: { id: "pi" } }), null);
});

test("fleet: a terminal without an adapter keeps its launch identity", () => {
  // The store's terminal.created carries the bare row; the CLI a Muse Code or
  // Antigravity row draws arrives in terminal.changed. Either frame can land
  // first, and the row must never end up wearing the bare one.
  const bare = { id: "t", name: "Muse Code", cwd: "/w", workspaceId: "ws", createdAt: "now" };
  const live = { ...bare, session: "picode-sh-t", running: true, launchCli: "muse", launchPending: false };
  const seeded = applyFleet({ workspaces: [], freeAgents: [], terminals: [] }, { type: "terminal.changed", data: live });
  assert.deepEqual(seeded.terminals, [live]);
  assert.deepEqual(applyFleet(seeded, { type: "terminal.created", data: bare }).terminals, [live]);
  const inserted = applyFleet({ workspaces: [], freeAgents: [], terminals: [] }, { type: "terminal.created", data: bare });
  assert.deepEqual(inserted.terminals, [bare]);
  assert.deepEqual(applyFleet(inserted, { type: "terminal.changed", data: live }).terminals, [live]);
});

test("fleet: terminal.runtime keeps run identities and rejects stale ends", () => {
  let s = { workspaces: [], freeAgents: [], terminals: [{ id: "t1", name: "T" }] };
  s = applyFleet(s, { type: "terminal.runtime", data: { termId: "t1", action: "started", cli: "pi", source: "wrapper", runId: "new", startedAt: "2026-09-04T10:00:00Z" } });
  assert.deepEqual(s.terminals[0].tui, { cli: "pi", source: "wrapper", runId: "new", startedAt: "2026-09-04T10:00:00Z" });
  s = applyFleet(s, { type: "terminal.state", data: { termId: "t1", state: "working", cli: "pi", runId: "new" } });
  assert.equal(s.terminals[0].state, "working");
  // A reconnect can deliver the new start without its paired clear event;
  // starting a new wrapper run must still drop the old activity locally.
  s = applyFleet(s, { type: "terminal.runtime", data: { termId: "t1", action: "started", cli: "pi", source: "wrapper", runId: "newer", startedAt: "2026-09-04T10:01:00Z" } });
  assert.equal(s.terminals[0].state, undefined);
  assert.equal(applyFleet(s, { type: "terminal.runtime", data: { termId: "t1", action: "started", cli: "pi", source: "wrapper", runId: "older", startedAt: "2026-09-04T10:00:30Z" } }), s);
  const stale = applyFleet(s, { type: "terminal.runtime", data: { termId: "t1", action: "ended", runId: "old" } });
  assert.equal(stale, s);
  s = applyFleet(s, { type: "terminal.runtime", data: { termId: "t1", action: "ended", runId: "newer" } });
  assert.equal(s.terminals[0].tui, undefined);
  assert.equal(s.terminals[0].state, undefined);
  assert.equal(s.terminals[0].cli, undefined);
  // Once the current lease is gone, a delayed state/end event still cannot
  // resurrect or clear a later legacy projection.
  const quiet = applyFleet(s, { type: "terminal.state", data: { termId: "t1", state: "working", cli: "pi", runId: "newer" } });
  assert.equal(quiet, s);
  assert.equal(applyFleet(s, { type: "terminal.runtime", data: { termId: "t1", action: "ended", runId: "old" } }), s);
});

test("fleet: terminal.last_session pins the conversation the menus read", () => {
  const s = { workspaces: [], freeAgents: [], terminals: [{ id: "t1", name: "communication" }] };
  const next = applyFleet(s, {
    type: "terminal.last_session",
    data: { termId: "t1", cli: "pi", sessionId: "s1", path: "/p", cwd: "/w", name: "Race" },
  });
  assert.deepEqual(next.terminals[0].lastSession, {
    cli: "pi", sessionId: "s1", path: "/p", cwd: "/w", name: "Race",
    updatedAt: "", preview: "", resumeArgs: [],
  });
  assert.equal(applyFleet(s, { type: "terminal.last_session", data: { termId: "gone", sessionId: "s1" } }), s);
  assert.equal(applyFleet(s, { type: "terminal.last_session", data: { termId: "t1" } }), s);
});

test("inbox reducer", () => {
  let l = [{ id: "a", state: "unread" }];
  l = applyInbox(l, { type: "inbox.created", data: { id: "b", state: "unread" } });
  assert.deepEqual(l.map((i) => i.id), ["b", "a"]);
  l = applyInbox(l, { type: "inbox.updated", data: { id: "a", state: "done" } });
  assert.equal(l[1].state, "done");
  l = applyInbox(l, { type: "inbox.cleared", data: { count: 1 } });
  assert.deepEqual(l.map((i) => i.id), ["b"]);
  l = applyInbox(l, { type: "inbox.deleted", data: { id: "b" } });
  assert.equal(l.length, 0);
  assert.equal(applyInbox(l, { type: "inbox.created", data: {} }), null);
});

test("automations and runs reducers", () => {
  let l = [{ id: "x", name: "X", running: false, lastRun: null, sparkline: [1] }];
  assert.equal(applyAutomations(l, { type: "automation.created", data: { id: "y" } }), null);
  l = applyAutomations(l, { type: "automation.updated", data: { id: "x", name: "X2", enabled: false } });
  assert.equal(l[0].name, "X2");
  assert.deepEqual(l[0].sparkline, [1], "view extras survive");
  l = applyAutomations(l, { type: "run.created", data: { id: "r1", automationId: "x", status: "running" } });
  assert.equal(l[0].running, true);
  l = applyAutomations(l, { type: "run.finished", data: { id: "r1", automationId: "x", status: "done" } });
  assert.equal(l[0].running, false);
  assert.equal(l[0].lastRun.status, "done");
  l = applyAutomations(l, { type: "automation.deleted", data: { id: "x" } });
  assert.equal(l.length, 0);

  let runs = [];
  runs = applyRuns(runs, "x", { type: "run.created", data: { id: "r1", automationId: "x", status: "running" } });
  runs = applyRuns(runs, "x", { type: "run.finished", data: { id: "r1", automationId: "x", status: "done" } });
  assert.equal(runs.length, 1);
  assert.equal(runs[0].status, "done");
  assert.equal(applyRuns(runs, "x", { type: "run.created", data: { id: "r9", automationId: "other" } }), runs);
  assert.equal(touches({ type: "inbox.created" }, ["inbox", "apps"]), true);
  assert.equal(touches({ type: "run.created" }, ["inbox"]), false);
});

test("tui and usage reducers", () => {
  let ids = [];
  ids = applyTui(ids, { type: "agent.tui", data: { agentId: "a", working: true } });
  ids = applyTui(ids, { type: "agent.tui", data: { agentId: "a", working: true } });
  assert.deepEqual(ids, ["a"]);
  ids = applyTui(ids, { type: "agent.tui", data: { agentId: "a", working: false } });
  assert.deepEqual(ids, []);
  assert.equal(applyUsage(null, { cost: 1 }), null);
  const bar = applyUsage({ cost: 0.5, input: 10, cacheRead: 30, contextWindow: 1000, contextPercent: 5 }, { cost: 0.25, input: 20, output: 5, cacheRead: 60, cacheWrite: 1, totalTokens: 400 });
  assert.equal(bar.cost, 0.75);
  assert.equal(bar.input, 30);
  assert.equal(bar.output, 5);
  assert.equal(bar.cacheRead, 90);
  assert.equal(bar.cacheHit, 75);
  assert.equal(bar.contextTokens, 400);
  assert.equal(bar.contextPercent, 40);
  const noCtx = applyUsage({ cost: 0 }, { cost: 0.1, input: 1 });
  assert.equal(noCtx.contextTokens, undefined);
});

test("fleet: git.updated patches the pills in place", () => {
  const state = {
    workspaces: [{ id: "w1", path: "/repo", git: { branch: "main", dirty: 2 }, agents: [{ id: "a1", git: { branch: "main", dirty: 2 } }, { id: "a2" }] }],
    freeAgents: [{ id: "f1", workPath: "/repo", git: { branch: "main", dirty: 2 } }],
    terminals: [],
  };
  // Branch flip + dirty cleared: workspace, its agent and the free agent
  // on the same path all move together.
  const next = applyFleet(state, { type: "git.updated", data: { path: "/repo", branch: "feat", workspaceIds: ["w1"], agentIds: ["a1", "f1"] } });
  assert.deepEqual(next.workspaces[0].git, { branch: "feat", dirty: 0, worktree: undefined });
  assert.deepEqual(next.workspaces[0].agents[0].git, { branch: "feat", dirty: 0, worktree: undefined });
  assert.equal(next.workspaces[0].agents[1].git, undefined); // untouched agent keeps its shape
  assert.deepEqual(next.freeAgents[0].git, { branch: "feat", dirty: 0, worktree: undefined });
  // A dirty count arrives only when non-zero (omitempty on the server).
  const dirty = applyFleet(state, { type: "git.updated", data: { path: "/repo", branch: "main", dirty: 5, workspaceIds: ["w1"] } });
  assert.equal(dirty.workspaces[0].git.dirty, 5);
  // Not a repo any more (no branch in the event): the pills clear.
  const cleared = applyFleet(state, { type: "git.updated", data: { path: "/repo", workspaceIds: ["w1"] } });
  assert.equal(cleared.workspaces[0].git, null);
  // Unknown path: untouched, not a refetch signal.
  const miss = applyFleet(state, { type: "git.updated", data: { path: "/elsewhere", branch: "x" } });
  assert.equal(miss, state);
  // A linked worktree's path-only event (branch + dirty, no ids) is a miss
  // too: pills describe the anchor folder, never the sibling checkout.
  const sibling = applyFleet(state, { type: "git.updated", data: { path: "/repo/.worktrees/side", branch: "side", dirty: 3, worktree: "side" } });
  assert.equal(sibling, state);
});

test("snips: create, patch, delete; unknown update refetches", () => {
  let list = [];
  list = applySnips(list, { type: "snip.created", data: { id: "s1", title: "A", slug: "a" } });
  assert.equal(list[0].id, "s1");
  list = applySnips(list, { type: "snip.updated", data: { id: "s1", title: "B", slug: "a" } });
  assert.equal(list[0].title, "B");
  assert.equal(applySnips(list, { type: "snip.updated", data: { id: "ghost", title: "x" } }), null);
  list = applySnips(list, { type: "snip.deleted", data: { id: "s1" } });
  assert.equal(list.length, 0);
  assert.equal(applySnips(list, { type: "pin.created", data: { id: "p" } }), list);
  assert.equal(touches({ type: "snip.created" }, ["snip"]), true);
});

test("a sign-in terminal never joins the terminal list (ADR-0184)", () => {
  const state = { workspaces: [], freeAgents: [], terminals: [] };
  assert.equal(applyFleet(state, { type: "terminal.created", data: { id: "s1", kind: "signin" } }), state);
  assert.equal(applyFleet(state, { type: "terminal.changed", data: { id: "s1", kind: "signin", running: true } }), state);
  assert.equal(applyFleet(state, { type: "terminal.created", data: { id: "t1" } }).terminals.length, 1);
});
