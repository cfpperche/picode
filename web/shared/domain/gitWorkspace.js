// Which workspace a Git surface reads through, and which of the user's
// workspaces it can be pointed at — the same rule in the browser and on the
// phone (ADR-0022: the owner authorises the read; ADR-0095: mobile takes its
// Git tools through shared logic, never a desktop component).
//
// Pure, and shell-free: no React, no fetch, and no tab ids — the browser's
// `g:<key>` / `ownerIdOf` identities stay in its own lib.

import { locate } from "./tree.js";

// pickerOptions is the list a workspace picker shows: the workspaces whose
// folder is a repository. The test is the same one the sidebar makes before
// offering "Git graph" (`ws.git` is nil on a plain folder), so a row is never
// offered for a folder the route would answer 404 on — an unavailable choice
// is hidden, not disabled.
export function pickerOptions(workspaces) {
  return (workspaces || [])
    .filter((ws) => ws && ws.id && ws.git)
    .map((ws) => ({
      id: ws.id,
      label: ws.name || ws.path || ws.id,
      hint: hintOf(ws),
      title: ws.path || "",
    }))
    .sort((a, b) => a.label.localeCompare(b.label, undefined, { sensitivity: "base" }));
}

// A row's hint: the branch that folder is on. The folder itself is the row's
// title — a deep worktree path elided to `~/picode/.worktrees/gg-wo…` is not a
// hint, it is noise (a first draft shipped exactly that). The branch is short,
// and it is the fact that tells two folders of one project apart.
function hintOf(ws) {
  return (ws && ws.git && ws.git.branch) || "";
}

// workspaceForOwner is the workspace an owner belongs to, so a surface can name
// it and mark it in the list. Every kind answers: a workspace is itself; an
// agent resolves through the workspace it lives in (a free agent has none); a
// terminal through the workspace it was created in (a free terminal has none).
// null means the owner is not attached to a workspace — the control then wears
// the repository name.
export function workspaceForOwner(owner, { workspaces, freeAgents, terminals } = {}) {
  if (!owner || !owner.id) return null;
  const byId = (id) => (workspaces || []).find((w) => w && w.id === id) || null;
  if (owner.kind === "workspace") return byId(owner.id);
  if (owner.kind === "term") {
    // The same reading as the browser's termGroups.termWorkspaceId: a terminal
    // carries the workspace it was born in, and "ws_free" is nobody's.
    const t = (terminals || []).find((x) => x && x.id === owner.id);
    const wsId = (t && t.workspaceId) || "";
    return wsId === "ws_free" ? null : byId(wsId);
  }
  const loc = locate(workspaces, freeAgents, owner.id);
  return (loc && loc.workspace) || null;
}

// triggerLabel is what the control wears: the workspace's name, or the
// repository's when there is no workspace to name.
export function triggerLabel(workspace, fallback) {
  return (workspace && (workspace.name || workspace.path)) || fallback || "";
}
