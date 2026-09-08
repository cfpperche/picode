// The notice model: what the UI must announce, never how it looks.
// Study: docs/benchmarks/2026-09-07-superset-notifications.md.
//
// ADR-0072 keeps React out of web/shared, and that is the right cut here.
// The model and its three policies — how long a notice lives, when it must
// stay silent, and what makes two notices the same one — are shared. The
// card is not: desktop draws a 300px card that steps left of the inspector
// rail, the phone draws a full-width one above the tab bar.
//
//   notice := { level, actor, status, title, body, meta[], actions[],
//               key, target, duration }
//
//   level  ok | info | warn | error | busy
//   actor  who is speaking — { kind, id, name, cli }; cli picks the glyph
//   status the muted phrase beside the name ("finished · worked for 7s")
//   title  the sentence the notice is about
//   meta   footer chips — [{ text, tone: "add" | "del" | "muted" }]
//   action a way out — [{ label, hash, primary }], at most two
//   key    identity: a second notice with the same key REPLACES the first
//   target the surface this notice is about. It is what suppression
//          compares against; the way *out* is always an explicit action,
//          never a clickable div.

import { fmtElapsed, turnDurationMs } from "./turns.js";

export const NOTICE_LEVELS = Object.freeze(["ok", "info", "warn", "error", "busy"]);

const KIND_LEVEL = Object.freeze({
  ok: "ok",
  info: "info",
  warn: "warn",
  err: "error",
  error: "error",
});

const MAX_META = 4;
const MAX_ACTIONS = 2;
const MAX_TITLE = 400;
const MAX_BODY = 2000;

// error/warn outlive the confirmation of a click, because the user was
// probably not looking when they fired. They are NOT sticky: sonner holds
// everything past `visibleToasts` in a queue that only drains as toasts
// expire, so an unbounded class of notice would wall the screen off.
const ALERT_FLOOR_MS = 12000;
const ALERT_CAP_MS = 30000;
// A notice with a way out needs long enough to read the way out.
const ACTION_FLOOR_MS = 8000;

function clip(text, max) {
  const s = String(text == null ? "" : text);
  return s.length > max ? s.slice(0, max - 1) + "…" : s;
}

function normalizeActor(actor) {
  if (!actor || typeof actor !== "object") return null;
  const name = clip(actor.name, 80);
  if (!name) return null;
  return {
    kind: actor.kind || "agent",
    id: actor.id || "",
    name,
    cli: actor.cli || "",
  };
}

function normalizeMeta(meta) {
  if (!Array.isArray(meta)) return [];
  const out = [];
  for (const chip of meta) {
    if (!chip) continue;
    const text = clip(chip.text, 40);
    if (!text) continue;
    out.push({ text, tone: chip.tone === "add" || chip.tone === "del" ? chip.tone : "muted" });
    if (out.length === MAX_META) break;
  }
  return out;
}

// Two actions is the budget the workspace card header settled on
// (docs/benchmarks/2026-09-07-workspace-card-toolbar.md); a toast has less
// room than a header, not more.
function normalizeActions(actions) {
  if (!Array.isArray(actions)) return [];
  const out = [];
  for (const a of actions) {
    if (!a) continue;
    const label = clip(a.label, 24);
    if (!label || (!a.hash && typeof a.run !== "function")) continue;
    out.push({ label, hash: a.hash || "", run: typeof a.run === "function" ? a.run : null, primary: !!a.primary });
    if (out.length === MAX_ACTIONS) break;
  }
  return out;
}

export function normalizeNotice(n) {
  const src = n && typeof n === "object" ? n : {};
  const level = NOTICE_LEVELS.includes(src.level) ? src.level : "info";
  return {
    level,
    actor: normalizeActor(src.actor),
    status: clip(src.status, 60),
    title: clip(src.title, MAX_TITLE),
    body: clip(src.body, MAX_BODY),
    meta: normalizeMeta(src.meta),
    actions: normalizeActions(src.actions),
    key: src.key ? String(src.key) : "",
    target: src.target ? String(src.target) : "",
    duration: Number.isFinite(src.duration) ? src.duration : null,
  };
}

// The 317 legacy call sites: one string, one kind, nothing else. They stay
// one line and gain only the level glyph.
export function plainNotice(text, kind) {
  return normalizeNotice({ level: KIND_LEVEL[kind] || "error", title: text });
}

