import { useCallback, useEffect, useRef, useState } from "react";
import { mintPreview, previewReachable } from "@picode/shared/client/preview.js";

// One ticket per open preview (ADR-0136). enabled=false (Raw mode, another
// kind) keeps the last answer without minting; a path/owner/root change or
// reload() mints again, which is also how an expired hour is renewed. Every
// settled answer is fenced by `gen`, so a slow mint for a file the viewer
// already left can never overwrite the current one.
export function usePreviewTicket({ ownerKind, ownerId, path, root, enabled }) {
  const [state, setState] = useState({ status: "idle", url: "", error: "" });
  const gen = useRef(0);
  const load = useCallback(async () => {
    const mine = ++gen.current;
    setState((s) => ({ ...s, status: "loading", error: "" }));
    try {
      const { url } = await mintPreview({ kind: ownerKind, id: ownerId }, path, root);
      if (mine !== gen.current) return;
      if (!url || !(await previewReachable(url))) throw new Error("Can't open this preview.");
      if (mine !== gen.current) return;
      setState({ status: "ready", url, error: "" });
    } catch (err) {
      if (mine !== gen.current) return;
      const msg = err && err.message && !/failed to fetch|networkerror|load failed/i.test(err.message)
        ? err.message
        : "Can't open this preview.";
      setState({ status: "error", url: "", error: msg });
    }
  }, [ownerKind, ownerId, path, root]);
  useEffect(() => {
    if (!enabled) return undefined;
    void load();
    return () => { gen.current++; };
  }, [enabled, load]);
  return { ...state, reload: load };
}
