// Pure reducers that apply change-feed events (ADR-0048) to the shapes the
// shells hold. Every reducer returns the next state, or null when the event
// cannot be applied faithfully — the caller then refetches. Unknown types
// return the same state (untouched), never null.

const FREE = "ws_free";

function entity(type) {
  const i = type.indexOf(".");
  return i < 0 ? type : type.slice(0, i);
}

function agentView(a, prev) {
  return { running: false, mode: "stopped", streaming: false, waiting: false, ...(prev || {}), ...a };
}

function byName(list) {
  return [...list].sort((x, y) => String(x.name || "").localeCompare(String(y.name || "")));
}

function compareBase36(a, b) {
  if (!/^[0-9a-z]+$/.test(a) || !/^[0-9a-z]+$/.test(b)) return null;
  if (a.length !== b.length) return a.length > b.length ? 1 : -1;
  return a === b ? 0 : a > b ? 1 : -1;
}

// Burst generations are "<unix-ns-base36>-<process-sequence-base36>". An
// HTTP fleet reconciliation can overtake an already-buffered SSE notice, so
// the browser must reject an older generation too—not only the coordinator.
function compareBurstGeneration(a, b) {
  const left = String(a || "").split("-");
  const right = String(b || "").split("-");
  if (left.length !== 2 || right.length !== 2) return null;
  const stamp = compareBase36(left[0], right[0]);
  if (stamp == null || stamp !== 0) return stamp;
  return compareBase36(left[1], right[1]);
}

