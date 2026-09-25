// Terminals carry a workspaceId (ADR-0026); ws_free means nobody's.
// Grouping lives here, client-side — the wire stays a flat list.
export const FREE_WS = "ws_free";

// A payload without the field (a partial record) is a free terminal.
export function termWorkspaceId(t) {
  return (t && t.workspaceId) || FREE_WS;
}

// Order is the server's sidebar position (ADR-0173). Grouping stays here;
// sorting again would throw away a reorder.
export function freeTerminals(terminals) {
  return (terminals || []).filter((t) => termWorkspaceId(t) === FREE_WS);
}

export function workspaceTerminals(terminals, wsId) {
  return (terminals || []).filter((t) => termWorkspaceId(t) === wsId);
}

// Workspace a terminal tab belongs to. Free terminals (and unknown ids) yield null.
export function workspaceForTerminal(terminals, workspaces, termId) {
  const t = (terminals || []).find((x) => x && x.id === termId);
  const wsId = termWorkspaceId(t);
  if (!t || wsId === FREE_WS) return null;
  return (workspaces || []).find((w) => w && w.id === wsId) || null;
}
