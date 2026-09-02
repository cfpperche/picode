import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "../../lib/api.js";
import { agentsOf } from "../../lib/tree.js";
import { subscribeFeed } from "../../lib/feed.js";
import { applyFleet, touches } from "../../lib/feedReducers.js";
import { usePoll } from "./usePoll.js";

// The fleet: every workspace with its agents, plus free agents. Since
// ADR-0044 each agent carries streaming / waiting / dialog, so this one
// poll is what the Now screen reads "who needs me" from.
export function useFleet(ms) {
  const [workspaces, setWorkspaces] = useState([]);
  const [freeAgents, setFreeAgents] = useState([]);
  const [terminals, setTerminals] = useState([]);
  const [loaded, setLoaded] = useState(false);
  const reload = useCallback(async () => {
    const list = await api("/api/workspaces");
    let free = [];
    let terms = [];
    try { free = await api("/api/agents?free=1"); } catch { free = []; }
    try { terms = (await api("/api/terminals")).terminals || []; } catch { terms = []; }
    setWorkspaces(Array.isArray(list) ? list : []);
    setFreeAgents(Array.isArray(free) ? free : []);
    setTerminals(Array.isArray(terms) ? terms : []);
    setLoaded(true);
    return list;
  }, []);
  usePoll(reload, ms);
  // Change feed (ADR-0048): patch in place; anything the reducer cannot
  // apply faithfully falls back to one reload.
  const ref = useRef({ workspaces, freeAgents, terminals });
  ref.current = { workspaces, freeAgents, terminals };
  useEffect(() => subscribeFeed((ev) => {
    if (!touches(ev, ["workspace", "agent", "terminal"])) return;
    const next = applyFleet(ref.current, ev);
    if (next === null) { reload().catch(() => {}); return; }
    if (next === ref.current) return;
    setWorkspaces(next.workspaces);
    setFreeAgents(next.freeAgents);
    setTerminals(next.terminals);
  }), [reload]);
  return { workspaces, freeAgents, terminals, loaded, reload };
}

// flatAgents: [{ agent, workspace|null }] in sidebar order.
export function flatAgents(workspaces, freeAgents) {
  const out = [];
  for (const ws of workspaces || []) for (const a of agentsOf(ws)) out.push({ agent: a, workspace: ws });
  for (const a of freeAgents || []) if (a && a.id) out.push({ agent: a, workspace: null });
  return out;
}

export function findAgent(workspaces, freeAgents, id) {
  if (!id) return null;
  return flatAgents(workspaces, freeAgents).find((x) => x.agent.id === id) || null;
}
