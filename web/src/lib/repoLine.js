export function repoLine(ag, ws) {
  const g = (ag && ag.git) || (ws && ws.git);
  if (g && (g.branch || g.worktree)) {
    const text = [g.worktree, g.branch].filter(Boolean).join(" / ");
    return { git: true, text };
  }
  return { git: false, text: "local" };
}
