import { useCallback, useEffect, useRef, useState } from "react";
import { stripState, wheelToScroll, arrowStep } from "./tabStrip.js";

// Phase 2 of docs/benchmarks/2026-09-06-tab-strip-overflow.md: the strip
// has no scrollbar, so this hook supplies what the benchmarks put around
// the content instead — overflow / at-start / at-end flags for the arrows
// and edge fades (Firefox, Material), a non-interactive position indicator
// that shows while the pointer is over the strip or it is moving and fades
// half a second later (VS Code's ScrollableElement HIDE_TIMEOUT), and a
// vertical wheel that scrolls sideways (VS Code scrollYToX).
//
// `ref` is the scroll box; `deps` re-measures when its content changed
// (the tabs array). Sizes come from one ResizeObserver over the strip and
// its tabs, so a rename or a status dot cannot leave a stale arrow.
const HIDE_MS = 500;
const WHEEL_SETTLE_MS = 400;

export function useTabStrip(ref, deps) {
  const [state, setState] = useState(() => stripState({ scrollLeft: 0, clientWidth: 0, scrollWidth: 0 }));
  const [scrolling, setScrolling] = useState(false);
  const observer = useRef(null);
  // Where the last wheel tick sent the strip. A smooth scroll is still in
  // flight when the next tick arrives, and reading scrollLeft then would
  // lose the distance already asked for (the strip would crawl).
  const target = useRef(null);
  const timers = useRef({ hide: 0, wheel: 0 });

  const measure = useCallback(() => {
    const el = ref.current;
    if (!el) return;
    const next = stripState(el);
    setState((prev) => (same(prev, next) ? prev : next));
  }, [ref]);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const ro = new ResizeObserver(measure);
    observer.current = ro;
    const onScroll = () => {
      measure();
      setScrolling(true);
      clearTimeout(timers.current.hide);
      timers.current.hide = setTimeout(() => setScrolling(false), HIDE_MS);
    };
    const settle = () => { target.current = null; };
    const onWheel = (e) => {
      const dx = wheelToScroll(e, el.clientWidth);
      if (!dx) return;
      const max = el.scrollWidth - el.clientWidth;
      if (max <= 1) return;
      const from = target.current ?? el.scrollLeft;
      const to = Math.min(max, Math.max(0, from + dx));
      // Already at that end: not consumed (VS Code consumes only what it scrolled).
      if (to === from) return;
      e.preventDefault();
      target.current = to;
      el.scrollTo({ left: to });
      clearTimeout(timers.current.wheel);
      timers.current.wheel = setTimeout(settle, WHEEL_SETTLE_MS);
    };
    el.addEventListener("scroll", onScroll, { passive: true });
    el.addEventListener("scrollend", settle);
    // React registers wheel listeners as passive; preventDefault needs this one.
    el.addEventListener("wheel", onWheel, { passive: false });
    return () => {
      ro.disconnect();
      observer.current = null;
      el.removeEventListener("scroll", onScroll);
      el.removeEventListener("scrollend", settle);
      el.removeEventListener("wheel", onWheel);
      clearTimeout(timers.current.hide);
      clearTimeout(timers.current.wheel);
    };
  }, [ref, measure]);

  // Content changed: watch the current tabs and measure once now.
  useEffect(() => {
    const el = ref.current;
    const ro = observer.current;
    if (!el || !ro) return;
    ro.disconnect();
    ro.observe(el);
    for (const child of el.children) ro.observe(child);
    measure();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);

  const step = useCallback((dir) => {
    const el = ref.current;
    if (!el) return;
    target.current = null;
    el.scrollBy({ left: dir * arrowStep(el.clientWidth) });
  }, [ref]);

  return { ...state, scrolling, step };
}

function same(a, b) {
  return a.overflow === b.overflow && a.atStart === b.atStart && a.atEnd === b.atEnd
    && a.thumb.left === b.thumb.left && a.thumb.width === b.thumb.width;
}
