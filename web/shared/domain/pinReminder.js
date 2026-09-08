// Pin reminders on the client (ADR-0100): the picker's presets, what they
// send to PUT /api/pins/{id}/reminder, and the next fire in words for the
// sidebar and the studio. Pure functions; `now` and the zone are injected
// so the rules are testable under node.

const PREFS_KEY = "picode-reminder-prefs";

// The morning hour behind every day-only preset (Slack's 9 a.m., Keep's
// editable Morning). Stored per viewer, "HH:MM".
export function defaultReminderPrefs() {
  return { morning: "09:00", snoozeMin: 60 };
}

export function readReminderPrefs(store) {
  const d = defaultReminderPrefs();
  try {
    const j = JSON.parse((store || localStorage).getItem(PREFS_KEY) || "{}");
    if (/^\d{2}:\d{2}$/.test(j.morning || "")) d.morning = j.morning;
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

function hm(morning) {
  const m = /^(\d{2}):(\d{2})$/.exec(morning || "09:00") || [null, "09", "00"];
  return { h: Number(m[1]), m: Number(m[2]) };
}

// A Date at `morning` on `day` (local calendar of the runtime).
function atMorning(day, morning) {
  const { h, m } = hm(morning);
  const d = new Date(day);
  d.setHours(h, m, 0, 0);
  return d;
}

// Presets, in the order the popover shows them. `once` presets resolve to
// an instant at build time; recurring ones to a cron or an interval.
export const REMINDER_PRESETS = Object.freeze([
  { id: "in1h", label: "In 1 hour", group: "once" },
  { id: "in3h", label: "In 3 hours", group: "once" },
  { id: "tomorrow", label: "Tomorrow morning", group: "once" },
  { id: "nextMonday", label: "Next Monday morning", group: "once" },
  { id: "daily", label: "Every day", group: "repeat" },
  { id: "weekdays", label: "Every weekday", group: "repeat" },
  { id: "everyNh", label: "Every N hours", group: "repeat" },
  { id: "pick", label: "Pick date & time", group: "custom" },
  { id: "cron", label: "Cron (advanced)", group: "custom" },
]);

// buildReminder turns a preset (plus its inputs) into the request body.
//   opts: { now: Date, tz, morning: "HH:MM", hours: N, at: Date|string,
//           cron: "…", fromClose: bool }
export function buildReminder(presetId, opts = {}) {
  const now = opts.now instanceof Date ? opts.now : new Date();
  const tz = opts.tz || browserZone();
  const morning = opts.morning || "09:00";
  const { h, m } = hm(morning);
  switch (presetId) {
    case "in1h":
      return { kind: "once", at: new Date(now.getTime() + 3600_000).toISOString(), tz };
    case "in3h":
      return { kind: "once", at: new Date(now.getTime() + 3 * 3600_000).toISOString(), tz };
    case "tomorrow": {
      const d = atMorning(now, morning);
      d.setDate(d.getDate() + 1);
      return { kind: "once", at: d.toISOString(), tz };
    }
    case "nextMonday": {
      const d = atMorning(now, morning);
      const dow = d.getDay(); // 0 = Sunday
      const ahead = ((8 - dow) % 7) || 7;
      d.setDate(d.getDate() + ahead);
      return { kind: "once", at: d.toISOString(), tz };
    }
    case "daily":
      return { kind: "cron", cron: m + " " + h + " * * *", tz };
    case "weekdays":
      return { kind: "cron", cron: m + " " + h + " * * 1-5", tz };
    case "everyNh": {
      const hours = Math.max(1, Math.min(24 * 366, Math.round(Number(opts.hours) || 24)));
      return { kind: "interval", intervalMin: hours * 60, anchor: opts.fromClose ? "completion" : "schedule", tz };
    }
    case "pick": {
      const at = opts.at instanceof Date ? opts.at : new Date(opts.at || NaN);
      if (Number.isNaN(at.getTime())) return null;
      return { kind: "once", at: at.toISOString(), tz };
    }
    case "cron": {
      const cron = String(opts.cron || "").trim();
      return cron ? { kind: "cron", cron, tz } : null;
    }
  }
  return null;
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
