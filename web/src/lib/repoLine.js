// The sidebar's second line: where this thing lives, and on what branch.
// One shape for agents and terminals, so the two lists read the same. The
// line renders as two pills — dir (opens the file tree) and branch (opens
// the git graph) — so `dir` is exposed on its own beside the joined text.

export function shortPath(p) {
  const s = String(p || "").replace(/\\/g, "/").replace(/\/+$/, "");
  if (!s) return "—";
  const m = s.match(/^\/(?:home|Users)\/[^/]+(?:\/(.*))?$/);
  if (m) return m[1] ? "~/" + m[1] : "~";
  return s;
}

// gitText: `path / branch`, spaced so the branch cannot be misread as a
// subfolder. The worktree name is not appended — it is already the last
// segment of the path itself (~/picode/.worktrees/foo).
function gitText(path, g) {
  return [shortPath(path), g.branch].filter(Boolean).join(" / ");
}

// repoLine describes an agent's line. `git` is the git info object itself
// (or null) — a caller wanting the branch for a tooltip reads it directly;
// the earlier boolean made `repo.git.branch` silently undefined.
export function repoLine(ag, ws) {
  // The git info must describe the same directory as the path beside it. An
  // agent on its own workPath speaks only for that dir — falling back to the
  // workspace's git there would print the agent's path with the workspace's
  // branch, a pair that describes nothing.
  const ownPath = ag && ag.workPath;
  const g = (ownPath ? ag.git : (ag && ag.git) || (ws && ws.git)) || null;
  const path = ownPath || (ws && ws.path) || "";
  if (g && (g.branch || g.worktree)) {
    return { git: g, dir: shortPath(path), text: gitText(path, g) };
  }
  return { git: null, dir: shortPath(path), text: shortPath(path) };
}

// termLine is the same line for a terminal, whose path is its live pane cwd.
export function termLine(t) {
  const g = (t && t.git) || null;
  const path = (t && t.cwd) || "";
  if (g && (g.branch || g.worktree)) {
    return { git: g, dir: shortPath(path), text: gitText(path, g) };
  }
  return { git: null, dir: shortPath(path), text: shortPath(path) };
}
