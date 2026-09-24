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

// restoredToast is the one message a restore leaves, from the history page
// or from Undo: back and running, back but not started (and why), plus the
// environment variables to set again (their values were never kept).
export function restoredToast(name, out) {
  const keys = (out && out.envKeys) || [];
  const note = keys.length ? ` Set ${keys.join(", ")} again in its launch settings.` : "";
  if (out && out.startError) return { ok: false, text: `"${name}" is back, but it didn't start: ${out.startError}.` + note };
  return { ok: true, text: `"${name}" is back.` + note };
}
