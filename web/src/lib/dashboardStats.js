import { agentsOf } from "./tree.js";

// deltaPercent: null (not a chip-worthy comparison) when there's no prior
// window (range=all) or the prior total was zero — "vs $0" isn't a
// percentage worth showing, and would divide by zero.
export function deltaPercent(current, prior) {
  if (prior == null || !prior) return null;
  return ((current - prior) / prior) * 100;
}

// fleetStats: agents running now vs. total, across every workspace plus
// free agents. Reuses agentsOf() (lib/tree.js) so this counts exactly the
// same set Sidebar/HomeView-style views already do, including its legacy
// single-`agent`-object fallback.
export function fleetStats(workspaces, freeAgents) {
  let running = 0;
  let total = 0;
  const count = (agents) => {
    for (const a of agents) {
      total++;
      if (a && a.mode && a.mode !== "stopped") running++;
    }
  };
  for (const ws of workspaces || []) count(agentsOf(ws));
  count(freeAgents || []);
  return { running, total };
}

const RANGE_LABELS = { today: "Today", "7d": "7 days", "30d": "30 days", all: "All time" };

export function rangeLabel(range) {
  return RANGE_LABELS[range] || RANGE_LABELS["7d"];
}

const COMPARE_LABELS = { today: "vs. yesterday", "7d": "vs. prior 7 days", "30d": "vs. prior 30 days" };

// compareLabel: "" for range=all — there's no prior-of-all-time to compare
// against, so the stat tile renders no delta chip at all for that range.
export function compareLabel(range) {
  return COMPARE_LABELS[range] || "";
}
