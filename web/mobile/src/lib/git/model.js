// Git reads resolve through an owner. root is an equality precondition;
// worktree is a server-resolved branch/head, never a client filesystem path.
export function gitURL(owner, resource, { root = "", path = "", hash = "", worktree = "", limit, branches = [], remotes, refresh } = {}) {
  if (!owner?.id || !["workspace", "agent", "term"].includes(owner.kind)) return "";
  const kind = { workspace: "workspaces", agent: "agents", term: "terminals" }[owner.kind];
  const q = new URLSearchParams();
  if (root) q.set("root", root);
  if (path) q.set("path", path);
  if (hash) q.set("hash", hash);
  if (worktree) q.set("worktree", worktree);
  if (limit) q.set("limit", String(limit));
  for (const branch of branches) if (branch) q.append("branches", branch);
  if (remotes === false) q.set("remotes", "0");
  if (refresh) q.set("refresh", "1");
  return `/api/${kind}/${encodeURIComponent(owner.id)}/${resource}${q.size ? "?" + q : ""}`;
}

export function isMoved(error) {
  return error?.status === 409 && (!!error?.body?.cwd || /moved|folder changed|root/i.test(error?.message || ""));
}

export function rootProblem(error) {
  return error?.body?.cwd ? `The working folder changed to ${error.body.cwd}.` : error?.message || "The working folder changed.";
}

export function fileKind(kind) {
  return ({ M: "Modified", A: "Added", D: "Deleted", R: "Renamed", C: "Copied", "?": "New", "??": "New", U: "Conflict", modified: "Modified", added: "Added", deleted: "Deleted", renamed: "Renamed", copied: "Copied", untracked: "New", conflict: "Conflict", conflicted: "Conflict" })[kind] || kind || "Changed";
}

export function isDeleted(file) { return file?.kind === "deleted" || file?.kind === "D" || file?.status === "deleted"; }
export function shortHash(hash) { return String(hash || "").slice(0, 7); }
export function historyDate(at) {
  if (!at) return "";
  return new Date(at * 1000).toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" });
}
export function worktreeRef(wt) {
  return wt && !wt.bare && !wt.prunable ? wt.branch || wt.head || "" : "";
}
export function worktreeName(wt) {
  return wt?.branch || String(wt?.path || "").split(/[\\/]/).filter(Boolean).pop() || shortHash(wt?.head);
}
export function branchSummary(status) {
  if (!status?.git) return "No Git repository";
  const branch = status.branch || "No commits yet";
  if (status.detached) return `${branch} · detached`;
  const distance = [status.ahead ? `${status.ahead} ahead` : "", status.behind ? `${status.behind} behind` : ""].filter(Boolean).join(" · ");
  return [branch, status.upstream ? distance : "unpublished"].filter(Boolean).join(" · ");
}

export const ACTION_LABELS = { fetch: "Fetch", pull: "Pull", push: "Push", commit: "Commit", "commit-push": "Commit and push", pr: "Create pull request" };
