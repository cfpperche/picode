import { agentsOf, displayAgentName } from "./tree.js";

// deltaPercent: null (not a chip-worthy comparison) when there's no prior
// window (range=all) or the prior total was zero — "vs $0" isn't a
// percentage worth showing, and would divide by zero.
export function deltaPercent(current, prior) {
  if (prior == null || !prior) return null;
  return ((current - prior) / prior) * 100;
}

// fleetStats: agents by live state, across every workspace plus free
// agents. Reuses agentsOf() (lib/tree.js) so this counts exactly the same
// set the sidebar does, including its legacy single-`agent` fallback.
//   running / total  — kept from v1 (mode !== "stopped")
//   working          — streaming right now (App's workingIds)
//   waiting          — blocked on the user (App's waitingId)
//   idle             — running but neither of the above
//   agents           — the running ones, for the tile's name/model line
export function fleetStats(workspaces, freeAgents, live) {
  const workingIds = (live && live.workingIds) || [];
  const waitingId = live && live.waitingId;
  const out = { running: 0, total: 0, working: 0, waiting: 0, idle: 0, agents: [] };
  const count = (agents, ws) => {
    for (const a of agents) {
      out.total++;
      if (!(a && a.mode && a.mode !== "stopped")) continue;
      out.running++;
      let state = "idle";
      if (a.id === waitingId) state = "waiting";
      else if (workingIds.includes(a.id)) state = "working";
      out[state]++;
      out.agents.push({ id: a.id, name: displayAgentName(a, ws), model: a.model || "", provider: a.provider || "", state });
    }
  };
  for (const ws of workspaces || []) count(agentsOf(ws), ws);
  count(freeAgents || [], null);
  return out;
}

const RANGE_LABELS = { today: "Today", "7d": "7 days", "30d": "30 days", all: "All time" };

export function rangeLabel(range) {
  return RANGE_LABELS[range] || RANGE_LABELS["7d"];
}

const COMPARE_LABELS = { today: "vs. yesterday", "7d": "vs. prior 7 days", "30d": "vs. prior 30 days" };

// compareLabel: "" for range=all — there's no prior-of-all-time to compare
// against, so the stat tile renders no delta chip at all for that range.
export function compareLabel(range) {
  return COMPARE_LABELS[range] || "";
}

// formatTokens: 48.2M-style short counts for token totals — the number is
// a magnitude, not an invoice, so three significant figures is honest.
export function formatTokens(n) {
  const v = Number(n) || 0;
  if (v < 1000) return String(v);
  if (v < 1e6) return (v / 1e3).toFixed(v < 1e4 ? 1 : 0) + "K";
  if (v < 1e9) return (v / 1e6).toFixed(1) + "M";
  return (v / 1e9).toFixed(2) + "B";
}

// percent: "0.7%" style, one decimal below 10, none above; null when there
// is no denominator (no turns → no rate, not "0%").
export function percent(part, whole) {
  if (!whole) return null;
  const p = (100 * (part || 0)) / whole;
  return (p < 10 ? p.toFixed(1) : p.toFixed(0)) + "%";
}

// tokenSegments: the stacked bar's four slices as widths in percent of the
// total, in a fixed order so the same slice always sits in the same place.
export function tokenSegments(tokens) {
  const t = tokens || {};
  const parts = [
    { key: "input", label: "input", value: t.input || 0 },
    { key: "output", label: "output", value: t.output || 0 },
    { key: "cacheRead", label: "cache read", value: t.cacheRead || 0 },
    { key: "cacheWrite", label: "cache write", value: t.cacheWrite || 0 },
  ];
  const total = parts.reduce((s, p) => s + p.value, 0);
  return { total, parts: parts.map((p) => ({ ...p, pct: total ? (100 * p.value) / total : 0 })) };
}

// dayLabel: "Aug 25" for a YYYY-MM-DD series key, without letting Date
// parse it as UTC midnight (which would shift it a day in the Americas).
export function dayLabel(ymd) {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(ymd || "");
  if (!m) return ymd || "";
  return new Date(+m[1], +m[2] - 1, +m[3]).toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

// folderLabel: the last path segment of a session cwd. A session file with
// no header line only has pi's encoded folder name ("--home-goat-picode--")
// to go on; that is shown with the fence stripped rather than decoded,
// since "-" is ambiguous in the encoding.
export function folderLabel(cwd) {
  const s = String(cwd || "").replace(/\/+$/, "");
  if (/^--.*--$/.test(s)) return s.slice(2, -2);
  const i = s.lastIndexOf("/");
  return i >= 0 ? s.slice(i + 1) : s;
}
