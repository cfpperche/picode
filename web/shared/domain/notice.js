// The notice model: what the UI must announce, never how it looks.
// Study: docs/benchmarks/2026-09-07-superset-notifications.md.
//
// ADR-0072 keeps React out of web/shared, and that is the right cut here.
// The model and its three policies — how long a notice lives, when it must
// stay silent, and what makes two notices the same one — are shared. The
// card is not: desktop draws a 300px card that steps left of the inspector
// rail, the phone draws a full-width one above the tab bar.
//
//   notice := { level, channel, actor, status, title, body, meta[],
//               actions[], key, target, duration }
//
//   level  ok | info | warn | error | busy
//   channel which announce preference may mute this notice, if any:
//          "finished" | "needsYou" | "reminder". A notice with no channel
//          is feedback for something the user just did and is never muted.
//   actor  who is speaking — { kind, id, name, cli }; cli picks the glyph
//   status the muted phrase beside the name ("finished · worked for 7s")
//   title  the sentence the notice is about
//   meta   footer chips — [{ text, tone: "add" | "del" | "muted" }]
//   action a way out — [{ label, hash, primary }], at most two
//   onClose what the close control means beyond hiding the card — a pin
//          reminder's X marks its Inbox item done (ADR-0100)
//   key    identity: a second notice with the same key REPLACES the first
//   target the surface this notice is about. It is what suppression
//          compares against; the way *out* is always an explicit action,
//          never a clickable div.

import { fmtElapsed, turnDurationMs } from "./turns.js";

export const NOTICE_LEVELS = Object.freeze(["ok", "info", "warn", "error", "busy"]);

// The two announcements a user can turn off (the Notifications preferences
// mirror the push switches: "when an agent needs me", "when a run
// finishes"). Feedback for an action the user just took has no channel and
// cannot be muted — silencing "Saved." would silence the app itself.
export const CHANNELS = Object.freeze(["finished", "needsYou", "reminder"]);

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
    channel: CHANNELS.includes(src.channel) ? src.channel : "",
    actor: normalizeActor(src.actor),
    status: clip(src.status, 60),
    title: clip(src.title, MAX_TITLE),
    body: clip(src.body, MAX_BODY),
    meta: normalizeMeta(src.meta),
    actions: normalizeActions(src.actions),
    key: src.key ? String(src.key) : "",
    target: src.target ? String(src.target) : "",
    onClose: typeof src.onClose === "function" ? src.onClose : null,
    // Infinity is a legitimate value here (a needs-you card outlives the
    // clock), so this cannot be Number.isFinite.
    duration: typeof src.duration === "number" && !Number.isNaN(src.duration) ? src.duration : null,
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
  if (typeof n.duration === "number" && !Number.isNaN(n.duration)) return n.duration;
  // A busy notice is replaced by its own outcome, never by the clock.
  if (n.level === "busy") return Infinity;
  if (n.level === "error" || n.level === "warn") {
    return Math.min(ALERT_CAP_MS, Math.max(ALERT_FLOOR_MS, ms * 3));
  }
  if (n.actions.length) return Math.max(ACTION_FLOOR_MS, ms);
  return ms;
}

// The announce preferences. Muting is not suppression: a suppressed
// notice was redundant right now, a muted one is a class of announcement
// the user turned off.
export function noticeMuted(n, prefs) {
  if (!n || !n.channel) return false;
  const p = prefs || {};
  if (n.channel === "finished") return p.announceFinished === false;
  if (n.channel === "needsYou") return p.announceNeedsYou === false;
  if (n.channel === "reminder") return p.announceReminders === false;
  return false;
}

