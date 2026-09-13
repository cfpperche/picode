// The git graph's first toolbar item: which workspace's folder the history is
// read through. The tab is the repository (ADR-0022) and the owner is who
// authorises the read and where a write is delivered (ADR-0096), so picking a
// workspace retargets the owner — it never resolves a repository from a path.
//
// Pure: no React, no fetch. The surface renders these; the App turns a pick
// into a tab move.

import { locate } from "@picode/shared/domain/tree.js";
import { isGitTab, gitTabKey } from "./routes.js";
import { workspaceForTerminal } from "./termGroups.js";

// pickerOptions is the list the popover shows: the workspaces whose folder is
// a repository. The test is the same one the sidebar makes before offering
// "Git graph" (ws.git is nil on a plain folder), so a row is never offered for
// a folder the route would answer 404 on — an unavailable choice is hidden,
// not disabled (uiux-review).
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

// A row's right-hand hint: the branch that folder is on. The folder itself is
// the row's title — a deep worktree path elided to `~/picode/.worktrees/gg-wo…`
// is not a hint, it is noise (a first draft shipped exactly that). The branch
// is short, and it is the fact that tells two folders of one project apart.
function hintOf(ws) {
  return (ws && ws.git && ws.git.branch) || "";
}

// currentWorkspaceId is which workspace the owner belongs to, so the trigger
// can name it and the popover can mark it. Every kind answers: a workspace is
// itself; an agent resolves through the workspace it lives in (a free agent
// has none); a terminal through the workspace it was created in (a free
// terminal has none). "" means the owner is not attached to a workspace — the
// trigger then wears the repository name, which is what the item showed
// before it became a control.
export function currentWorkspaceId(owner, { workspaces, freeAgents, terminals } = {}) {
  if (!owner || !owner.id) return "";
  if (owner.kind === "workspace") {
    return (workspaces || []).some((w) => w && w.id === owner.id) ? owner.id : "";
  }
  if (owner.kind === "term") {
    const ws = workspaceForTerminal(terminals, workspaces, owner.id);
    return ws ? ws.id : "";
  }
  const loc = locate(workspaces, freeAgents, owner.id);
  return loc && loc.workspace ? loc.workspace.id : "";
}

// triggerLabel is what the control wears: the workspace's name, or the
// repository name when there is no workspace to name.
export function triggerLabel(workspace, repoName) {
  return (workspace && (workspace.name || workspace.path)) || repoName || "";
}

// openRepoKeys is the repositories already on the tab strip. A provisional tab
// (`g:@t:<id>`, opened before its key is known) is not a repository yet:
// adopting it would send a pick to a tab that is about to rename itself.
export function openRepoKeys(tabs) {
  return (tabs || [])
    .filter(isGitTab)
    .map(gitTabKey)
    .filter((key) => key && !key.startsWith("@"));
}

// pickAction says where a pick lands. `targetKey` is the picked workspace's
// repository (its own /git/head, read before anything moves), `currentKey` the
// tab's (gitTabKey), `openKeys` the strip's (openRepoKeys):
//   same   — the tab stays and only the owner changes (sibling worktrees);
//   rename — this tab becomes the other repository's, so one tab per
//            repository still holds;
//   adopt  — that repository already has a tab: select it and it takes the
//            picked owner, because an explicit pick wins over the owner the
//            tab happened to be opened with;
//   none   — no repository resolved, which is nothing to do, not a retarget.
export function pickAction(targetKey, currentKey, openKeys) {
  if (!targetKey) return "none";
  if (targetKey === currentKey) return "same";
  return (openKeys || []).includes(targetKey) ? "adopt" : "rename";
}
