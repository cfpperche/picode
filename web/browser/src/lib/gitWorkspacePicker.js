// The git graph's first toolbar item: which workspace's folder the history is
// read through. The tab is the repository (ADR-0022) and the owner is who
// authorises the read and where a write is delivered (ADR-0096), so picking a
// workspace retargets the owner — it never resolves a repository from a path.
//
// Pure: no React, no fetch. The surface renders these; the App turns a pick
// into a tab move.
//
// The rule itself — which workspaces a picker may offer, and which one an owner
// belongs to — lives in @picode/shared/domain/gitWorkspace.js, because the
// phone asks the same question and an app may not import another app.

import { ownerBase } from "@picode/shared/domain/gitOwner.js";
import { isGitTab, gitTabKey } from "./routes.js";

// pickWorkspace resolves a pick into the move the App makes, reading the picked
// workspace's own /git/head — a repository is never resolved from a path in a
// URL (ADR-0022). `readHead` is injected so the two failure rows are tests and
// not hopes: a request that throws and an answer that names no repository both
// leave the reader exactly where they were (`none`), and the difference is only
// whether there is an error to show. The caller keeps the tab: nothing here
// touches state, which is what makes it testable.
export async function pickWorkspace({ tabKey, openKeys, readHead }) {
  let head = null;
  let error = null;
  try {
    head = await readHead();
  } catch (err) {
    error = err;
  }
  const key = (head && head.key) || "";
  return { key, action: pickAction(key, tabKey, openKeys), error };
}

// ownerIdOf is the identity a graph surface's per-owner state hangs on: the
// open commit and a pending/undo banner belong to the folder that asked
// (ADR-0096), so this is what the reset watches. The kind is part of it, like
// the owner letter in a tab id — an agent and a terminal that happen to share
// an id are still two different folders.
//
// It is NOT what a route takes. Keep the two apart: `ownerRoute` below is the
// URL, this is the state key.
export function ownerIdOf(owner) {
  if (!owner || !owner.id) return "";
  return `${owner.kind || "agent"}:${owner.id}`;
}

// ownerRoute is the pair a request is composed from: the namespace the owner
// kind answers under, and the plain id. One function so a surface cannot hand
// the state key to a URL — asking the server for a workspace literally named
// `workspace:<id>` is a 404, and it cost the graph one round trip on
// 2026-09-13.
export function ownerRoute(owner) {
  return { base: ownerBase(owner), id: (owner && owner.id) || "" };
}

// graphUrl and headUrl are the only two requests a graph surface makes: the
// history it draws and the cheap token the pending watch polls. Both take the
// route and the plain id — the state key (ownerIdOf) never rides a URL.
export function graphUrl(base, id, params) {
  return `${base}${encodeURIComponent(id || "")}/git?${params}`;
}

export function headUrl(base, id) {
  return `${base}${encodeURIComponent(id || "")}/git/head`;
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
