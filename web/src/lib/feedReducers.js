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
    case "agent.status": {
      // lastStatus alone cannot say managed vs interactive: a start needs a
      // refetch; a stop can be applied.
      const running = d.lastStatus === "running";
      let found = false;
      const patch = (a) => {
        if (a.id !== d.id) return a;
        found = true;
        return running ? a : { ...a, running: false, mode: "stopped", streaming: false, waiting: false, dialog: undefined, lastStatus: d.lastStatus };
      };
      const next = { ...state, workspaces: workspaces.map((w) => ({ ...w, agents: (w.agents || []).map(patch) })), freeAgents: freeAgents.map(patch) };
      if (!found) return state;
      return running ? null : next;
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
    default:
      return state;
  }
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