// Superset's shouldSuppressForVisiblePane, at our granularity: never
// announce what the user is already looking at. An error is the exception
// — it is the only feedback some failures have, and it reports something
// the user just did rather than something on screen. A needs-you for the
// conversation already showing its ask card IS redundant, so `warn` is
// suppressed like the rest.
export function suppressNotice(n, surface) {
  if (!n || !n.target) return false;
  if (n.level === "error") return false;
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
    channel: "finished",
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

// An agent stopped and is waiting on a person (ADR-0044's live dialog, as
// `needsYou` already shapes it for the phone's home queue). Unlike every
// other notice this one is sticky: it is bounded by the number of agents,
// deduplicated per agent, and the caller dismisses it the moment the
// dialog is gone — so it lives exactly as long as the question does.
// One card per question, not per agent: when an agent answers one dialog
// and raises the next, the id changes, so the old card is withdrawn and
// the new question announced instead of the card keeping stale text.
export function askNoticeKey(entry) {
  const e = entry || {};
  if (!e.agentId) return "";
  return e.dialogId ? "ask:" + e.agentId + ":" + e.dialogId : "ask:" + e.agentId;
}

export function needsYouNotice(entry, target) {
  const e = entry || {};
  const where = e.where ? " · " + e.where : "";
  return normalizeNotice({
    level: "warn",
    channel: "needsYou",
    actor: { kind: "agent", id: e.agentId || "", name: e.agentName || "Agent", cli: e.cli || "pi" },
    status: "needs you" + where,
    title: e.title || "Waiting for your answer.",
    body: e.message || "",
    actions: target ? [{ label: "Answer", hash: target, primary: true }] : [],
    key: askNoticeKey(e),
    target: target || "",
    duration: Infinity,
  });
}

// A standing needs-you card is the one notice that outlives the clock, so
// suppression has to keep working after it is on screen: these are the
// asks the user is now looking at, and their cards should go.
export function asksOnSurface(entries, surface, target) {
  if (!surface || !target) return [];
  const out = [];
  for (const e of entries || []) {
    if (!e || e.kind !== "ask" || !e.agentId) continue;
    if (target(e.agentId) === surface) out.push(askNoticeKey(e));
  }
  return out;
}

// What changed since the last pass over the needs-you queue: which asks
// are new (announce them) and which are gone (dismiss their card). The
// caller owns the routes, so it supplies `target(agentId)`.
// `announced` is null the first time a shell looks at the queue: a page
// load must not toast a backlog the sidebar and the badge already show,
// so the first pass only records what is waiting. Superset's rule that
// status is derived and only the user's own marks are kept.
export function needsYouPlan(entries, announced, target) {
  const first = announced == null;
  const seen = new Set(announced || []);
  const keys = new Set();
  const fresh = [];
  for (const e of entries || []) {
    if (!e || e.kind !== "ask" || !e.agentId) continue;
    const key = askNoticeKey(e);
    keys.add(key);
    if (!first && !seen.has(key)) fresh.push(needsYouNotice(e, target ? target(e.agentId) : ""));
  }
  const gone = first ? [] : [...seen].filter((k) => !keys.has(k));
  return { fresh, gone, keys };
}

// A pin reminder fired (ADR-0100). Sticky like needs-you: it lives until
// the person closes it (X → the Inbox item goes done, on every device) or
// snoozes it; the card is a projection of the Inbox row, never the state.
// `fire` is the pin.reminded payload or an open Inbox item mapped to it:
// { inboxId, pinId, title, label, catchUp, body }.
export const REMINDER_KEY = "reminder:";
export const REMINDERS_COLLAPSED_KEY = "reminders:all";
export const REMINDER_CARD_CAP = 3;

export function reminderNoticeKey(fire) {
  return fire && fire.inboxId ? REMINDER_KEY + fire.inboxId : "";
}

export function reminderNotice(fire, { pinHash, snooze, close } = {}) {
  const f = fire || {};
  const status = "reminder" + (f.label ? " · " + f.label : "") + (f.catchUp ? " · was due earlier" : "");
  const actions = [];
  if (typeof snooze === "function") actions.push({ label: "Snooze", run: snooze });
  if (pinHash) actions.push({ label: "Open", hash: pinHash, primary: true });
  return normalizeNotice({
    level: "info",
    channel: "reminder",
    actor: { kind: "pin", id: f.pinId || "", name: f.title || "Pin" },
    status,
    title: f.body || f.title || "Reminder",
    actions,
    key: reminderNoticeKey(f),
    // Never suppressed by the visible surface: the person scheduled it.
    target: "",
    duration: Infinity,
    onClose: typeof close === "function" ? close : null,
  });
}

// Above the cap, one card stands for all of them: sonner keeps only
// `visibleToasts` on screen and queues the rest, and a wall of sticky
// cards would hide the app. The Inbox is the list.
export function remindersCollapsedNotice(count, inboxHash) {
  return normalizeNotice({
    level: "info",
    channel: "reminder",
    actor: { kind: "pin", id: "", name: "Reminders" },
    status: count + " waiting",
    title: count + " reminders are waiting for you.",
    actions: inboxHash ? [{ label: "Open Inbox", hash: inboxHash, primary: true }] : [],
    key: REMINDERS_COLLAPSED_KEY,
    target: "",
    duration: Infinity,
  });
}

// An open reminder item (GET /api/inbox?kind=reminder) as a fire.
export function fireFromInboxItem(it) {
  const i = it || {};
  return { inboxId: i.id || "", pinId: i.sourceId || "", title: i.title || "", label: i.reason || "", body: (i.body || "").split("\n")[0], catchUp: /Was due /.test(i.body || "") };
}

// reminderPlan: given the open (not done, not snoozed) reminder items and
// the keys currently shown, decide what to raise and what to withdraw.
// Unlike needsYouPlan there is no silent first pass: a reminder owed on
// page load is shown — that is the point of it.
//   returns { show: [notice…], hide: [key…], keys: Set }
export function reminderPlan(items, shownKeys, { cap = REMINDER_CARD_CAP, notice, collapsed } = {}) {
  const open = (items || []).filter((it) => it && it.id && it.state !== "done");
  const shown = new Set(shownKeys || []);
  const keys = new Set();
  const show = [];
  const hide = [];
  if (open.length > cap) {
    keys.add(REMINDERS_COLLAPSED_KEY);
    if (collapsed) show.push(collapsed(open.length));
  } else {
    for (const it of open) {
      const key = REMINDER_KEY + it.id;
      keys.add(key);
      if (notice) show.push(notice(fireFromInboxItem(it)));
    }
  }
  for (const k of shown) if (!keys.has(k)) hide.push(k);
  return { show, hide, keys };
}
