import { useEffect } from "react";
import { attachSplitSync } from "@picode/shared/editor/splitSync.js";

// useSplitSync is the Split view's scroll sync (shared/editor/splitSync.js)
// for the desktop pane. `cmTick` changes whenever the editor is recreated.
export function useSplitSync({ enabled, cmRef, hostRef, cmTick }) {
  useEffect(() => {
    const cm = cmRef.current;
    const host = hostRef.current;
    if (!enabled || !cm || !host) return undefined;
    return attachSplitSync(cm, host);
  }, [enabled, cmRef, hostRef, cmTick]);
}
