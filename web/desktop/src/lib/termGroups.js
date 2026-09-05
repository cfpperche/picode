// Terminals carry a workspaceId (ADR-0026); ws_free means nobody's.
// Grouping lives here, client-side — the wire stays a flat list.
export const FREE_WS = "ws_free";

// A payload without the field (older server, cached state) is a free terminal.
export function termWorkspaceId(t) {
  return (t && t.workspaceId) || FREE_WS;
}

export function sortTermsByName(list) {
  return [...(list || [])].sort((a, b) =>
    String(a.name || "").localeCompare(String(b.name || ""), undefined, { sensitivity: "base" }));
}

export function freeTerminals(terminals) {
  return sortTermsByName((terminals || []).filter((t) => termWorkspaceId(t) === FREE_WS));
}

export function workspaceTerminals(terminals, wsId) {
  return sortTermsByName((terminals || []).filter((t) => termWorkspaceId(t) === wsId));
}

// Workspace a terminal tab belongs to. Free terminals (and unknown ids) yield null.
export function workspaceForTerminal(terminals, workspaces, termId) {
  const t = (terminals || []).find((x) => x && x.id === termId);
  const wsId = termWorkspaceId(t);
  if (!t || wsId === FREE_WS) return null;
  return (workspaces || []).find((w) => w && w.id === wsId) || null;
}
