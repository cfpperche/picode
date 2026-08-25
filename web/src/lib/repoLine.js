export function shortPath(p) {
  const s = String(p || "").replace(/\\/g, "/").replace(/\/+$/, "");
  if (!s) return "—";
  const m = s.match(/^\/(?:home|Users)\/[^/]+\/(.*)$/);
  if (m) return "~/" + m[1];
  return s;
}

export function repoLine(ag, ws) {
  const g = (ag && ag.git) || (ws && ws.git);
  if (g && (g.branch || g.worktree)) {
    const text = [g.worktree, g.branch].filter(Boolean).join(" / ");
    return { git: true, text };
  }
  const path = (ag && ag.workPath) || (ws && ws.path) || "";
  return { git: false, text: shortPath(path) };
}
