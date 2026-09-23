// Sidebar reorder (ADR-0173). The fleet reducer is the one applier; a drop
// and a menu move both go through it, then the same ids are PUT.

import { applyFleet } from "@picode/shared/domain/feedReducers.js";
import { api } from "@picode/shared/client/api.js";
import { agentsOf } from "@picode/shared/domain/tree.js";
import { FREE_WS, termWorkspaceId } from "./termGroups.js";

export function sameIds(a, b) {
  if (!a || !b || a.length !== b.length) return false;
  for (let i = 0; i < a.length; i++) if (a[i] !== b[i]) return false;
  return true;
}

// Move id by delta within ids. Out of range or unknown returns the same array.
export function moveId(ids, id, delta) {
  const from = (ids || []).indexOf(id);
  const to = from + delta;
  if (from < 0 || to < 0 || to >= ids.length) return ids;
  const next = ids.slice();
  const [row] = next.splice(from, 1);
  next.splice(to, 0, row);
  return next;
}

// Drop active onto over. Same place returns the same array.
export function reorderIds(ids, activeId, overId) {
  if (!overId || activeId === overId) return ids;
  const from = (ids || []).indexOf(activeId);
  const to = ids.indexOf(overId);
  if (from < 0 || to < 0) return ids;
  const next = ids.slice();
  const [row] = next.splice(from, 1);
  next.splice(to, 0, row);
  return next;
}

// Sort one container's ids by a display key — the "Sort agents by name"
// action. Pure: the result still goes through the same reorder path as a
// drop (ADR-0173), so every client converges on one stored order and the
// next drag lands where it is dropped. Comparison is case-insensitive and
// numeric ("agent 2" before "agent 10"); equal or missing keys keep the
// stored position, so a sort that changes nothing writes nothing.
export function sortIdsBy(ids, keyOf) {
  const list = ids || [];
  const rank = new Map(list.map((id, i) => [id, i]));
  const key = (id) => String(keyOf(id) ?? "");
  return [...list].sort((a, b) =>
    key(a).localeCompare(key(b), undefined, { sensitivity: "base", numeric: true }) || rank.get(a) - rank.get(b)
  );
}

export function movePhrase(before, after, id, label) {
  const i = (after || []).indexOf(id);
  if (i < 0 || sameIds(before, after)) return "";
  const name = label(id) || "Item";
  if (i === 0) return "Moved " + name + " to the top";
  return "Moved " + name + " below " + (label(after[i - 1]) || "the item above");
}

function ownedTerminalIds(fleet) {
  const owned = new Set();
  for (const ws of fleet.workspaces || []) {
    for (const a of agentsOf(ws)) if (a.terminalId) owned.add(a.terminalId);
  }
  for (const a of fleet.freeAgents || []) if (a && a.terminalId) owned.add(a.terminalId);
  return owned;
}

// The ids of one sidebar container, in the order it is showing.
export function containerIds(fleet, kind, workspaceId) {
  if (!fleet) return [];
  if (kind === "workspaces") return (fleet.workspaces || []).map((w) => w.id);
  if (kind === "agents") {
    if (!workspaceId || workspaceId === FREE_WS) return (fleet.freeAgents || []).map((a) => a.id);
    const ws = (fleet.workspaces || []).find((w) => w.id === workspaceId);
    return ws ? agentsOf(ws).map((a) => a.id) : [];
  }
  const owned = ownedTerminalIds(fleet);
  const wsId = workspaceId || FREE_WS;
  return (fleet.terminals || [])
    .filter((t) => termWorkspaceId(t) === wsId && !owned.has(t.id))
    .map((t) => t.id);
}

// Returns the same fleet when ids already match, or null when the reducer
// cannot apply them.
export function optimisticFleet(fleet, kind, workspaceId, ids) {
  if (sameIds(containerIds(fleet, kind, workspaceId), ids)) return fleet;
  const type = kind === "workspaces" ? "workspace.reordered" : kind === "agents" ? "agent.reordered" : "terminal.reordered";
  const data = kind === "workspaces" ? { ids } : { workspaceId: workspaceId || FREE_WS, ids };
  return applyFleet(fleet, { type, data });
}

export function putSidebarOrder(kind, workspaceId, ids) {
  const path = kind === "workspaces"
    ? "/api/workspaces/order"
    : "/api/workspaces/" + encodeURIComponent(workspaceId || FREE_WS) + (kind === "agents" ? "/agents/order" : "/terminals/order");
  return api(path, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ ids }),
  });
}
