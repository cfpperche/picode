import { useEffect, useRef } from "react";
import { feedConnected, subscribeFeed } from "../../lib/feed.js";

// usePoll: run fn now and every ms while the page is visible; pause when
// hidden, run again on the way back (visibilitychange / focus). Errors
// are the caller's to swallow — a transient miss keeps the last value.
// With the change feed connected (ADR-0048) the interval is a fallback:
// ticks are skipped, and one runs when the feed (re)opens or resets.
export function usePoll(fn, ms, enabled = true) {
  const fnRef = useRef(fn);
  fnRef.current = fn;
  useEffect(() => {
    if (!enabled) return undefined;
    let timer = null;
    let stopped = false;
    const tick = async (force) => {
      if (stopped || document.hidden) return;
      if (!force && feedConnected()) return;
      try { await fnRef.current(); } catch { /* keep last */ }
    };
    const onFeed = (ev) => { if (ev.type === "feed.open" || ev.type === "feed.reset") tick(true); };
    const unsubFeed = subscribeFeed(onFeed);
    const start = () => { if (timer) return; tick(true); timer = setInterval(() => tick(false), ms); };
    const halt = () => { if (timer) clearInterval(timer); timer = null; };
    const onVis = () => { if (document.hidden) halt(); else start(); };
    start();
    document.addEventListener("visibilitychange", onVis);
    window.addEventListener("focus", tick);
    return () => {
      stopped = true;
      halt();
      unsubFeed();
      document.removeEventListener("visibilitychange", onVis);
      window.removeEventListener("focus", tick);
    };
  }, [ms, enabled]);
}