// applyFleet({workspaces, freeAgents, terminals}, ev) -> next | null | same
export function applyFleet(state, ev) {
  const { workspaces = [], freeAgents = [], terminals = [] } = state || {};
  const d = ev && ev.data ? ev.data : {};
  switch (ev.type) {
    case "workspace.added": {
      if (!d.id || workspaces.some((w) => w.id === d.id)) return state;
      return { ...state, workspaces: byName([...workspaces, { ...d, agents: [] }]) };
    }
    case "workspace.deleted":
      return {
        ...state,
        workspaces: workspaces.filter((w) => w.id !== d.id),
        terminals: terminals.filter((t) => t.workspaceId !== d.id),
      };
    case "agent.added": {
      if (!d.id) return state;
      if (d.workspaceId === FREE) {
        if (freeAgents.some((a) => a.id === d.id)) return state;
        return { ...state, freeAgents: [...freeAgents, agentView(d)] };
      }
      const ws = workspaces.find((w) => w.id === d.workspaceId);
      if (!ws) return null;
      if ((ws.agents || []).some((a) => a.id === d.id)) return state;
      return { ...state, workspaces: workspaces.map((w) => (w.id === ws.id ? { ...w, agents: [...(w.agents || []), agentView(d)] } : w)) };
    }
    case "agent.updated": {
      let found = false;
      const patch = (a) => { if (a.id !== d.id) return a; found = true; return agentView(d, a); };
      const next = {
        ...state,
        workspaces: workspaces.map((w) => ({ ...w, agents: (w.agents || []).map(patch) })),
        freeAgents: freeAgents.map(patch),
      };
      if (!found) return null;
      // Moved between workspaces: the lists no longer match the server.
      const home = d.workspaceId === FREE ? null : workspaces.find((w) => w.id === d.workspaceId);
      const inFree = freeAgents.some((a) => a.id === d.id);
      if ((d.workspaceId === FREE && !inFree) || (home && !(home.agents || []).some((a) => a.id === d.id))) return null;
      return next;
    }
    case "agent.deleted":
      return {
        ...state,
        workspaces: workspaces.map((w) => ({ ...w, agents: (w.agents || []).filter((a) => a.id !== d.id) })),
        freeAgents: freeAgents.filter((a) => a.id !== d.id),
      };
    case "agent.burst": {
      if (!d.agentId) return state;
      let found = false;
      const patch = (a) => {
        if (a.id !== d.agentId) return a;
        found = true;
        if (a.burst && a.burst.generation && d.generation && a.burst.generation !== d.generation) {
          const order = compareBurstGeneration(d.generation, a.burst.generation);
          if (order == null || order <= 0) return a;
        }
        if (d.phase === "idle") {
          const rest = { ...a };
          delete rest.burst;
          if (d.terminalUnavailable) {
            return { ...rest, running: false, mode: "stopped", streaming: false, waiting: false, dialog: undefined };
          }
          return rest;
        }
        // The control process is an implementation detail. Preserve the
        // logical TUI mode even while agent.state streams underneath it.
        return { ...a, running: true, mode: "interactive", burst: d };
      };
      const next = { ...state, workspaces: workspaces.map((w) => ({ ...w, agents: (w.agents || []).map(patch) })), freeAgents: freeAgents.map(patch) };
      return found ? next : state;
    }
    case "agent.status": {
      // A start carries the mode the server started it in (managed |
      // interactive); a start without one cannot be applied faithfully.
      const running = d.lastStatus === "running";
      const mode = running ? d.mode : "stopped";
      if (running && mode !== "managed" && mode !== "interactive") {
        return workspaces.some((w) => (w.agents || []).some((a) => a.id === d.id)) || freeAgents.some((a) => a.id === d.id) ? null : state;
      }
      let found = false;
      const patch = (a) => {
        if (a.id !== d.id) return a;
        found = true;
        return running
          ? { ...a, running: true, mode, lastStatus: d.lastStatus }
          : { ...a, running: false, mode: "stopped", streaming: false, waiting: false, dialog: undefined, lastStatus: d.lastStatus };
      };
      const next = { ...state, workspaces: workspaces.map((w) => ({ ...w, agents: (w.agents || []).map(patch) })), freeAgents: freeAgents.map(patch) };
      return found ? next : state;
    }
    case "agent.state": {
      let found = false;
      const patch = (a) => {
        if (a.id !== d.agentId) return a;
        found = true;
        return { ...a, running: true, mode: a.mode === "stopped" ? "managed" : a.mode, streaming: !!d.streaming, waiting: !!d.waiting, dialog: d.dialog || undefined };
      };
      const next = { ...state, workspaces: workspaces.map((w) => ({ ...w, agents: (w.agents || []).map(patch) })), freeAgents: freeAgents.map(patch) };
      return found ? next : state;
    }
    case "terminal.created":
      if (!d.id || terminals.some((t) => t.id === d.id)) return state;
      return { ...state, terminals: [...terminals, d] };
    case "terminal.updated":
      return terminals.some((t) => t.id === d.id) ? { ...state, terminals: terminals.map((t) => (t.id === d.id ? { ...t, ...d } : t)) } : null;
    case "terminal.deleted":
      return { ...state, terminals: terminals.filter((t) => t.id !== d.id) };
    case "terminal.state": {
      // Ephemeral (id 0): a terminal CLI hook reported lifecycle state
      // (ADR-0056 tier 1) — same words as agent.state. Unknown terminals
      // stay untouched; the durable terminal.updated events carry the
      // state field for reconciliation after a refetch.
      if (!d.termId) return state;
      if (!terminals.some((t) => t.id === d.termId)) return state;
      return {
        ...state,
        terminals: terminals.map((t) =>
          t.id === d.termId
            ? d.state
              ? { ...t, state: d.state, cli: d.cli || undefined }
              : { ...t, state: undefined, cli: undefined }
            : t,
        ),
      };
    }
    case "git.updated": {
      // Ephemeral (id 0): one fleet-wide watcher Inspect per directory,
      // fanned out here. The git shape mirrors gitinfo.Info; a missing
      // branch (not a repo / gone directory) clears the pills. Unknown
      // paths return the state untouched — the durable events reconcile
      // the lists themselves.
      if (!d.path) return state;
      const wIDs = d.workspaceIds || [];
      const aIDs = d.agentIds || [];
      const git = d.branch || d.dirty ? { branch: d.branch || "", dirty: d.dirty || 0, worktree: d.worktree || undefined } : null;
      let found = false;
      const wsNext = workspaces.map((w) => {
        let changed = false;
        let ws = w;
        if (wIDs.includes(w.id) || w.path === d.path) {
          ws = { ...w, git };
          changed = true;
          found = true;
        }
        const agents = (ws.agents || []).map((a) => {
          if (!aIDs.includes(a.id)) return a;
          found = true;
          changed = true;
          return { ...a, git };
        });
        if (!changed) return w;
        return { ...ws, agents };
      });
      const freeNext = freeAgents.map((a) => {
        if (!aIDs.includes(a.id)) return a;
        found = true;
        return { ...a, git };
      });
      return found ? { ...state, workspaces: wsNext, freeAgents: freeNext } : state;
    }
    default:
      return state;
  }
}

