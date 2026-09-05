import { usePullToRefresh } from "../hooks/usePullToRefresh.js";

const LABEL = { pull: "Pull to refresh", armed: "Release to refresh", refreshing: "Refreshing…" };

// A `.m-screen` scroll container with pull-to-refresh. The indicator is
// one line at the top; it exists only while a finger is pulling.
export default function PullScreen({ onRefresh, className, children }) {
  const { ref, state } = usePullToRefresh(onRefresh);
  return (
    <div className={"m-screen" + (className ? " " + className : "")} ref={ref}>
      {state ? <div className={"m-pull is-" + state} aria-live="polite">{LABEL[state]}</div> : null}
      {children}
    </div>
  );
}
