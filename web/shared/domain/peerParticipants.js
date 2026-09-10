export const participantKey = p => `${p.kind}:${p.ownerId}`;
export function participantState(owner, preference, connection, live, checks = []) {
  if (!preference?.enabled || preference.workspaceId !== owner.workspaceId || preference.cli !== owner.cli) return { label: connection?.active ? "Current conversation only" : "Off", kind: "off" };
  if (!owner.sessionKey) return { label: "Needs a conversation", kind: "action" };
  if (preference.problem && preference.phase === "error") return { label: "Needs attention", kind: "error" };
  if (!live) return { label: "Stopped", kind: "action" };
  if (live === "needs-you") return { label: "Needs your input", kind: "action" };
  if (preference.problem || preference.phase === "error") return { label: "Needs attention", kind: "error" };
  if (preference.phase === "connected" && connection?.active && preference.appliedConnection === connection.id) {
    const verified = checks.some(c => c.phase === "passed" && [c.senderId, c.recipientId].includes(connection.id));
    return { label: verified ? "Verified" : "Connected · not tested", kind: "connected" };
  }
  if (live === "working" || preference.phase === "waiting") return { label: "Waiting to connect", kind: "waiting" };
  return { label: "Preparing…", kind: "preparing" };
}
export function selectedWorkspace(data, route) {
  if (!data) return "";
  if (route.startsWith("workspace:")) return route.slice(10);
  if (route) return data.owners.find(o => participantKey(o) === route)?.workspaceId || "";
  return data.workspaces.length === 1 ? data.workspaces[0].id : "";
}
