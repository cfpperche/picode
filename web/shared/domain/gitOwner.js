// Git reads resolve through an owner, never through a repository path
// (ADR-0022). The three owner kinds answer the same routes under their own
// namespace, so a surface only has to know who asked: agents and terminals
// read through the process, a workspace reads through its folder — which is
// the only owner an empty workspace has (ADR-0027/ADR-0030).
export function ownerNamespace(kind) {
  if (kind === "term") return "terminals";
  if (kind === "workspace") return "workspaces";
  return "agents";
}

// ownerBase is that namespace as the prefix the surfaces concatenate onto:
// `${base}${id}/git`. An absent owner keeps the agent default the callers
// had, so a transiently undefined owner never builds "/api/undefined/".
export function ownerBase(owner) {
  return "/api/" + ownerNamespace(owner ? owner.kind : "") + "/";
}
