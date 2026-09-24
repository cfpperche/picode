// Which icon a scope chip carries: this computer (globe), a workspace
// (folder) or one agent. Every setup pane names its scopes its own way
// (machine, user, global; workspace, project, local; agent), so the chips
// share one reading. Unknown scopes get no icon rather than a wrong one.
const KINDS = {
  machine: "global", user: "global", global: "global",
  workspace: "workspace", project: "workspace", local: "workspace",
  agent: "agent",
};

export function scopeKind(scope) {
  return KINDS[String(scope || "").toLowerCase()] || "";
}
