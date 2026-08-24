export function statusSegments(bar) {
  if (!bar) return [];
  const out = [];
  if (bar.cwd) out.push({ key: "cwd", text: bar.cwd });
  if (bar.branch) {
    let g = bar.branch;
    if (bar.worktree) g += "@" + bar.worktree;
    if (bar.dirty) g += "*";
    out.push({ key: "git", text: g });
  }
  if (bar.contextWindow) {
    const win = formatTokens(bar.contextWindow);
    const pct = bar.contextPercent;
    const unknown = pct == null;
    const tone = unknown ? "" : (pct > 90 ? "bad" : pct > 70 ? "warn" : "ok");
    let text = unknown ? "? / " + win : pct.toFixed(1) + "% · " + win;
    if (bar.autoCompact) text += " (auto)";
    out.push({
      key: "ctx",
      kind: "bar",
      pct: unknown ? 0 : Math.max(0, Math.min(100, pct)),
      text,
      tone,
    });
  }
  const io = [];
  if (bar.input) io.push("↑" + formatTokens(bar.input));
  if (bar.output) io.push("↓" + formatTokens(bar.output));
  if (io.length) out.push({ key: "io", text: io.join(" ") });
  if (bar.cacheHit != null && bar.cacheRead) {
    out.push({ key: "ch", text: "CH" + bar.cacheHit.toFixed(0) + "%" });
  }
  if (bar.cost > 0) {
    out.push({ key: "cost", text: "$" + bar.cost.toFixed(2) });
  }
  if (bar.sessionName) {
    out.push({ key: "name", text: bar.sessionName });
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
