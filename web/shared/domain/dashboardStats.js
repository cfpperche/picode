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

// --- cross-CLI (ADR-0097) ---------------------------------------------

// Signal states the server sends. A metric a CLI never records comes back
// as "not-reported", and the surface must render a dash and a reason for
// it — a 0 there reads as "free" or "clean", which is the one thing the
// cross-CLI dashboard must never say.
export const REPORTED = "reported";
export const PARTIAL = "partial";
export const NOT_REPORTED = "not-reported";
export const UNAVAILABLE = "unavailable";

// measured is true when a value means something. Everything else renders as
// an em dash.
export function measured(state) {
  return state === REPORTED || state === PARTIAL;
}

// coverageOf indexes the coverage rows by CLI so a panel can ask "did this
// CLI report that" without scanning.
export function coverageOf(coverage) {
  const out = new Map();
  for (const row of coverage || []) {
    if (row && row.cli) out.set(row.cli, row);
  }
  return out;
}

// signalState answers for one CLI and signal, defaulting to unavailable
// rather than to a confident "reported" the payload never claimed.
export function signalState(coverage, cli, signal) {
  const row = coverageOf(coverage).get(cli);
  if (!row || !row.signals) return UNAVAILABLE;
  return row.signals[signal] || UNAVAILABLE;
}

// reporters lists the CLIs that actually reported a signal, which is what a
// panel footnote needs to say who its number covers.
export function reporters(coverage, signal) {
  const out = [];
  for (const row of coverage || []) {
    if (row && row.signals && measured(row.signals[signal])) out.push(row);
  }
  return out;
}

// coversAll is true when every CLI that saw activity also reported this
// signal. A panel that is not covered by everyone has to say so.
export function coversAll(coverage, byCli, signal) {
  const active = new Set((byCli || []).filter((c) => c && c.messages > 0).map((c) => c.cli));
  if (active.size === 0) return true;
  for (const row of coverage || []) {
    if (!active.has(row.cli)) continue;
    if (!measured(row.signals && row.signals[signal])) return false;
  }
  return true;
}

// coverageNote is the whole line a panel prints under its number, subject
// included — the caller renders it verbatim rather than gluing a prefix on,
// which is how "Spend covers No CLI that ran records this.." happened.
//
// Empty when everyone covered it: a footnote that always shows is noise,
// one that shows only when something is missing is a fact.
export function coverageNote(coverage, byCli, signal) {
  if (coversAll(coverage, byCli, signal)) return "";
  // Only CLIs that actually ran. Naming one that reported nothing this
  // period read as "OpenCode and Pi only" under a card whose single row was
  // Claude Code.
  const active = new Set((byCli || []).filter((c) => c && c.messages > 0).map((c) => c.cli));
  const names = reporters(coverage, signal)
    .filter((r) => active.size === 0 || active.has(r.cli))
    .map((r) => r.label || r.cli);
  if (names.length === 0) return "No CLI that ran records this.";
  const list = names.length === 1
    ? names[0]
    : names.slice(0, -1).join(", ") + " and " + names[names.length - 1];
  // Subject-free on purpose: the panel's own label is the subject, and any
  // supplied one has to agree with the verb ("These counts covers" was the
  // first attempt).
  return "Covers " + list + " only.";
}

// spendState answers whether a window's cost total means anything.
//
// This exists because of a real reading on this machine: a window whose 260
// messages were all live Claude Code sessions had a true cost of "unknown"
// — none had written a snapshot yet — and both surfaces printed "$0.00".
// A headline that says zero when it means unmeasured is the single most
// damaging thing this dashboard can do, so the number is a dash until some
// active CLI has actually priced something.
export function spendState(byCli) {
  let any = false;
  let all = true;
  for (const c of byCli || []) {
    if (!c || !c.messages) continue;
    if (measured(c.costState)) any = true;
    else all = false;
  }
  if (!any) return NOT_REPORTED;
  return all ? REPORTED : PARTIAL;
}

const BILLING_LABELS = { api: "api", subscription: "sub" };

// billingBadge is the short mark beside a spend row. "unknown" gets no
// badge at all: a row that says nothing is honest, one that says "unknown"
// is chrome carrying a shrug.
export function billingBadge(billing) {
  return BILLING_LABELS[billing] || "";
}

// billingTitle spells the badge out for the row's tooltip.
export function billingTitle(billing) {
  if (billing === "subscription") return "covered by a plan — this is list-price equivalent, not metered spend";
  if (billing === "api") return "metered against a key";
  return "billing mode not set for this CLI";
}

// formatDuration renders agent time. Sessions run concurrently, so these
// totals routinely exceed the wall clock — the caller labels them as agent
// time, and this only has to stay readable up to hundreds of hours.
export function formatDuration(ms) {
  const n = Number(ms);
  if (!Number.isFinite(n) || n <= 0) return "0m";
  const mins = Math.round(n / 60000);
  if (mins < 60) return mins + "m";
  const hours = Math.floor(mins / 60);
  const rest = mins % 60;
  if (hours < 100) return rest ? hours + "h " + rest + "m" : hours + "h";
  return hours.toLocaleString() + "h";
}

// formatLines renders a signed edit count.
export function formatLines(n) {
  const v = Number(n) || 0;
  return v.toLocaleString();
}

// resetsIn is the human gap to a quota window's reset, or "" when the
// payload carries no reset time.
export function resetsIn(resetsAt, now) {
  if (!resetsAt) return "";
  const t = Date.parse(resetsAt);
  if (!Number.isFinite(t)) return "";
  const diff = t - (now || Date.now());
  if (diff <= 0) return "resets now";
  const hours = Math.floor(diff / 3600000);
  if (hours < 1) return "resets in " + Math.max(1, Math.round(diff / 60000)) + "m";
  if (hours < 48) return "resets in " + hours + "h";
  return "resets in " + Math.round(hours / 24) + "d";
}

// costPerTurn and costPerLine are the efficiency levers. Both return null
// rather than 0 when the denominator is missing, so the panel prints a dash
// instead of a number that would read as "free".
export function costPerTurn(cost, turns) {
  if (!turns || !Number.isFinite(cost)) return null;
  return cost / turns;
}

export function costPerLine(cost, linesAdded, linesRemoved) {
  const lines = (Number(linesAdded) || 0) + (Number(linesRemoved) || 0);
  if (!lines || !Number.isFinite(cost)) return null;
  return cost / lines;
}
