// Automation schedules (ADR-0045 amendment 2026-09-09): an automation
// has a list of rules, each a cron in a zone with a label and a switch.
// The editor shows one preset row per rule; this module is the form's
// shape, its translation to and from the API, and the plain words the
// list and the detail use. The store only ever sees cron strings.

import { presetToCron, cronToPreset, describeCron, cronError } from "./cron.js";

export const MAX_SCHEDULES = 10;
export const MAX_SCHEDULE_LABEL = 40;

let seq = 0;
function key() {
  seq += 1;
  return "s" + seq;
}

// browserZone() -> the IANA zone the browser runs in, or "" when the
// platform cannot say (the daemon's zone then applies).
export function browserZone() {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || "";
  } catch {
    return "";
  }
}

// rowFromSchedule(schedule) -> one editor row. `schedule` is an API row
// ({id, label, cron, tz, enabled}) or just {cron} for a draft.
export function rowFromSchedule(s, fallbackCron = "0 9 * * 1-5") {
  const p = cronToPreset(s && s.cron ? s.cron : fallbackCron);
  return {
    key: key(),
    id: (s && s.id) || "",
    label: (s && s.label) || "",
    tz: s && typeof s.tz === "string" ? s.tz : "",
    enabled: s && s.enabled === false ? false : true,
    preset: p.kind,
    time: p.time,
    dow: p.dow,
    cron: p.cron,
  };
}

// rowsFromAutomation(a) -> the editor rows for an automation view, a
// draft (`cron` only) or nothing (one default row).
export function rowsFromAutomation(a) {
  if (a && Array.isArray(a.schedules) && a.schedules.length) return a.schedules.map((s) => rowFromSchedule(s));
  if (a && a.cron) return [rowFromSchedule({ cron: a.cron })];
  return [rowFromSchedule(null)];
}

// rowCron(row) -> the cron the row currently means ("" when incomplete).
export function rowCron(row) {
  if (row.preset === "custom") return presetToCron({ kind: "custom", cron: row.cron });
  if (!row.time) return "";
  return presetToCron({ kind: row.preset, time: row.time, dow: row.dow });
}

// rowError(row) -> "" or the reason this row cannot be saved.
export function rowError(row) {
  if (String(row.label || "").trim().length > MAX_SCHEDULE_LABEL) return "A label has up to " + MAX_SCHEDULE_LABEL + " characters.";
  const cron = rowCron(row);
  if (!cron) return "Pick a time.";
  return cronError(cron);
}

// rowsError(rows) -> "" or the first reason the list cannot be saved.
export function rowsError(rows) {
  if (!rows.length) return "Add a schedule or turn the schedule off.";
  if (rows.length > MAX_SCHEDULES) return "Up to " + MAX_SCHEDULES + " schedules per automation.";
  const seen = new Set();
  for (const row of rows) {
    const err = rowError(row);
    if (err) return err;
    const sig = rowCron(row) + " @ " + (row.tz || "");
    if (seen.has(sig)) return "Two schedules say the same thing (" + describeCron(rowCron(row)) + ").";
    seen.add(sig);
  }
  return "";
}

// rowsToBody(rows, zone) -> what the API takes. A row that was saved in
// a zone keeps it; a new row is stamped with the browser's.
export function rowsToBody(rows, zone) {
  return rows.map((row) => ({
    id: row.id || undefined,
    label: String(row.label || "").trim(),
    cron: rowCron(row),
    tz: row.id ? row.tz : (zone || ""),
    enabled: !!row.enabled,
  }));
}

// describeSchedules(list) -> "Weekdays at 09:00 · Saturdays at 12:00 (off)".
// A label replaces the words when it is more telling than the time.
export function describeSchedules(list) {
  return (list || []).map((s) => {
    const words = describeCron(s.cron);
    const off = s.enabled === false;
    if (s.label) return s.label + " (" + words + (off ? ", off" : "") + ")";
    return off ? words + " (off)" : words;
  }).filter(Boolean).join(" · ");
}

// scheduleLabelOf(list, id) -> the words for a run's schedule column.
export function scheduleLabelOf(list, id) {
  if (!id) return "";
  const s = (list || []).find((x) => x.id === id);
  if (!s) return "";
  return s.label || describeCron(s.cron);
}
