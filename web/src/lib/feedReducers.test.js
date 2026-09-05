import test from "node:test";
import assert from "node:assert/strict";
import { applyFleet, applyInbox, applyAutomations, applyRuns, applyTui, applyUsage, touches } from "./feedReducers.js";

const ws = (id, agents = []) => ({ id, name: id, path: "/" + id, agents });
const ag = (id, workspaceId, extra = {}) => ({ id, workspaceId, name: id, lastStatus: "never_started", running: false, mode: "stopped", streaming: false, waiting: false, ...extra });

test("fleet: workspaces and agents are added, updated and removed in place", () => {
  let s = { workspaces: [ws("b")], freeAgents: [], terminals: [] };
  s = applyFleet(s, { type: "workspace.added", data: { id: "a", name: "a", path: "/a" } });
  assert.deepEqual(s.workspaces.map((w) => w.id), ["a", "b"]);
  s = applyFleet(s, { type: "agent.added", data: { id: "x", workspaceId: "a", name: "x" } });
  assert.equal(s.workspaces[0].agents[0].mode, "stopped");
  s = applyFleet(s, { type: "agent.added", data: { id: "f", workspaceId: "ws_free", name: "f" } });
  assert.equal(s.freeAgents.length, 1);
  s = applyFleet(s, { type: "agent.updated", data: { id: "x", workspaceId: "a", name: "renamed", model: "m" } });
  assert.equal(s.workspaces[0].agents[0].name, "renamed");
  assert.equal(s.workspaces[0].agents[0].mode, "stopped", "view fields survive a patch");
  assert.equal(applyFleet(s, { type: "agent.updated", data: { id: "nope", workspaceId: "a" } }), null, "unknown agent → refetch");
  assert.equal(applyFleet(s, { type: "agent.added", data: { id: "y", workspaceId: "missing" } }), null, "unknown workspace → refetch");
  s = applyFleet(s, { type: "agent.deleted", data: { id: "x" } });
  assert.equal(s.workspaces[0].agents.length, 0);
  s = applyFleet(s, { type: "workspace.deleted", data: { id: "a" } });
  assert.deepEqual(s.workspaces.map((w) => w.id), ["b"]);
  assert.equal(applyFleet(s, { type: "pin.created", data: {} }), s, "unrelated events leave state untouched");
});

test("fleet: status and live state", () => {
  const s = { workspaces: [ws("a", [ag("x", "a", { running: true, mode: "managed", streaming: true })])], freeAgents: [], terminals: [] };
  assert.equal(applyFleet(s, { type: "agent.status", data: { id: "x", lastStatus: "running" } }), null, "a start without mode → refetch");
  const started = applyFleet(s, { type: "agent.status", data: { id: "x", lastStatus: "running", mode: "interactive" } });
  assert.equal(started.workspaces[0].agents[0].mode, "interactive");
  assert.equal(started.workspaces[0].agents[0].running, true);
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

test("fleet: terminal.state (guest CLI, ADR-0056 tier 1)", () => {
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

test("fleet: terminal lifecycle clears stale presence; launch defaults invalidate", () => {
  const state = { workspaces: [], freeAgents: [], terminals: [{ id: "t", running: true, cli: "pi", tui: { runId: "old" }, state: "working" }] };
  const stopped = { id: "t", running: false, launchCli: "pi" };
  const next = applyFleet(state, { type: "terminal.changed", data: stopped });
  assert.deepEqual(next.terminals, [stopped]);
  assert.equal(applyFleet(next, { type: "terminal.launch", data: { id: "t" } }), null);
  assert.equal(applyFleet(next, { type: "cli.updated", data: { id: "pi" } }), null);
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
});
