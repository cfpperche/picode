import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "@picode/shared/client/api.js";
import { agentsOf } from "@picode/shared/domain/tree.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { applyFleet, touches } from "@picode/shared/domain/feedReducers.js";
import { usePoll } from "./usePoll.js";
import { emptyFleet, mergeFleetReads } from "../lib/fleetReads.js";
import { createFleetReload } from "../lib/fleetReload.js";

// The fleet: every workspace with its agents, plus free agents. Since
// ADR-0044 each agent carries streaming / waiting / dialog, so this one
// poll is what the Now screen reads "who needs me" from.
export function useFleet(ms) {
  const [fleet, setFleet] = useState(emptyFleet);
  const loader = useRef(null);
  const events = useRef([]);
  if (!loader.current) loader.current = createFleetReload(() => {
    events.current = [];
    return Promise.allSettled([
      api("/api/workspaces"), api("/api/agents?free=1"), api("/api/terminals"),
    ]).then(reads => {
      const duringRead = events.current;
      setFleet(previous => {
        let next = mergeFleetReads(previous, reads);
        for (const event of duringRead) {
          const patch = applyFleet(next, event);
          if (patch) next = { ...next, ...patch };
        }
        return next;
      });
      return reads[0].status === "fulfilled" ? reads[0].value : null;
    });
  });
  const reload = useCallback(options => loader.current.reload(options), []);
  usePoll(reload, ms);
  // Change feed (ADR-0048): patch in place; anything the reducer cannot
  // apply faithfully falls back to one reload.
  const ref = useRef(fleet);
  ref.current = fleet;
  useEffect(() => subscribeFeed((ev) => {
    if (!touches(ev, ["workspace", "agent", "terminal", "git"])) return;
    if (loader.current.pending) events.current.push(ev);
    const next = applyFleet(ref.current, ev);
    if (next === null) {
      reload({ force: true }).catch(() => {});
      return;
    }
    if (next === ref.current) return;
    setFleet(current => ({ ...current, ...applyFleet(current, ev) }));
  }), [reload]);
  return { ...fleet, reload };
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
