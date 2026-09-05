// 5-field cron helpers for the Automations editor. The grammar mirrors
// internal/cron (Go): `*`, single values, `a-b`, `*/n`, `a-b/n`, comma
// lists; Sunday 0 or 7; no names or L/W/?. Presets are the plain-language
// layer over it — the store only ever sees the cron string.

const FIELDS = [
  { name: "minute", min: 0, max: 59 },
  { name: "hour", min: 0, max: 23 },
  { name: "day of month", min: 1, max: 31 },
  { name: "month", min: 1, max: 12 },
  { name: "day of week", min: 0, max: 7 },
];

const DOW = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"];

export const PRESETS = [
  { id: "hourly", label: "Hourly" },
  { id: "daily", label: "Daily" },
  { id: "weekdays", label: "Weekdays" },
  { id: "weekly", label: "Weekly" },
  { id: "custom", label: "Custom" },
];

function num(s, f) {
  if (!/^\d+$/.test(s)) return null;
  const v = parseInt(s, 10);
  if (v < f.min || v > f.max) return null;
  return v;
}

// cronError returns "" when expr is valid, else a one-line reason.
export function cronError(expr) {
  const parts = String(expr || "").trim().split(/\s+/).filter(Boolean);
  if (parts.length !== 5) return "A schedule has 5 fields: minute hour day month weekday.";
  for (let i = 0; i < 5; i++) {
    const f = FIELDS[i];
    for (const item of parts[i].split(",")) {
      if (!item) return "Empty item in " + f.name + ".";
      const [rng, step, extra] = item.split("/");
      if (extra !== undefined) return "Too many slashes in " + f.name + ".";
      if (step !== undefined && (!/^\d+$/.test(step) || parseInt(step, 10) < 1)) return "Bad step in " + f.name + ".";
      if (rng === "*") continue;
      if (rng.includes("-")) {
        const [a, b, more] = rng.split("-");
        const lo = num(a, f);
        const hi = num(b, f);
        if (more !== undefined || lo === null || hi === null) return "Bad range in " + f.name + ".";
        if (lo > hi) return "Range is reversed in " + f.name + ".";
        continue;
      }
      if (num(rng, f) === null) return f.name[0].toUpperCase() + f.name.slice(1) + " must be " + f.min + "-" + f.max + ".";
    }
  }
  return "";
}

export function isValidCron(expr) {
  return cronError(expr) === "";
}

function pad(n) {
  return String(n).padStart(2, "0");
}

function splitTime(time) {
  const m = /^(\d{1,2}):(\d{2})$/.exec(String(time || "").trim());
  if (!m) return null;
  const h = parseInt(m[1], 10);
  const mi = parseInt(m[2], 10);
  if (h > 23 || mi > 59) return null;
  return { h, m: mi };
}

// presetToCron({kind, time, dow, cron}) -> cron string or "" when the
// preset is incomplete (e.g. a bad time).
export function presetToCron(p) {
  const kind = p && p.kind ? p.kind : "daily";
  if (kind === "custom") return String((p && p.cron) || "").trim().split(/\s+/).join(" ");
  const t = splitTime(p && p.time ? p.time : "09:00");
  if (!t) return "";
  switch (kind) {
    case "hourly": return t.m + " * * * *";
    case "daily": return t.m + " " + t.h + " * * *";
    case "weekdays": return t.m + " " + t.h + " * * 1-5";
    case "weekly": {
      const d = Number.isInteger(p.dow) ? p.dow : 1;
      return t.m + " " + t.h + " * * " + d;
    }
    default: return "";
  }
}

// cronToPreset(cron) -> {kind, time, dow, cron}; kind is "custom" when the
// expression is not one of the four shapes the presets produce.
export function cronToPreset(cron) {
  const expr = String(cron || "").trim().split(/\s+/).join(" ");
  const base = { kind: "custom", time: "09:00", dow: 1, cron: expr };
  const parts = expr.split(" ");
  if (parts.length !== 5 || !/^\d+$/.test(parts[0])) return base;
  const m = parseInt(parts[0], 10);
  if (m > 59) return base;
  if (parts[1] === "*" && parts[2] === "*" && parts[3] === "*" && parts[4] === "*") {
    return { ...base, kind: "hourly", time: "00:" + pad(m) };
  }
  if (!/^\d+$/.test(parts[1]) || parts[2] !== "*" || parts[3] !== "*") return base;
  const h = parseInt(parts[1], 10);
  if (h > 23) return base;
  const time = pad(h) + ":" + pad(m);
  if (parts[4] === "*") return { ...base, kind: "daily", time };
  if (parts[4] === "1-5") return { ...base, kind: "weekdays", time };
  if (/^[0-7]$/.test(parts[4])) return { ...base, kind: "weekly", time, dow: parseInt(parts[4], 10) % 7 };
  return base;
}

// describeCron(cron) -> plain words for a row subtitle.
export function describeCron(cron) {
  if (!cron) return "";
  const p = cronToPreset(cron);
  switch (p.kind) {
    case "hourly": return "Hourly at :" + p.time.slice(3);
    case "daily": return "Daily at " + p.time;
    case "weekdays": return "Weekdays at " + p.time;
    case "weekly": return DOW[p.dow] + "s at " + p.time;
    default: return "Custom schedule " + p.cron;
  }
}

export { DOW };
