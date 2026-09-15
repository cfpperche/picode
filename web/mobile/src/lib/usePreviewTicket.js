import { useCallback, useEffect, useRef, useState } from "react";
import { mintPreview, previewReachable, putPreviewOverlay } from "@picode/shared/client/preview.js";

// One ticket per open preview (ADR-0136). enabled=false (Edit mode, another
// kind, empty text) keeps the last answer without minting; a path/owner/root
// change or reload() mints again, which is also how an expired hour is
// renewed. Every settled answer is fenced by `gen`, so a slow mint for a file
// the viewer already left can never overwrite the current one.
//
// Unsaved text is the ticket's overlay: while the buffer is dirty the pane
// PUTs it and the frame renders the editor's version (relative assets still
// resolve); when the buffer is clean again the ticket is re-minted so the
// file on disk wins.
//
// Live reload: the daemon holds one EventSource per ticket. A served file
// changed on disk emits `change`, and this hook bumps a frame nonce so the
// parent reloads the iframe — the page itself is never injected into and the
// sandbox stays exactly as it was.
export function usePreviewTicket({ ownerKind, ownerId, path, root, text, dirty, enabled }) {
  const [state, setState] = useState({ status: "idle", url: "", events: "", error: "" });
  const [nonce, setNonce] = useState(0);
  const gen = useRef(0);
  const overlaid = useRef(false);
  const load = useCallback(async () => {
    const mine = ++gen.current;
    setState((s) => ({ ...s, status: "loading", error: "" }));
    try {
      const { url, events } = await mintPreview({ kind: ownerKind, id: ownerId }, path, root);
      if (mine !== gen.current) return;
      if (!url || !(await previewReachable(url))) throw new Error("Can't open this preview.");
      if (mine !== gen.current) return;
      setState({ status: "ready", url, events: events || "", error: "" });
    } catch (err) {
      if (mine !== gen.current) return;
      const msg = err && err.message && !/failed to fetch|networkerror|load failed/i.test(err.message)
        ? err.message
        : "Can't open this preview.";
      setState({ status: "error", url: "", events: "", error: msg });
    }
  }, [ownerKind, ownerId, path, root]);
  useEffect(() => {
    if (!enabled) return undefined;
    void load();
    return () => { gen.current++; };
  }, [enabled, load]);

  const { status, url, events } = state;
  useEffect(() => {
    if (!enabled || status !== "ready" || !url || !dirty) return undefined;
    let stop = false;
    void (async () => {
      try {
        await putPreviewOverlay(url, text == null ? "" : text);
      } catch {
        return; // the frame keeps the last version; the dirty dot still says why
      }
      if (!stop) setNonce((n) => n + 1);
    })();
    overlaid.current = true;
    return () => { stop = true; };
  }, [enabled, status, url, dirty, text]);

  useEffect(() => {
    if (!enabled || !overlaid.current || dirty || status !== "ready") return;
    // Saved (or discarded): a fresh ticket serves the file, not the buffer.
    overlaid.current = false;
    void load();
  }, [enabled, dirty, status, load]);

  useEffect(() => {
    if (status !== "ready" || !events) return undefined;
    let source = null;
    let failures = 0;
    let timer = 0;
    try { source = new EventSource(events); } catch { return undefined; }
    source.addEventListener("change", () => {
      clearTimeout(timer);
      timer = setTimeout(() => setNonce((n) => n + 1), 200);
    });
    source.addEventListener("error", () => {
      // EventSource retries by itself; a ticket that expired 404s forever,
      // so a few failures close the stream and leave the manual Reload.
      failures += 1;
      if (failures >= 5) source.close();
    });
    return () => { clearTimeout(timer); source.close(); };
  }, [status, events]);

  const frameUrl = state.url
    ? state.url + (state.url.includes("?") ? "&" : "?") + "v=" + nonce
    : "";
  return { ...state, frameUrl, reload: load };
}
