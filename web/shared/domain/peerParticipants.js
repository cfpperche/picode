export const participantKey = p => `${p.kind}:${p.ownerId}`;
// Activity, connection preparation and historical test proof are independent.
export function participantState(owner, preference, connection, live, checks = [], identity, recovery) {
  const enabled = preference?.enabled && preference.workspaceId === owner.workspaceId && preference.cli === owner.cli;
  if (!enabled) return { label: connection?.active ? "Current conversation only" : "Off", connection: "", kind: "off", ready: false, next: "" };
  const phase = preference.phase;
  const observed = identity !== "unobserved" && !!owner.sessionKey;
  const ready = observed && !!live && phase === "connected" && connection?.active && preference.appliedConnection === connection.id;
  const verified = ready && checks.some(c => c.phase === "passed" && [c.senderId, c.recipientId].includes(connection.id));
  const activity = ({idle:"Idle",working:"Working","needs-you":"Needs your input"})[live] || "Syncing";
  const activation = owner.kind === "terminal" && !!connection?.active && (live === "open" || live === "idle") && !ready && identity === "unobserved" && phase === "waiting-conversation";
  const result = { label: activity, connection: ready ? "Connected" : "Connecting", kind: ready ? "connected" : "preparing", ready: !!ready, verified: !!verified, reason: "", action: "", activation, next: "" };
  if (!live) return {...result,label:"Stopped",connection:"Not connected",kind:"action",reason:"stopped",action:"open",next:"Open this conversation to connect."};
  if (recovery === "restart-required") return {...result,label:"State unavailable",connection:"Connection failed",kind:"error",ready:false,verified:false,reason:"restart-required",action:"terminal-controls",next:"Restore state access, then restart this CLI."};
  if (!owner.sessionKey) return {...result,label:"No conversation",connection:"Not connected",kind:"action",reason:"no-conversation",action:"open",next:"Open the conversation and send its first message."};
  if (!observed || phase === "waiting-conversation") return {...result,label:live === "needs-you" ? activity : "First message needed",connection:"Waiting for identity",kind:"waiting",ready:false,verified:false,reason:"unobserved",action:"open",next:"Send one message in this conversation to finish connecting."};
  if (phase === "adapter-missing") return {...result,connection:"Connection failed",kind:"error",reason:"adapter-missing",action:"packages",next:"Enable the communication adapter in Packages."};
  if (phase === "waiting-receiver") return {...result,connection:"Waiting to reconnect",kind:"waiting",reason:"waiting-receiver",action:"open",next:"Reopen this conversation so its receiver can reconnect."};
  if (phase === "error") return {...result,connection:"Connection failed",kind:"error",reason:"setup-failed",action:owner.kind === "agent" ? "reconnect" : "retry",next:"Retry the connection after checking the conversation."};
  if (phase === "waiting") return {...result,connection:"Waiting for turn",kind:"waiting",reason:"waiting",action:"open",next:"Let the current turn finish, then reconnect."};
  return result;
}
export function participantOptions(owners, selectedFrom, selectedTo) {
  // Selection is by owner, so a reconnect or new conversation cannot silently
  // substitute a different participant while the owner is looking at this form.
  const sender = owners.find(o => participantKey(o) === selectedFrom) || owners[0];
  const receivers = owners.filter(o => participantKey(o) !== (sender && participantKey(sender)));
  const recipient = receivers.find(o => participantKey(o) === selectedTo) || receivers[0];
  return {sender,receivers,recipient};
}
export function selectedWorkspace(data, route) {
  if (!data) return "";
  if (route.startsWith("workspace:")) return route.slice(10);
  if (route) return data.owners.find(o => participantKey(o) === route)?.workspaceId || "";
  return data.workspaces.length === 1 ? data.workspaces[0].id : "";
}