export function noticeKey(n) {
  return (n && n.key) || "";
}

// How long the notice lives. `base` is the user's Duration preference.
export function noticeDuration(n, base = 4000) {
  const ms = Math.max(1000, Number(base) || 4000);
  if (!n) return ms;
  if (Number.isFinite(n.duration)) return n.duration;
  // A busy notice is replaced by its own outcome, never by the clock.
  if (n.level === "busy") return Infinity;
  if (n.level === "error" || n.level === "warn") {
    return Math.min(ALERT_CAP_MS, Math.max(ALERT_FLOOR_MS, ms * 3));
  }
  if (n.actions.length) return Math.max(ACTION_FLOOR_MS, ms);
  return ms;
}

// Superset's shouldSuppressForVisiblePane, at our granularity: never
// announce what the user is already looking at. An alert is never
// suppressed — an error on the visible surface is still news, and it is
// the only feedback some failures have.
export function suppressNotice(n, surface) {
  if (!n || !n.target) return false;
  if (n.level === "error" || n.level === "warn") return false;
  if (!surface || !surface.focused) return false;
  return surface.target === n.target;
}

// Per-file totals for one turn. `change` rides every edit/write tool item
// already (web/shared/domain/diff.js), so the browser can count lines
// without asking the server anything.
export function changeTotals(turn) {
  const byPath = new Map();
  for (const it of (turn && turn.work) || []) {
    const c = it && it.change;
    if (!c || !c.path) continue;
    const prev = byPath.get(c.path) || { add: 0, del: 0 };
    byPath.set(c.path, { add: prev.add + (Number(c.add) || 0), del: prev.del + (Number(c.del) || 0) });
  }
  let add = 0;
  let del = 0;
  for (const v of byPath.values()) {
    add += v.add;
    del += v.del;
  }
  return { files: byPath.size, add, del };
}

export function changeMeta(totals) {
  const t = totals || { files: 0, add: 0, del: 0 };
  if (!t.files) return [];
  const meta = [{ text: t.files === 1 ? "1 file" : t.files + " files", tone: "muted" }];
  if (t.add) meta.push({ text: "+" + t.add, tone: "add" });
  if (t.del) meta.push({ text: "−" + t.del, tone: "del" });
  return meta;
}

export function finishStatus(ms) {
  const n = Number(ms) || 0;
  // Under half a second there is no duration worth claiming.
  return n < 400 ? "finished" : "finished · worked for " + fmtElapsed(n);
}

// The first line of prose in a markdown reply: fenced code is skipped
// whole (its contents are not a sentence), and so are headings and rules.
function firstProseLine(text) {
  let fenced = false;
  for (const raw of String(text || "").split("\n")) {
    const line = raw.trim();
    if (line.startsWith("```") || line.startsWith("~~~")) {
      fenced = !fenced;
      continue;
    }
    if (fenced || !line) continue;
    if (line.startsWith("#") || /^[-*_=]{3,}$/.test(line)) continue;
    return line;
  }
  return "";
}

// The last thing the agent actually said, as the notice's sentence. Not a
// summary we invented: the honesty bar says the chrome reports what
// happened, and an assistant reply is the only sentence we have.
export function lastReplyLine(turn) {
  const replies = (turn && turn.replies) || [];
  for (let i = replies.length - 1; i >= 0; i--) {
    const it = replies[i];
    if (!it || it.kind !== "block" || it.cls === "thinking") continue;
    const line = firstProseLine(it.text);
    if (line) return line;
  }
  return "";
}

// The card from the study, built from what the browser already holds at
// agent_settled: the turn's own timestamps, its edit tools' line counts,
// and the agent's last sentence.
export function agentFinishNotice({ agent, turn, target, title }) {
  const a = agent || {};
  const totals = changeTotals(turn);
  const said = title || lastReplyLine(turn);
  return normalizeNotice({
    level: "ok",
    actor: { kind: "agent", id: a.id || "", name: a.name || "Agent", cli: a.cli || "pi" },
    status: finishStatus(turn ? turnDurationMs(turn) : 0),
    title: said || "Finished this turn.",
    meta: changeMeta(totals),
    // A ghost pill, not a filled one: the card arrives unbidden, so its
    // way out should read as an offer rather than a demand.
    actions: target ? [{ label: "Open", hash: target }] : [],
    key: a.id ? "agent:" + a.id : "",
    target: target || "",
  });
}
