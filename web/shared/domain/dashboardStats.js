import { agentsOf, displayAgentName } from "./tree.js";
import { agentRowStatus } from "./agentStatus.js";
import { terminalDisplayCli, terminalStatus } from "./terminalCli.js";

// deltaPercent: null (not a chip-worthy comparison) when there's no prior
// window (range=all) or the prior total was zero — "vs $0" isn't a
// percentage worth showing, and would divide by zero.
export function deltaPercent(current, prior) {
  if (prior == null || !prior) return null;
  return ((current - prior) / prior) * 100;
}

// The four live buckets, in the order a supervisor reads them: who is stuck
// on me, who is busy, who is ready, who cannot say.
//
// "unreported" is the reason this shape exists rather than a plain running
// count. A CLI running in a terminal reports activity through its own hooks
// (ADR-0062), and one whose hooks were never wired has presence but no
// signal. Folding that into "idle" would be the same lie as printing $0.00
// for spend no CLI priced (ADR-0097): the surface would claim a quiet agent
// where it only has a blind one.
export const FLEET_WORKING = "working";
export const FLEET_NEEDS_YOU = "needs-you";
export const FLEET_IDLE = "idle";
export const FLEET_UNREPORTED = "unreported";

// The strip order, and the words the tile prints.
export const FLEET_ORDER = [FLEET_NEEDS_YOU, FLEET_WORKING, FLEET_IDLE, FLEET_UNREPORTED];
export const FLEET_LABELS = Object.freeze({
  [FLEET_NEEDS_YOU]: "need you",
  [FLEET_WORKING]: "working",
  [FLEET_IDLE]: "idle",
  [FLEET_UNREPORTED]: "no signal",
});

// FLEET_HINTS is the one line behind each bucket's word, for a reader who has
// never wired a CLI's hooks and would read "no signal" as a broken terminal.
// Same vocabulary as the Agent CLIs page ("Activity not reported").
export const FLEET_HINTS = Object.freeze({
  [FLEET_NEEDS_YOU]: "Blocked on you — answer it to let it continue",
  [FLEET_WORKING]: "Running a turn right now",
  [FLEET_IDLE]: "Open, nothing running",
  [FLEET_UNREPORTED]: "Running, but it has not reported activity — the CLI's hooks are not wired",
});

// agentState / terminalState fold each kind's own vocabulary (agentStatus.js,
// terminalCli.js) into the four buckets. "stopped" maps to "" — a stopped
// agent or a dead session is not a fleet member, it is a row in the store.
function agentState(ag, live) {
  const status = agentRowStatus(ag, live);
  if (status === "needs-you") return FLEET_NEEDS_YOU;
  if (status === "working") return FLEET_WORKING;
  if (status === "stopped") return "";
  // "interactive" is a TUI sitting open with no turn running: ready, the
  // same as a managed agent that is neither streaming nor blocked.
  return FLEET_IDLE;
}

function terminalState(term) {
  const status = terminalStatus(term);
  if (status === "needs-you") return FLEET_NEEDS_YOU;
  if (status === "working") return FLEET_WORKING;
  if (status === "ready") return FLEET_IDLE;
  if (status === "open") return FLEET_UNREPORTED;
  return ""; // stopped
}

function emptyCounts() {
  return {
    total: 0,
    running: 0,
    [FLEET_WORKING]: 0,
    [FLEET_NEEDS_YOU]: 0,
    [FLEET_IDLE]: 0,
    [FLEET_UNREPORTED]: 0,
  };
}

// fleetStats: what is alive on this machine right now — every managed agent
// (workspace and free), every agent-CLI terminal, and plain shell terminals
// counted apart because they are not agents.
//
// The dashboard measures the machine (ADR-0127), and the windowed tiles have
// always done so: Spend and By CLI read every agent CLI's own session store.
// This tile was the exception — it counted managed agents only, so a sidebar
// of seven live Claude Code and Grok terminals sat beside "1 / 1 running".
// Terminals are the fleet's other half (ADR-0056 hosts CLI agents in them,
// ADR-0062 gives them presence and activity), so they count here.
//
//   agents.running / terminals.running / shells.running — live, per kind
//   live    — agents.running + terminals.running, the tile's headline
//   units   — one row per live agent and CLI terminal, attention first
//
export function fleetStats(workspaces, freeAgents, terminals, live) {
  const out = {
    agents: emptyCounts(),
    terminals: emptyCounts(),
    shells: emptyCounts(),
    live: 0,
    [FLEET_WORKING]: 0,
    [FLEET_NEEDS_YOU]: 0,
    [FLEET_IDLE]: 0,
    [FLEET_UNREPORTED]: 0,
    units: [],
  };

  const addAgent = (ag, ws) => {
    if (!ag) return;
    out.agents.total++;
    const state = agentState(ag, live);
    if (!state) return;
    out.agents.running++;
    out.agents[state]++;
    out[state]++;
    out.units.push({
      key: "agent:" + ag.id,
      kind: "agent",
      id: ag.id,
      name: displayAgentName(ag, ws),
      detail: ag.model || "",
      state,
    });
  };

  for (const ws of workspaces || []) for (const ag of agentsOf(ws)) addAgent(ag, ws);
  for (const ag of freeAgents || []) addAgent(ag, null);

  for (const term of terminals || []) {
    if (!term) continue;
    const cli = terminalDisplayCli(term);
    const bucket = cli ? out.terminals : out.shells;
    bucket.total++;
    const state = terminalState(term);
    if (!state) continue;
    bucket.running++;
    bucket[state]++;
    if (!cli) continue; // a shell is counted, never listed as an agent
    out[state]++;
    out.units.push({
      key: "terminal:" + term.id,
      kind: "terminal",
      id: term.id,
      name: term.name || term.id,
      detail: cli,
      state,
    });
  }

  out.live = out.agents.running + out.terminals.running;
  const rank = (u) => FLEET_ORDER.indexOf(u.state);
  out.units.sort((a, b) => rank(a) - rank(b) || String(a.name).localeCompare(String(b.name), undefined, { sensitivity: "base" }));
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

// bucketLabel: the axis and tooltip label for one series key. The server
// sends calendar days ("Aug 25") and, when the window is a single day, hours
// of that day ("14:00") — the key states its own granularity, so this reads it
// instead of being told. Day keys are never fed to Date as UTC midnight, which
// would shift them a day in the Americas.
export function bucketLabel(key) {
  const s = String(key || "");
  const hour = /^(\d{4})-(\d{2})-(\d{2})T(\d{2})$/.exec(s);
  if (hour) return hour[4] + ":00";
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(s);
  if (!m) return s;
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

// limitReading describes what a quota snapshot can still claim. A quota
// percentage is a *reading of a window*, not a sum over a period: Codex
// writes it on an event, and the window it describes ends at resetsAt. Once
// that instant is past, the number belongs to a window that no longer exists
// — showing it beside "resets in 6d" would present a stale figure as the
// current state, which is the failure ADR-0097 refuses for cost.
//   "current" — read inside the window it describes
//   "expired" — the window it describes has already reset
//   "unknown" — the payload carries no reset time to judge by
export function limitReading(limit, now) {
  const reset = Date.parse((limit && limit.resetsAt) || "");
  if (!Number.isFinite(reset)) return "unknown";
  return reset <= (now || Date.now()) ? "expired" : "current";
}

// observedAge is how long ago the CLI wrote this reading, in milliseconds, or
// null when the payload carries no observation time.
export function observedAge(limit, now) {
  const at = Date.parse((limit && limit.observedAt) || "");
  if (!Number.isFinite(at)) return null;
  return Math.max(0, (now || Date.now()) - at);
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
