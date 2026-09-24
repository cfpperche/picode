import { useEffect, useState } from "react";

// One display-only minute tick for a subtree that renders relative ages
// (the sidebar's status pills). Nothing here polls an API — the feed owns
// state changes (ADR-0048); this only re-renders so relTime rows read
// "now" → "1m" → "2m" without waiting for an unrelated event. Paused while
// the tab is hidden (the same rhythm Automations.jsx uses) and re-reads
// the clock on return, so a tab that slept an hour renders true ages.
export function useNow(ms = 30_000) {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    let t = null;
    const start = () => { if (!t) { setNow(Date.now()); t = setInterval(() => setNow(Date.now()), ms); } };
    const stop = () => { if (t) { clearInterval(t); t = null; } };
    const vis = () => { if (document.hidden) stop(); else start(); };
    if (!document.hidden) start();
    document.addEventListener("visibilitychange", vis);
    return () => { stop(); document.removeEventListener("visibilitychange", vis); };
  }, [ms]);
  return now;
}
