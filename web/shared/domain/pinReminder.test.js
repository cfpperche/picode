import assert from "node:assert/strict";
import { test } from "node:test";
import { buildReminder, formFromReminder, reminderLine, snoozeUntil, toLocalInput, whenNext, readReminderPrefs, persistReminderPrefs } from "./pinReminder.js";

function memStore() {
  const m = new Map();
  return { getItem: (k) => (m.has(k) ? m.get(k) : null), setItem: (k, v) => m.set(k, String(v)) };
}

test("the form builds the request the server accepts, nothing preset", () => {
  const now = new Date(2026, 8, 8, 14, 20, 0); // Tue 8 Sep 2026 14:20 local
  const tz = "America/Sao_Paulo";
  assert.deepEqual(buildReminder({ mode: "once", at: "2026-09-10T10:00", now, tz }), { body: { kind: "once", at: new Date("2026-09-10T10:00").toISOString(), tz } });
  assert.equal(buildReminder({ mode: "once", at: "nope", now, tz }).error, "Pick a date and time.");
  assert.equal(buildReminder({ mode: "once", at: "2026-09-08T10:00", now, tz }).error, "That time has already passed.");
  assert.deepEqual(buildReminder({ mode: "repeat", every: 3, unit: "hours", now, tz }), { body: { kind: "interval", intervalMin: 180, anchor: "schedule", tz } });
  assert.deepEqual(buildReminder({ mode: "repeat", every: "24", unit: "hours", fromClose: true, now, tz }), { body: { kind: "interval", intervalMin: 1440, anchor: "completion", tz } });
  // One day at a time of day is a wall-clock rule.
  assert.deepEqual(buildReminder({ mode: "repeat", every: 1, unit: "days", time: "09:15", now, tz }), { body: { kind: "cron", cron: "15 9 * * *", tz } });
  // N days at a time of day: an interval from the next such time.
  assert.deepEqual(buildReminder({ mode: "repeat", every: 3, unit: "days", time: "09:00", now, tz }), { body: { kind: "interval", intervalMin: 4320, anchor: "schedule", at: new Date(2026, 8, 9, 9, 0, 0).toISOString(), tz } });
  assert.deepEqual(buildReminder({ mode: "repeat", every: 2, unit: "days", time: "18:00", now, tz }).body.at, new Date(2026, 8, 8, 18, 0, 0).toISOString(), "later today counts as the first fire");
  // Every day counted from the close is an interval, not a clock rule.
  assert.equal(buildReminder({ mode: "repeat", every: 1, unit: "days", time: "09:00", fromClose: true, now, tz }).body.kind, "interval");
  assert.equal(buildReminder({ mode: "repeat", every: 0, unit: "hours", now, tz }).error, "Repeat every how many?");
  assert.equal(buildReminder({ mode: "repeat", every: 2, unit: "days", time: "", now, tz }).error, "Pick a time of day.");
  assert.equal(buildReminder({ mode: "repeat", every: 400, unit: "days", time: "09:00", now, tz }).error, "At most a year apart.");
  assert.equal(buildReminder({ mode: "repeat", every: 2, unit: "weeks", now, tz }).error, "Hours or days.");
  assert.equal(buildReminder({ now, tz }).error, "Once or repeat.");
});

test("formFromReminder starts the picker from the rule it shows", () => {
  assert.equal(formFromReminder(null).mode, "once");
  assert.deepEqual(formFromReminder({ kind: "cron", cron: "30 8 * * *" }), { mode: "repeat", at: "", every: 1, unit: "days", time: "08:30", fromClose: false });
  assert.deepEqual(formFromReminder({ kind: "interval", intervalMin: 180, anchor: "schedule" }), { mode: "repeat", at: "", every: 3, unit: "hours", time: "09:00", fromClose: false });
  const threeDays = formFromReminder({ kind: "interval", intervalMin: 4320, anchor: "schedule", at: new Date(2026, 8, 9, 9, 0).toISOString() });
  assert.deepEqual([threeDays.mode, threeDays.every, threeDays.unit, threeDays.time], ["repeat", 3, "days", "09:00"]);
  const daily = formFromReminder({ kind: "interval", intervalMin: 1440, anchor: "completion" });
  assert.deepEqual([daily.every, daily.unit, daily.fromClose], [1, "days", true]);
  const once = formFromReminder({ kind: "once", at: new Date(2026, 8, 10, 10, 0).toISOString() });
  assert.deepEqual([once.mode, once.at], ["once", "2026-09-10T10:00"]);
});

test("whenNext and reminderLine say the next fire in words", () => {
  const now = new Date(2026, 8, 8, 14, 20, 0);
  const min = (n) => new Date(now.getTime() + n * 60_000).toISOString();
  assert.equal(whenNext("", now), "");
  assert.equal(whenNext(min(-5), now), "overdue");
  assert.equal(whenNext(min(0.5), now), "now");
  assert.equal(whenNext(min(45), now), "in 45 min");
  assert.equal(whenNext(min(180), now), "in 3 h");
  assert.equal(whenNext(min(190), now), "in 3 h 10 min");
  assert.equal(whenNext(new Date(now.getTime() + 179.9 * 60_000).toISOString(), now), "in 3 h");
  assert.equal(whenNext(new Date(2026, 8, 8, 22, 5).toISOString(), now), "today 22:05");
  assert.equal(whenNext(new Date(2026, 8, 9, 9, 0).toISOString(), now), "tomorrow 09:00");
  assert.equal(whenNext(new Date(2026, 8, 11, 9, 0).toISOString(), now), "Fri 09:00");
  assert.equal(whenNext(new Date(2026, 9, 12, 9, 0).toISOString(), now), "12 Oct 09:00");
  assert.equal(reminderLine(null), "");
  assert.equal(reminderLine({ kind: "once", enabled: true, nextAt: min(45) }, now), "remind in 45 min");
  assert.equal(reminderLine({ kind: "once", enabled: false }, now), "reminded");
  assert.equal(reminderLine({ kind: "cron", enabled: true, label: "every day at 09:00", nextAt: new Date(2026, 8, 9, 9, 0).toISOString() }, now), "every day at 09:00 · next tomorrow 09:00");
  assert.equal(reminderLine({ kind: "interval", enabled: true, anchor: "completion", label: "every 3 h after you close it", nextAt: null }, now), "every 3 h after you close it · after you close the last one");
  assert.equal(reminderLine({ kind: "interval", enabled: false, label: "every 3 h" }, now), "reminder off");
});

test("snooze, local input and prefs", () => {
  const now = new Date("2026-09-08T14:20:30.500Z");
  assert.equal(snoozeUntil(60, now), "2026-09-08T15:20:30Z");
  assert.equal(snoozeUntil(0, now), "2026-09-08T15:20:30Z", "no minutes means an hour");
  assert.equal(toLocalInput(new Date(2026, 8, 8, 9, 5)), "2026-09-08T09:05");
  assert.equal(toLocalInput("nope"), "");
  const s = memStore();
  assert.deepEqual(readReminderPrefs(s), { snoozeMin: 60 });
  persistReminderPrefs({ snoozeMin: 15 }, s);
  assert.deepEqual(readReminderPrefs(s), { snoozeMin: 15 });
  persistReminderPrefs({ snoozeMin: 1 }, s);
  assert.deepEqual(readReminderPrefs(s), { snoozeMin: 60 }, "bad values fall back");
});
