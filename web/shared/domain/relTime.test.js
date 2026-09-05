import assert from "node:assert/strict";
import { test } from "node:test";
import { relTime, absTime } from "./relTime.js";

const NOW = Date.parse("2026-09-01T12:00:00Z");
const ago = (ms) => new Date(NOW - ms).toISOString();

test("relTime uses short units up to a week", () => {
  assert.equal(relTime(ago(0), NOW), "now");
  assert.equal(relTime(ago(59_000), NOW), "now");
  assert.equal(relTime(ago(60_000), NOW), "1m");
  assert.equal(relTime(ago(59 * 60_000), NOW), "59m");
  assert.equal(relTime(ago(60 * 60_000), NOW), "1h");
  assert.equal(relTime(ago(23 * 3600_000), NOW), "23h");
  assert.equal(relTime(ago(24 * 3600_000), NOW), "1d");
  assert.equal(relTime(ago(6 * 24 * 3600_000), NOW), "6d");
});

test("relTime falls back to an absolute date past a week", () => {
  const eightDays = relTime(ago(8 * 24 * 3600_000), NOW);
  assert.ok(!/^\d+d$/.test(eightDays), "8 days should not be relative: " + eightDays);
  assert.ok(eightDays.length > 0);
  // A different year keeps the year in the label.
  const old = relTime("2019-03-04T00:00:00Z", NOW);
  assert.ok(/2019/.test(old), "old date should carry its year: " + old);
});

test("relTime is quiet about junk and future stamps", () => {
  assert.equal(relTime("", NOW), "");
  assert.equal(relTime("not a date", NOW), "");
  assert.equal(relTime(null, NOW), "");
  assert.equal(relTime(new Date(NOW + 5000).toISOString(), NOW), "now");
});

test("absTime renders a full local timestamp, or nothing", () => {
  assert.ok(absTime("2026-09-01T12:00:00Z").length > 0);
  assert.equal(absTime(""), "");
  assert.equal(absTime("nope"), "");
});
