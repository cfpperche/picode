import test from "node:test";
import assert from "node:assert/strict";
import { cronError, isValidCron, presetToCron, cronToPreset, describeCron } from "./cron.js";

test("cronError mirrors the Go grammar", () => {
  for (const ok of ["* * * * *", "*/5 * * * *", "0 9 * * 1-5", "30 14 15 3 *", "1,15,30 * * * *", "1-5/2 * * * *", "0 9 * * 7"]) {
    assert.equal(cronError(ok), "", ok);
  }
  for (const bad of ["", "* * * *", "60 * * * *", "* 24 * * *", "* * 0 * *", "* * * 13 *", "* * * * 8", "a * * * *", "*/0 * * * *", "5-1 * * * *", "1,,2 * * * *", "MON * * * *"]) {
    assert.notEqual(cronError(bad), "", bad);
  }
  assert.equal(isValidCron("0 9 * * *"), true);
});

test("presets round-trip through cron", () => {
  assert.equal(presetToCron({ kind: "hourly", time: "00:05" }), "5 * * * *");
  assert.equal(presetToCron({ kind: "daily", time: "09:00" }), "0 9 * * *");
  assert.equal(presetToCron({ kind: "weekdays", time: "18:30" }), "30 18 * * 1-5");
  assert.equal(presetToCron({ kind: "weekly", time: "09:00", dow: 1 }), "0 9 * * 1");
  assert.equal(presetToCron({ kind: "custom", cron: "  */10  * * * * " }), "*/10 * * * *");
  assert.equal(presetToCron({ kind: "daily", time: "25:00" }), "");

  assert.deepEqual(cronToPreset("5 * * * *"), { kind: "hourly", time: "00:05", dow: 1, cron: "5 * * * *" });
  assert.deepEqual(cronToPreset("0 9 * * *"), { kind: "daily", time: "09:00", dow: 1, cron: "0 9 * * *" });
  assert.deepEqual(cronToPreset("30 18 * * 1-5"), { kind: "weekdays", time: "18:30", dow: 1, cron: "30 18 * * 1-5" });
  assert.deepEqual(cronToPreset("0 9 * * 7"), { kind: "weekly", time: "09:00", dow: 0, cron: "0 9 * * 7" });
  assert.equal(cronToPreset("*/10 * * * *").kind, "custom");
  assert.equal(cronToPreset("0 9 1 * *").kind, "custom");
});

test("describeCron speaks plain English", () => {
  assert.equal(describeCron("5 * * * *"), "Hourly at :05");
  assert.equal(describeCron("0 9 * * *"), "Daily at 09:00");
  assert.equal(describeCron("0 9 * * 1-5"), "Weekdays at 09:00");
  assert.equal(describeCron("0 9 * * 1"), "Mondays at 09:00");
  assert.equal(describeCron("*/10 * * * *"), "Custom schedule */10 * * * *");
  assert.equal(describeCron(""), "");
});
