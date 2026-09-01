import { useState } from "react";
import AppSurface from "../../components/AppSurface.jsx";
import { usePullToRefresh } from "../hooks/usePullToRefresh.js";

// The Inbox app (ADR-0037) rendered by the same host surface as the
// desktop; below 880px it already stacks list over detail. The route's
// item id opens straight into that item so a notification tap lands on
// the decision, not the list.
export default function Inbox({ manifest, itemId }) {
  const [tick, setTick] = useState(0);
  const { ref, state } = usePullToRefresh(() => { setTick((t) => t + 1); return new Promise((r) => setTimeout(r, 300)); });
  return (
    <div className="m-screen m-inbox" ref={ref}>
      {state ? <div className={"m-pull is-" + state} aria-live="polite">{{ pull: "Pull to refresh", armed: "Release to refresh", refreshing: "Refreshing…" }[state]}</div> : null}
      <AppSurface appId="inbox" manifest={manifest || { id: "inbox", name: "Inbox", icon: "inbox", apiVersion: 1 }} hidden={false} initialPath={itemId ? "item/" + itemId : ""} refreshKey={tick} />
    </div>
  );
}
