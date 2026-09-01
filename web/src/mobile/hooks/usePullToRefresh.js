import { useEffect, useRef, useState } from "react";

// Pull-to-refresh on a scroll container (ADR-0044 phase 3). Only when the
// list is at the top: a downward drag past 64px arms it, letting go runs
// onRefresh. The state feeds a one-line indicator; no spinner theatre.
//   returns { ref, state }  state: "" | "pull" | "armed" | "refreshing"
export function usePullToRefresh(onRefresh) {
  const ref = useRef(null);
  const [state, setState] = useState("");
  const fnRef = useRef(onRefresh);
  fnRef.current = onRefresh;
  useEffect(() => {
    const el = ref.current;
    if (!el) return undefined;
    let startY = null;
    let armed = false;
    const THRESHOLD = 64;
    const onStart = (e) => {
      if (e.touches.length !== 1 || el.scrollTop > 0) { startY = null; return; }
      startY = e.touches[0].clientY;
      armed = false;
    };
    const onMove = (e) => {
      if (startY == null || e.touches.length !== 1) return;
      const dy = e.touches[0].clientY - startY;
      if (dy <= 0 || el.scrollTop > 0) { if (armed || state) { armed = false; setState(""); } return; }
      const next = dy > THRESHOLD ? "armed" : "pull";
      armed = next === "armed";
      setState(next);
    };
    const onEnd = async () => {
      if (startY == null) return;
      startY = null;
      if (!armed) { setState(""); return; }
      armed = false;
      setState("refreshing");
      try { await fnRef.current(); } catch { /* the caller toasts */ }
      setTimeout(() => setState(""), 400);
    };
    el.addEventListener("touchstart", onStart, { passive: true });
    el.addEventListener("touchmove", onMove, { passive: true });
    el.addEventListener("touchend", onEnd);
    el.addEventListener("touchcancel", onEnd);
    return () => {
      el.removeEventListener("touchstart", onStart);
      el.removeEventListener("touchmove", onMove);
      el.removeEventListener("touchend", onEnd);
      el.removeEventListener("touchcancel", onEnd);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
  return { ref, state };
}
