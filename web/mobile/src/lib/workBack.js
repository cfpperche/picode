import { FREE_WS, termWorkspaceId } from "./termGroups.js";

// Ownership of a pushed agent or terminal, feeding the parentHash wsId
// contract (Back lands where the resource lives): the owning workspace
// id, null when the resource is free, undefined when the fleet has not
// answered yet — absence is not conclusive before its source responded.
export function agentOwnerWs(found) {
  if (!found) return undefined;
  return found.workspace ? found.workspace.id : null;
}

// A payload without workspaceId (a partial record) is free.
export function termOwnerWs(term) {
  if (!term) return undefined;
  const wsId = termWorkspaceId(term);
  return wsId === FREE_WS ? null : wsId;
}
