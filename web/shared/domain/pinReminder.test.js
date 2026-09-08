import assert from "node:assert/strict";
import { test } from "node:test";
import { buildReminder, reminderLine, snoozeUntil, toLocalInput, whenNext, readReminderPrefs, persistReminderPrefs } from "./pinReminder.js";

function memStore() {
  const m = new Map();
  return { getItem: (k) => (m.has(k) ? m.get(k) : null), setItem: (k, v) => m.set(k, String(v)) };
}

test("presets build the request the server accepts", () => {
  const now = new Date(2026, 8, 8, 14, 20, 0); // Tue 8 Sep 2026 14:20 local
  const tz = "America/Sao_Paulo";
  assert.deepEqual(buildReminder("in1h", { now, tz }), { kind: "once", at: new Date(2026, 8, 8, 15, 20, 0).toISOString(), tz });
  assert.deepEqual(buildReminder("in3h", { now, tz }), { kind: "once", at: new Date(2026, 8, 8, 17, 20, 0).toISOString(), tz });
  assert.deepEqual(buildReminder("tomorrow", { now, tz, morning: "08:30" }), { kind: "once", at: new Date(2026, 8, 9, 8, 30, 0).toISOString(), tz });
  assert.deepEqual(buildReminder("nextMonday", { now, tz }), { kind: "once", at: new Date(2026, 8, 14, 9, 0, 0).toISOString(), tz });
  // From a Monday, next Monday is a week away, not today.
  assert.deepEqual(buildReminder("nextMonday", { now: new Date(2026, 8, 14, 7, 0, 0), tz }), { kind: "once", at: new Date(2026, 8, 21, 9, 0, 0).toISOString(), tz });
  assert.deepEqual(buildReminder("daily", { now, tz, morning: "09:15" }), { kind: "cron", cron: "15 9 * * *", tz });
  assert.deepEqual(buildReminder("weekdays", { now, tz }), { kind: "cron", cron: "0 9 * * 1-5", tz });
  assert.deepEqual(buildReminder("everyNh", { now, tz, hours: 24 }), { kind: "interval", intervalMin: 1440, anchor: "schedule", tz });
  assert.deepEqual(buildReminder("everyNh", { now, tz, hours: 3, fromClose: true }), { kind: "interval", intervalMin: 180, anchor: "completion", tz });
  assert.deepEqual(buildReminder("pick", { now, tz, at: "2026-09-10T10:00" }), { kind: "once", at: new Date("2026-09-10T10:00").toISOString(), tz });
  assert.equal(buildReminder("pick", { now, tz, at: "nope" }), null);
  assert.deepEqual(buildReminder("cron", { tz, cron: " 0 18 * * 5 " }), { kind: "cron", cron: "0 18 * * 5", tz });
  assert.equal(buildReminder("cron", { tz, cron: "" }), null);
  assert.equal(buildReminder("nope", { tz }), null);
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
  assert.equal(whenNext(min(124), now), "in 2 h 4 min".replace(" 4 min", ""), "under five minutes is dropped");
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
  assert.deepEqual(readReminderPrefs(s), { morning: "09:00", snoozeMin: 60 });
  persistReminderPrefs({ morning: "08:30", snoozeMin: 15 }, s);
  assert.deepEqual(readReminderPrefs(s), { morning: "08:30", snoozeMin: 15 });
  persistReminderPrefs({ morning: "8h", snoozeMin: 1 }, s);
  assert.deepEqual(readReminderPrefs(s), { morning: "09:00", snoozeMin: 60 }, "bad values fall back");
});
