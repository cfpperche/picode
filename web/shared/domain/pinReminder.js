// Pin reminders on the client (ADR-0100): the picker's presets, what they
// send to PUT /api/pins/{id}/reminder, and the next fire in words for the
// sidebar and the studio. Pure functions; `now` and the zone are injected
// so the rules are testable under node.

const PREFS_KEY = "picode-reminder-prefs";

// Snooze length behind the card's Snooze. Stored per viewer.
export function defaultReminderPrefs() {
  return { snoozeMin: 60 };
}

export function readReminderPrefs(store) {
  const d = defaultReminderPrefs();
  try {
    const j = JSON.parse((store || localStorage).getItem(PREFS_KEY) || "{}");
    const n = Number(j.snoozeMin);
    if (Number.isFinite(n) && n >= 5 && n <= 24 * 60) d.snoozeMin = n;
  } catch { /* defaults */ }
  return d;
}

export function persistReminderPrefs(prefs, store) {
  const next = { ...defaultReminderPrefs(), ...prefs };
  try { (store || localStorage).setItem(PREFS_KEY, JSON.stringify(next)); } catch { /* ignore */ }
  return next;
}

// The browser's IANA zone; the server evaluates wall-clock rules in it.
export function browserZone() {
  try { return Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC"; } catch { return "UTC"; }
}

export const REMINDER_UNITS = Object.freeze(["hours", "days"]);

// buildReminder turns the picker's form into the request body. Nothing is
// preset: the person names the date and time, or the cadence.
//   form: { mode: "once" | "repeat", at: "YYYY-MM-DDTHH:MM" | Date,
//           every: N, unit: "hours" | "days", time: "HH:MM",
//           fromClose: bool, tz }
//   once            → { kind: "once", at }
//   repeat N hours  → { kind: "interval", intervalMin: N*60, anchor }
//   repeat 1 day    → { kind: "cron", cron: "M H * * *" } (a wall-clock
//                     rule: 09:00 stays 09:00 across DST)
//   repeat N days   → { kind: "interval", intervalMin: N*1440, at: the
//                     next HH:MM, anchor } (a duration: it drifts an
//                     hour across DST, as the picker says)
// Returns { body } or { error } for what the server would refuse.
export function buildReminder(form = {}) {
  const tz = form.tz || browserZone();
  const now = form.now instanceof Date ? form.now : new Date();
  if (form.mode === "once") {
    const at = form.at instanceof Date ? form.at : new Date(form.at || NaN);
    if (Number.isNaN(at.getTime())) return { error: "Pick a date and time." };
    if (at.getTime() <= now.getTime()) return { error: "That time has already passed." };
    return { body: { kind: "once", at: at.toISOString(), tz } };
  }
  if (form.mode === "repeat") {
    const every = Math.round(Number(form.every));
    if (!Number.isFinite(every) || every < 1) return { error: "Repeat every how many?" };
    const anchor = form.fromClose ? "completion" : "schedule";
    if (form.unit === "hours") {
      if (every > 24 * 366) return { error: "At most a year apart." };
      return { body: { kind: "interval", intervalMin: every * 60, anchor, tz } };
    }
    if (form.unit === "days") {
      const m = /^(\d{2}):(\d{2})$/.exec(form.time || "");
      if (!m) return { error: "Pick a time of day." };
      const h = Number(m[1]);
      const mm = Number(m[2]);
      if (every > 366) return { error: "At most a year apart." };
      if (every === 1 && !form.fromClose) return { body: { kind: "cron", cron: mm + " " + h + " * * *", tz } };
      const first = new Date(now);
      first.setHours(h, mm, 0, 0);
      if (first.getTime() <= now.getTime()) first.setDate(first.getDate() + 1);
      return { body: { kind: "interval", intervalMin: every * 24 * 60, anchor, at: first.toISOString(), tz } };
    }
    return { error: "Hours or days." };
  }
  return { error: "Once or repeat." };
}

// formFromReminder fills the picker with the rule it shows, so "edit"
// starts from what is set rather than from blanks.
export function formFromReminder(r) {
  const blank = { mode: "once", at: "", every: 24, unit: "hours", time: "09:00", fromClose: false };
  if (!r) return blank;
  if (r.kind === "once") return { ...blank, mode: "once", at: toLocalInput(r.at || r.nextAt || "") };
  if (r.kind === "cron") {
    const f = String(r.cron || "").split(/\s+/);
    const time = f.length === 5 && /^\d+$/.test(f[0]) && /^\d+$/.test(f[1]) ? pad(Number(f[1])) + ":" + pad(Number(f[0])) : "09:00";
    return { ...blank, mode: "repeat", every: 1, unit: "days", time };
  }
  if (r.kind === "interval") {
    const mins = Number(r.intervalMin) || 60;
    const fromClose = r.anchor === "completion";
    if (mins % (24 * 60) === 0 && (r.at || mins > 24 * 60 || fromClose)) {
      const when = r.at ? new Date(r.at) : null;
      const time = when && !Number.isNaN(when.getTime()) ? pad(when.getHours()) + ":" + pad(when.getMinutes()) : "09:00";
      return { ...blank, mode: "repeat", every: mins / (24 * 60), unit: "days", time, fromClose };
    }
    return { ...blank, mode: "repeat", every: Math.max(1, Math.round(mins / 60)), unit: "hours", fromClose };
  }
  return blank;
}

function sameDay(a, b) {
  return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
}

function pad(n) { return String(n).padStart(2, "0"); }

const DAYS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

// whenNext says the reminder's next fire in words, for a sidebar line:
// "in 45 min", "in 3 h", "today 17:30", "tomorrow 09:00", "Wed 09:00",
// "12 Oct 09:00". Past and null read as what they are.
export function whenNext(nextAt, now = new Date()) {
  if (!nextAt) return "";
  const t = new Date(nextAt);
  if (Number.isNaN(t.getTime())) return "";
  const diff = t.getTime() - now.getTime();
  if (diff < -60_000) return "overdue";
  if (diff < 60_000) return "now";
  if (diff < 3600_000) return "in " + Math.round(diff / 60_000) + " min";
  if (diff < 6 * 3600_000) {
    // Round to whole minutes first, so 2 h 59.9 min reads "in 3 h" and
    // never "in 2 h 60 min".
    const total = Math.round(diff / 60_000);
    const hours = Math.floor(total / 60);
    const mins = total % 60;
    return "in " + hours + " h" + (mins >= 5 ? " " + mins + " min" : "");
  }
  const clock = pad(t.getHours()) + ":" + pad(t.getMinutes());
  if (sameDay(t, now)) return "today " + clock;
  const tomorrow = new Date(now);
  tomorrow.setDate(now.getDate() + 1);
  if (sameDay(t, tomorrow)) return "tomorrow " + clock;
  if (diff < 7 * 24 * 3600_000) return DAYS[t.getDay()] + " " + clock;
  return t.getDate() + " " + t.toLocaleString("en", { month: "short" }) + " " + clock;
}

// The sidebar line: the cadence for repeating rules, the moment for a
// one-shot; a completion-anchored interval with nothing scheduled says so.
export function reminderLine(reminder, now = new Date()) {
  if (!reminder) return "";
  if (!reminder.enabled) return reminder.kind === "once" ? "reminded" : "reminder off";
  if (reminder.kind === "once") return "remind " + whenNext(reminder.nextAt, now);
  if (!reminder.nextAt) return reminder.anchor === "completion" ? reminder.label + " · after you close the last one" : reminder.label;
  return reminder.label + " · next " + whenNext(reminder.nextAt, now);
}

// snoozeUntil: the instant a snooze of `minutes` from now ends, RFC 3339
// at second precision (what the Inbox stores).
export function snoozeUntil(minutes, now = new Date()) {
  const t = new Date(now.getTime() + Math.max(1, Number(minutes) || 60) * 60_000);
  t.setMilliseconds(0);
  return t.toISOString().replace(/\.000Z$/, "Z");
}

// A datetime-local input value for a Date (local wall clock).
export function toLocalInput(d) {
  const t = d instanceof Date ? d : new Date(d);
  if (Number.isNaN(t.getTime())) return "";
  return t.getFullYear() + "-" + pad(t.getMonth() + 1) + "-" + pad(t.getDate()) + "T" + pad(t.getHours()) + ":" + pad(t.getMinutes());
}
