// Relative timestamps for dense rows. Short units ("2m", "5h", "3d")
// keep a fixed-width meta column readable; past a week the relative form
// stops being informative and a short absolute date takes over (the
// convention Geist documents and Sentry's short unit style).
//
// Callers pass the raw ISO string — never a pre-formatted one — so the
// row can re-render live and expose the absolute time on hover.

const MIN = 60_000;
const HOUR = 60 * MIN;
const DAY = 24 * HOUR;
const WEEK = 7 * DAY;

export function relTime(iso, now = Date.now()) {
  const t = Date.parse(iso || "");
  if (!t) return "";
  const d = now - t;
  if (d < 0) return "now";
  if (d < MIN) return "now";
  if (d < HOUR) return Math.floor(d / MIN) + "m";
  if (d < DAY) return Math.floor(d / HOUR) + "h";
  if (d < WEEK) return Math.floor(d / DAY) + "d";
  const date = new Date(t);
  const sameYear = date.getFullYear() === new Date(now).getFullYear();
  return date.toLocaleDateString(undefined, sameYear
    ? { day: "2-digit", month: "short" }
    : { day: "2-digit", month: "short", year: "numeric" });
}

// absTime is the hover title behind a relative stamp: full local time,
// so the row never hides the fact.
export function absTime(iso) {
  const t = Date.parse(iso || "");
  if (!t) return "";
  return new Date(t).toLocaleString();
}