// applyTui(ids, ev) -> the working-id list after an agent.tui event.
export function applyTui(ids, ev) {
  const list = ids || [];
  const d = ev && ev.data ? ev.data : {};
  if (ev.type !== "agent.tui" || !d.agentId) return list;
  const has = list.includes(d.agentId);
  if (d.working && !has) return [...list, d.agentId];
  if (!d.working && has) return list.filter((id) => id !== d.agentId);
  return list;
}

// applyUsage(bar, u) -> the status bar after one assistant message's
// usage, the same arithmetic internal/session's scanUsage does over the
// file: sums for tokens and cost, last totalTokens as the context size,
// this message's cache hit. null bar (never fetched) stays null.
export function applyUsage(bar, u) {
  if (!bar || !u) return bar;
  const next = { ...bar };
  next.cost = (bar.cost || 0) + (u.cost || 0);
  next.input = (bar.input || 0) + (u.input || 0);
  next.output = (bar.output || 0) + (u.output || 0);
  next.cacheRead = (bar.cacheRead || 0) + (u.cacheRead || 0);
  next.cacheWrite = (bar.cacheWrite || 0) + (u.cacheWrite || 0);
  const denom = (u.input || 0) + (u.cacheRead || 0);
  if (denom > 0) next.cacheHit = (100 * (u.cacheRead || 0)) / denom;
  if (u.totalTokens > 0) {
    next.contextTokens = u.totalTokens;
    if (bar.contextWindow > 0) next.contextPercent = (100 * u.totalTokens) / bar.contextWindow;
  }
  return next;
}

// applyInbox(items, ev) -> next | null | same. items newest first.
export function applyInbox(items, ev) {
  const list = items || [];
  const d = ev && ev.data ? ev.data : {};
  switch (ev.type) {
    case "inbox.created":
      if (!d.id) return null;
      return list.some((it) => it.id === d.id) ? list.map((it) => (it.id === d.id ? d : it)) : [d, ...list];
    case "inbox.updated":
      if (!d.id) return null;
      return list.some((it) => it.id === d.id) ? list.map((it) => (it.id === d.id ? d : it)) : list;
    case "inbox.deleted":
      return list.filter((it) => it.id !== d.id);
    case "inbox.cleared":
      return list.filter((it) => it.state !== "done");
    default:
      return list;
  }
}

// applyAutomations(items, ev) -> next | null | same. Items are the API's
// automation views (with lastRun / running / sparkline); a created row
// lacks those, so creation refetches.
export function applyAutomations(items, ev) {
  const list = items || [];
  const d = ev && ev.data ? ev.data : {};
  switch (ev.type) {
    case "automation.created":
      return null;
    case "automation.updated":
      return list.some((a) => a.id === d.id) ? list.map((a) => (a.id === d.id ? { ...a, ...d } : a)) : null;
    case "automation.deleted":
      return list.filter((a) => a.id !== d.id);
    case "run.created":
    case "run.updated":
    case "run.finished": {
      if (!d.automationId) return list;
      if (!list.some((a) => a.id === d.automationId)) return list;
      return list.map((a) => (a.id === d.automationId ? { ...a, lastRun: d, running: d.status === "running" } : a));
    }
    default:
      return list;
  }
}

// applyRuns(runs, automationId, ev) -> next | same. runs newest first.
export function applyRuns(runs, automationId, ev) {
  const list = runs || [];
  const d = ev && ev.data ? ev.data : {};
  if (!ev.type.startsWith("run.") || d.automationId !== automationId) return list;
  if (ev.type === "run.created" && !list.some((r) => r.id === d.id)) return [d, ...list];
  return list.map((r) => (r.id === d.id ? d : r));
}

// touches(ev, entities) -> whether ev is about one of the entity prefixes.
export function touches(ev, entities) {
  const e = entity(ev.type);
  return entities.includes(e);
}
