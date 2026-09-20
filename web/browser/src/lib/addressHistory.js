// The address bar's suggestions: what you visited, typed URLs first.
//
// One reader, two panes. The desktop pane (the real WebView2 tab) offers every
// visit; the pane with no desktop shell can only open this machine's servers,
// so it asks for `localOnly` — the same list, filtered to what that pane can
// actually open (a row that could not load is worse than a missing row).
//
// The ranking is the spec's own sentence ("typed URLs first"): a URL the human
// typed outranks a URL they only landed on, and recency orders everything
// else. The server returns newest-first and this module keeps that order
// inside each group, so nothing here invents a relevance score.

import { isLoopbackUrl } from "@picode/shared/client/devservers.js";

// How many rows the dropdown shows before it stops being a glance.
export const SUGGEST_LIMIT = 6;

// splitAddress is what a row looks like: the host loud, the rest quiet. An
// unparseable URL is not worth a row of its own honesty — show it as it is.
export function splitAddress(raw) {
  const s = String(raw || "").trim();
  try {
    const u = new URL(s);
    const rest = (u.pathname === "/" ? "" : u.pathname) + u.search + u.hash;
    return { host: u.host, rest };
  } catch {
    return { host: s, rest: "" };
  }
}

// sameAddress ignores the one difference that is never the point: a trailing
// slash. `https://x/` and `https://x` are the same page to a human.
function sameAddress(a, b) {
  const norm = (v) => String(v || "").trim().replace(/\/+$/, "").toLowerCase();
  return norm(a) === norm(b) && norm(a) !== "";
}

// rankVisits folds the store's rows into the list the dropdown renders.
export function rankVisits(visits, { query = "", limit = SUGGEST_LIMIT, skipUrl = "", localOnly = false } = {}) {
  const q = String(query || "").trim().toLowerCase();
  const cap = Number.isFinite(Number(limit)) && Number(limit) > 0 ? Math.floor(Number(limit)) : SUGGEST_LIMIT;
  const seen = new Set();
  const typed = [];
  const rest = [];
  for (const raw of Array.isArray(visits) ? visits : []) {
    if (!raw || typeof raw !== "object") continue;
    const url = typeof raw.url === "string" ? raw.url.trim() : "";
    if (!url || seen.has(url.replace(/\/+$/, "").toLowerCase())) continue;
    if (skipUrl && sameAddress(url, skipUrl)) continue;
    if (localOnly && !isLoopbackUrl(url)) continue;
    const title = typeof raw.title === "string" ? raw.title : "";
    const host = typeof raw.host === "string" && raw.host ? raw.host : splitAddress(url).host;
    if (q && !(`${url} ${title} ${host}`.toLowerCase().includes(q))) continue;
    seen.add(url.replace(/\/+$/, "").toLowerCase());
    const row = { id: raw.id, url, title, host, typed: raw.typed === true };
    (row.typed ? typed : rest).push(row);
  }
  return [...typed, ...rest].slice(0, cap);
}
