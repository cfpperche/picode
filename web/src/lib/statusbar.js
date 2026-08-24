export function statusSegments(bar) {
  if (!bar) return [];
  const out = [];
  if (bar.cwd) out.push({ key: "cwd", text: bar.cwd });
  if (bar.branch) {
    let g = bar.branch;
    if (bar.worktree) g += "@" + bar.worktree;
    out.push({ key: "git", text: g });
  }
  if (bar.contextWindow) {
    const win = formatTokens(bar.contextWindow);
    if (bar.contextPercent == null) {
      out.push({ key: "ctx", text: "? / " + win });
    } else {
      const pct = bar.contextPercent;
      const tone = pct > 90 ? "bad" : pct > 70 ? "warn" : "";
      out.push({ key: "ctx", text: pct.toFixed(1) + "% / " + win, tone });
    }
  }
  if (bar.cost > 0) {
    out.push({ key: "cost", text: "$" + bar.cost.toFixed(2) });
  }
  return out;
}

function formatTokens(n) {
  if (n >= 1_000_000) {
    const m = n / 1_000_000;
    return (m % 1 === 0 ? m.toFixed(0) : m.toFixed(1)) + "M";
  }
  if (n >= 1000) return Math.round(n / 1000) + "k";
  return String(n);
}
