// Agent history (ADR-0205): what the restore row offers and picks.

const FREE = "ws_free";

// historyWorkspaceOptions lists where a removed agent can come back: every
// living workspace, then the free list. Pairs of [id, name].
export function historyWorkspaceOptions(workspaces) {
  const out = [];
  for (const w of workspaces || []) {
    if (!w || !w.id || w.id === FREE) continue;
    out.push([w.id, w.name || w.id]);
  }
  out.push([FREE, "Free agents"]);
  return out;
}

// historyWorkspaceChoice is the workspace a restore targets: the person's
// pick, else where the agent was while that workspace exists, else none
// (the row then asks for one instead of guessing).
export function historyWorkspaceChoice(entry, picked) {
  if (picked) return picked;
  if (entry && entry.workspaceExists && entry.exit && entry.exit.workspaceId) return entry.exit.workspaceId;
  return "";
}
