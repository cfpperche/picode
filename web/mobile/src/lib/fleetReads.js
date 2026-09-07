const KEYS = ["workspaces", "freeAgents", "terminals"];

export function emptyFleet() {
  return { workspaces: [], freeAgents: [], terminals: [], known: {}, loaded: false, error: "" };
}

// A found resource can open immediately. Only the sources that could
// contain a missing resource must answer before absence is conclusive.
export function fleetRouteReady(route, known, found = false) {
  if (found) return true;
  const kind = route.screen === "changes" ? route.section : route.screen;
  if (kind === "agent") return !!(known.workspaces && known.freeAgents);
  if (kind === "term") return !!known.terminals;
  if (route.screen === "changes") return !!known.workspaces;
  return true;
}

// One failed endpoint must never turn a previously known list into an empty
// list. Absence is conclusive only after every source has answered once.
export function mergeFleetReads(previous, reads) {
  const next = { ...previous, known: { ...previous.known } };
  const failed = [];
  KEYS.forEach((key, index) => {
    const read = reads[index];
    const rows = key === "terminals" ? read?.value?.terminals : read?.value;
    if (read?.status === "fulfilled" && Array.isArray(rows)) {
      next[key] = rows;
      next.known[key] = true;
    } else failed.push(key);
  });
  next.loaded = KEYS.every(key => next.known[key]);
  next.error = failed.length ? (next.loaded ? "Couldn’t refresh your work. Showing the last update." : "Couldn’t load your work.") : "";
  return next;
}
