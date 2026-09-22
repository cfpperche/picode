// The rows of the "…" menu on a workspace card, in the shape of
// agentRowMenu.js: pure, so what the menu offers follows what the folder is
// and the node tests hold the sidebar to it.
//
// Five groups, most-used first and the dangerous one last:
//   1. inside PiCode — Communication, Files, Git graph, Sessions
//   2. out of PiCode — the folder in the owner's file manager (Explorer under
//      WSL), the repository's web page, its pull request, and the path
//   3. Settings… — the name and how delivered work lands (WorkspaceSettings)
//   4. order — Move up / Move down, only the moves that exist
//   5. Remove workspace
// Every folder can be shown in the file manager, a repository included; the
// web page needs a remote with a web host, so a plain folder or a local-only
// repository simply has no such row. Items that cannot work are hidden,
// never greyed (uiux-review).
//
// The pull-request row reads `pr`, the GET /api/workspaces/{id}/pr page
// (ADR-0078), which the renderer fetches when the menu opens — GitHub only,
// since that page goes through gh. undefined means still asking: the row says
// so in its label rather than popping in late. Any other host, or a GitHub
// answer without a pull request, offers the host's "new pull request" page for
// the current branch — never for the remote's default branch, where there is
// nothing to propose.

const HOST_ACTION = { github: "pull request", bitbucket: "pull request", gitlab: "merge request" };

// newRequestURL is the host's "propose this branch" page, or "" when the host
// has none PiCode knows (Azure DevOps and self-hosted remotes stay link-only).
export function newRequestURL(remote, branch) {
  const b = encodeURIComponent(branch);
  switch (remote && remote.kind) {
    case "github": return remote.url + "/compare/" + b + "?expand=1";
    case "gitlab": return remote.url + "/-/merge_requests/new?merge_request%5Bsource_branch%5D=" + b;
    case "bitbucket": return remote.url + "/pull-requests/new?source=" + b;
    default: return "";
  }
}

// proposable is the branch a pull request could come from: the checkout's,
// unless it is the remote's default branch.
function proposable(ws) {
  const branch = String((ws.git && ws.git.branch) || "").trim();
  return branch && branch !== (ws.remote && ws.remote.defaultBranch) ? branch : "";
}

function requestRow(ws, remote, pr) {
  const noun = HOST_ACTION[remote.kind];
  if (!noun) return null;
  const branch = proposable(ws);
  if (!branch) return null;
  if (remote.kind === "github") {
    if (pr === undefined) return { id: "pr", label: "Checking pull request…", pending: true };
    const found = pr && pr.status === "ok" && pr.pr && pr.pr.url ? pr.pr : null;
    if (found) {
      const state = found.state && found.state !== "open" ? " · " + found.state : "";
      return { id: "pr", label: "Open pull request #" + found.number + state, url: found.url, title: found.title || "" };
    }
  }
  const url = newRequestURL(remote, branch);
  return url ? { id: "pr", label: "Create " + noun, url, title: "Propose " + branch + " on " + remote.host + "." } : null;
}

export function workspaceRowMenu(ws = {}, { hasAgents = false, canMoveUp = false, canMoveDown = false, pr } = {}) {
  const repo = !!(ws.git && (ws.git.branch || ws.git.worktree));
  const remote = ws.remote && ws.remote.url ? ws.remote : null;
  const inside = [
    { id: "communication", label: "Communication" },
    { id: "files", label: "Files" },
    ...(repo ? [{ id: "git-graph", label: "Git graph" }] : []),
    // Sessions read through an agent — an empty workspace answers 409, so it
    // does not offer the item (ADR-0027).
    ...(hasAgents ? [{ id: "sessions", label: "Sessions" }] : []),
  ];
  const request = remote && repo ? requestRow(ws, remote, pr) : null;
  const copy = ws.winPath
    ? { id: "copy-path", label: "Copy path", sub: [
      { id: "copy-linux", label: "Linux path", value: ws.path, title: ws.path },
      { id: "copy-windows", label: "Windows path", value: ws.winPath, title: ws.winPath },
    ] }
    : { id: "copy-path", label: "Copy path", value: ws.path, title: ws.path };
  const outside = [
    { id: "reveal", label: ws.winPath ? "Show in Explorer" : "Show in file manager" },
    ...(remote ? [{ id: "remote", label: "Open on " + remote.host, url: remote.url, title: remote.url }] : []),
    ...(request ? [request] : []),
    ...(ws.path ? [copy] : []),
  ];
  const own = [{ id: "settings", label: "Settings…" }];
  const order = [
    ...(canMoveUp ? [{ id: "move-up", label: "Move up" }] : []),
    ...(canMoveDown ? [{ id: "move-down", label: "Move down" }] : []),
  ];
  return [
    ...inside,
    { sep: true },
    ...outside,
    { sep: true },
    ...own,
    ...(order.length ? [{ sep: true }, ...order] : []),
    { sep: true },
    { id: "remove", label: "Remove workspace", danger: true },
  ];
}

// wantsPullRequest: whether opening the menu should ask for the pull-request
// page at all — only a GitHub repository off its default branch has one to
// ask about.
export function wantsPullRequest(ws = {}) {
  return !!(ws.remote && ws.remote.kind === "github" && proposable(ws));
}
