import test from "node:test";
import assert from "node:assert/strict";
import { rowFromSchedule, rowsFromAutomation, rowCron, rowsError, rowsToBody, describeSchedules, scheduleLabelOf, MAX_SCHEDULES } from "./automationSchedule.js";

test("rows come from the API list, a draft's cron, or a default", () => {
  const a = { schedules: [{ id: "s1", label: "Morning", cron: "0 9 * * 1-5", tz: "America/Sao_Paulo", enabled: true }, { id: "s2", cron: "0 12 * * 6", tz: "", enabled: false }] };
  const rows = rowsFromAutomation(a);
  assert.equal(rows.length, 2);
  assert.equal(rows[0].preset, "weekdays");
  assert.equal(rows[0].time, "09:00");
  assert.equal(rows[0].tz, "America/Sao_Paulo");
  assert.equal(rows[1].preset, "weekly");
  assert.equal(rows[1].dow, 6);
  assert.equal(rows[1].enabled, false);
  assert.notEqual(rows[0].key, rows[1].key);
  const draft = rowsFromAutomation({ cron: "*/10 * * * *" });
  assert.equal(draft.length, 1);
  assert.equal(draft[0].preset, "custom");
  assert.equal(draft[0].id, "");
  const fresh = rowsFromAutomation(null);
  assert.equal(fresh[0].preset, "weekdays");
  assert.equal(fresh[0].enabled, true);
});

test("a row means one cron and the list refuses what the store refuses", () => {
  const morning = rowFromSchedule({ cron: "0 9 * * 1-5" });
  assert.equal(rowCron(morning), "0 9 * * 1-5");
  assert.equal(rowsError([morning]), "");
  assert.notEqual(rowsError([]), "");
  assert.notEqual(rowsError([{ ...morning, label: "x".repeat(41) }]), "");
  assert.notEqual(rowsError([{ ...morning, time: "" }]), "");
  assert.notEqual(rowsError([{ ...morning, preset: "custom", cron: "* * *" }]), "");
  assert.match(rowsError([morning, { ...rowFromSchedule({ cron: "0  9 * * 1-5" }), label: "Again" }]), /same thing/);
  assert.equal(rowsError([morning, rowFromSchedule({ cron: "0 9 * * 1-5", tz: "Europe/Lisbon" })]), "");
  const many = Array.from({ length: MAX_SCHEDULES + 1 }, (_, i) => rowFromSchedule({ cron: i + " 9 * * *" }));
  assert.match(rowsError(many), /Up to/);
});

test("the body keeps a saved row's zone and stamps new rows with the browser's", () => {
  const saved = rowFromSchedule({ id: "s1", label: " Morning ", cron: "0 9 * * 1-5", tz: "America/Sao_Paulo", enabled: true });
  const added = rowFromSchedule({ cron: "0 23 * * *" });
  const body = rowsToBody([saved, added], "Europe/Lisbon");
  assert.deepEqual(body, [
    { id: "s1", label: "Morning", cron: "0 9 * * 1-5", tz: "America/Sao_Paulo", enabled: true },
    { id: undefined, label: "", cron: "0 23 * * *", tz: "Europe/Lisbon", enabled: true },
  ]);
  assert.equal(rowsToBody([added], "")[0].tz, "");
});

test("the list and the runs table speak plain words", () => {
  const list = [
    { id: "s1", label: "Morning", cron: "0 9 * * 1-5", enabled: true },
    { id: "s2", label: "", cron: "0 12 * * 6", enabled: false },
  ];
  assert.equal(describeSchedules(list), "Morning (Weekdays at 09:00) · Saturdays at 12:00 (off)");
  assert.equal(describeSchedules([{ label: "Noon", cron: "0 12 * * 6", enabled: false }]), "Noon (Saturdays at 12:00, off)");
  assert.equal(describeSchedules([]), "");
  assert.equal(scheduleLabelOf(list, "s1"), "Morning");
  assert.equal(scheduleLabelOf(list, "s2"), "Saturdays at 12:00");
  assert.equal(scheduleLabelOf(list, "gone"), "");
  assert.equal(scheduleLabelOf(list, ""), "");
});
