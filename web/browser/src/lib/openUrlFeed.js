// openUrlFeed.js — a CLI's "open in the browser" (OAuth logins above all)
// arrives as a terminal.open_url feed event (ADR-0180) instead of launching
// a chromium inside WSL. The client that is actually on screen answers it
// through the same path as a Ctrl+click, so the human's link destination
// preference decides: the desktop app opens its own tab, a plain browser
// opens itself. A hidden client stays out — when no client is visible the
// server already opened the host's default browser. Dedupe guards the rare
// burst (a CLI retrying its open) without queueing stale logins: the event
// is ephemeral, so nothing replays on reconnect.

export const OPEN_URL_EVENT = "terminal.open_url";
export const OPEN_URL_DEDUPE_MS = 1500;

// openUrlAllowed decides whether key may open at time at; seen persists
// across calls (the caller's Map). The map is swept lazily: nothing here
// may allocate timers.
export function openUrlAllowed(seen, key, at, dedupeMs = OPEN_URL_DEDUPE_MS) {
  const last = seen.get(key);
  if (last !== undefined && at - last < dedupeMs) return false;
  seen.set(key, at);
  if (seen.size > 64) {
    for (const [k, t] of seen) if (at - t > dedupeMs * 2) seen.delete(k);
  }
  return true;
}

// installOpenUrlFeed subscribes to the feed and calls open(url) whenever a
// terminal.open_url lands for a visible client. opts:
//   subscribe(fn) -> unsubscribe    the feed client's subscribeFeed
//   open(url)                        the destination (App's openTermLink)
//   hidden() -> bool                 client visibility probe (document)
//   now()    -> ms                   injectable for tests
// Returns the unsubscribe, or undefined when subscribe is unavailable.
export function installOpenUrlFeed({ subscribe, open, hidden, now } = {}) {
  if (typeof subscribe !== "function" || typeof open !== "function") return undefined;
  const clock = typeof now === "function" ? now : () => Date.now();
  const isHidden = typeof hidden === "function" ? hidden : () => false;
  const seen = new Map();
  return subscribe((ev) => {
    if (!ev || ev.type !== OPEN_URL_EVENT) return;
    const url = typeof ev.data?.url === "string" ? ev.data.url.trim() : "";
    if (!url) return;
    if (isHidden()) return;
    const key = (typeof ev.data.termId === "string" ? ev.data.termId : "") + "|" + url;
    if (!openUrlAllowed(seen, key, clock())) return;
    open(url);
  });
}
