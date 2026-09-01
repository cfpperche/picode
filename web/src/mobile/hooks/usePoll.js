import { useEffect, useRef } from "react";

// usePoll: run fn now and every ms while the page is visible; pause when
// hidden, run again on the way back (visibilitychange / focus). Errors
// are the caller's to swallow — a transient miss keeps the last value.
export function usePoll(fn, ms, enabled = true) {
  const fnRef = useRef(fn);
  fnRef.current = fn;
  useEffect(() => {
    if (!enabled) return undefined;
    let timer = null;
    let stopped = false;
    const tick = async () => {
      if (stopped || document.hidden) return;
      try { await fnRef.current(); } catch { /* keep last */ }
    };
    const start = () => { if (timer) return; tick(); timer = setInterval(tick, ms); };
    const halt = () => { if (timer) clearInterval(timer); timer = null; };
    const onVis = () => { if (document.hidden) halt(); else start(); };
    start();
    document.addEventListener("visibilitychange", onVis);
    window.addEventListener("focus", tick);
    return () => {
      stopped = true;
      halt();
      document.removeEventListener("visibilitychange", onVis);
      window.removeEventListener("focus", tick);
    };
  }, [ms, enabled]);
}
