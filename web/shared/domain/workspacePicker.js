// Which workspace a surface reads through — the Git graph, the file tree, and
// the phone's Git screen all ask it — and which of the user's workspaces a
// picker may offer. The same rule everywhere (ADR-0022: the owner authorises
// the read; ADR-0095: mobile takes its Git tools through shared logic, never a
// desktop component).
//
// Pure, and shell-free: no React, no fetch, and no tab ids — the browser's
// `g:<key>` / `ownerIdOf` identities stay in its own lib.

import { locate } from "./tree.js";
import { shortPath } from "./repoLine.js";

// pickerOptions is the list a workspace picker shows: the user's folders.
//
// `reposOnly` is the Git surfaces' question — the test is the same one the
// sidebar makes before offering "Git graph" (`ws.git` is nil on a plain
// folder), so a row is never offered for a folder the route would answer 404
// on. The file tree asks for no such filter: it draws any folder, decorated
// when that folder is a repository (ADR-0030: no repository is a state, not an
// error). A workspace without a folder is never offered — there is nothing to
// read.
//
// `hint` picks the row's second fact: the branch that folder is on (Git's
// question) or the folder itself (the tree's). The full path is always the
// row's title — a deep worktree path elided to `~/picode/.worktrees/gg-wo…` is
// not a hint, it is noise (a first draft shipped exactly that).
export function pickerOptions(workspaces, { reposOnly = false, hint = "branch" } = {}) {
  return (workspaces || [])
    .filter((ws) => ws && ws.id && ws.path && (!reposOnly || ws.git))
    .map((ws) => ({
      id: ws.id,
      label: ws.name || ws.path || ws.id,
      hint: hint === "path" ? hintPath(ws.path) : ((ws.git && ws.git.branch) || ""),
      title: ws.path || "",
    }))
    .sort((a, b) => a.label.localeCompare(b.label, undefined, { sensitivity: "base" }));
}

// hintPath is the folder as a row can wear it: `~/picode` when it fits, and the
// path's *tail* when it does not — `…/repos/alpha` names the folder, while the
// head elided to `~/picode/.worktrees/tree-w…` names nothing (a first draft
// shipped exactly that). One long segment has no tail to keep and comes back
// whole, over the bound: the row's own ellipsis takes it from there rather than
// the path being cut mid-name here.
export function hintPath(path, max = 30) {
  const raw = String(path || "").trim();
  // shortPath reads a missing path as an em dash; a row hint has nothing to say
  // then, which is the empty string, not a dash.
  if (!raw) return "";
  const s = shortPath(raw);
  if (s.length <= max) return s;
  const parts = s.split("/");
  let out = parts.pop() || "";
  while (parts.length) {
    const next = parts[parts.length - 1] + "/" + out;
    if (next.length + 2 > max) break;
    out = next;
    parts.pop();
  }
  return "…/" + out;
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
