import { useLayoutEffect } from "react";
import { usePullToRefresh } from "../hooks/usePullToRefresh.js";

const scrollOffsets = new Map();
const LABEL = { pull: "Pull to refresh", armed: "Release to refresh", refreshing: "Refreshing…" };

// A `.m-screen` scroll container with pull-to-refresh. The indicator is
// one line at the top; it exists only while a finger is pulling.
export default function PullScreen({ onRefresh, className, children, scrollKey }) {
  const { ref, state } = usePullToRefresh(onRefresh);
  useLayoutEffect(() => {
    if (scrollKey && ref.current) ref.current.scrollTop = scrollOffsets.get(scrollKey) || 0;
  }, [scrollKey, ref]);
  return (
    <div className={"m-screen" + (className ? " " + className : "")} ref={ref} onScroll={scrollKey ? event => scrollOffsets.set(scrollKey, event.currentTarget.scrollTop) : undefined}>
      {state ? <div className={"m-pull is-" + state} aria-live="polite">{LABEL[state]}</div> : null}
      {children}
    </div>
  );
}
