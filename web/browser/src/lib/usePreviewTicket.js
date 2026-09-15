import { useCallback, useEffect, useRef, useState } from "react";
import {
  clearPreviewOverlay,
  mintPreview,
  previewForms,
  previewReachable,
  putPreviewOverlay,
} from "@picode/shared/client/preview.js";

// One ticket per open preview (ADR-0136). enabled=false (Raw mode, another
// kind, empty text) keeps the last answer without minting; a path/owner/root
// change or reload() mints again, which is also how an expired hour is
// renewed. Every settled answer is fenced by `gen`, so a slow mint for a file
// the viewer already left can never overwrite the current one.
//
// Two shapes, one ticket (ADR-0137): the daemon hands back the ticket's own
// origin (`<label>.localhost` — storage, workers, its own cookie, no sandbox)
// and the sandboxed path form. The hook tries the origin first and falls back
// when this browser cannot reach the label; `mode` and `originOffered` are
// what the pane needs to say which one it got and why.
//
// Unsaved text is the ticket's overlay: while the buffer is dirty the pane
// PUTs it and the frame renders the editor's version (relative assets still
// resolve); when the buffer is clean again the sandbox form mints a fresh
// ticket, and the origin form hands the document back to disk so the
// preview's storage survives the save.
//
// Live reload: the daemon holds one EventSource per ticket. A served file
// changed on disk emits `change`, and this hook bumps a frame nonce so the
// parent reloads the iframe — the page itself is never injected into and its
// isolation stays exactly as it was.
export function usePreviewTicket({ ownerKind, ownerId, path, root, text, dirty, enabled }) {
  const [state, setState] = useState({
    status: "idle",
    mode: "sandbox",
    originOffered: false,
    url: "",
    events: "",
    error: "",
  });
  const [nonce, setNonce] = useState(0);
  const gen = useRef(0);
  const overlaid = useRef(false);
  // mintedKey is what the live ticket was minted for: re-enabling the pane
  // for the same file (Raw → Preview) must not mint again — a new ticket is a
  // new origin, and the page's storage would be gone.
  const mintedKey = useRef("");
  const refreshRef = useRef(null);
  const ticketKey = [ownerKind, ownerId, path, root].join("\u0000");
  // overlayWrites sequences the overlay PUT/DELETE: a slow answer from a
  // write the user has already replaced may not bump the frame.
  const overlayWrites = useRef(0);

  const load = useCallback(async () => {
    const mine = ++gen.current;
    setState((s) => ({ ...s, status: "loading", error: "" }));
    try {
      const mint = await mintPreview({ kind: ownerKind, id: ownerId }, path, root);
      if (mine !== gen.current) return;
      const forms = previewForms(mint);
      const originOffered = forms.some((f) => f.mode === "origin");
      for (const form of forms) {
        if (!(await previewReachable(form.url))) continue; // this browser can't reach it
        if (mine !== gen.current) return;
        mintedKey.current = ticketKey;
        setState({ status: "ready", mode: form.mode, originOffered, url: form.url, events: form.events, error: "" });
        return;
      }
      throw new Error("Can't open this preview.");
    } catch (err) {
      if (mine !== gen.current) return;
      const msg = err && err.message && !/failed to fetch|networkerror|load failed/i.test(err.message)
        ? err.message
        : "Can't open this preview.";
      setState((s) => ({ status: "error", mode: s.mode, originOffered: s.originOffered, url: "", events: "", error: msg }));
    }
  }, [ownerKind, ownerId, path, root, ticketKey]);

  useEffect(() => {
    if (!enabled) return undefined;
    if (mintedKey.current === ticketKey) {
      // The same preview, coming back: re-read the page on the ticket it
      // already has (refresh mints again only if that ticket is gone).
      const r = refreshRef.current;
      if (r) void r();
      return undefined;
    }
    void load();
    return () => { gen.current++; };
  }, [enabled, ticketKey, load]);

  const { status, mode, originOffered, url, events } = state;

  useEffect(() => {
    if (!enabled || status !== "ready" || !url || !dirty) return undefined;
    const seq = ++overlayWrites.current;
    let stop = false;
    overlaid.current = true;
    void (async () => {
      try {
        await putPreviewOverlay(url, text == null ? "" : text);
      } catch {
        return; // the frame keeps the last version; the dirty dot still says why
      }
      if (!stop && seq === overlayWrites.current) setNonce((n) => n + 1);
    })();
    return () => { stop = true; };
  }, [enabled, status, url, dirty, text]);

  useEffect(() => {
    if (!enabled || !overlaid.current || dirty || status !== "ready" || !url) return undefined;
    overlaid.current = false;
    if (mode === "sandbox") {
      // A fresh ticket serves the file; the fallback has no storage to keep.
      void load();
      return undefined;
    }
    // Origin: keep the ticket — and with it the page's storage — and hand the
    // document back to disk, so no overlay masks a later change.
    const seq = ++overlayWrites.current;
    let stop = false;
    void (async () => {
      let ok = true;
      try {
        await clearPreviewOverlay(url);
      } catch {
        ok = false;
      }
      if (stop || seq !== overlayWrites.current) return;
      if (!ok) {
        void load(); // the ticket is gone: start over rather than mask the disk
        return;
      }
      setNonce((n) => n + 1);
    })();
    return () => { stop = true; };
  }, [enabled, dirty, status, mode, url, load]);

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

  // refresh is the toolbar's Reload: on the ticket's own origin it re-reads
  // the page without touching the ticket, so the page's storage survives; if
  // the ticket is gone (an hour passed, the daemon restarted) it starts over.
  // The error state's Retry calls reload() instead: that one mints by design.
  const refresh = useCallback(async () => {
    if (mode !== "origin" || !url) {
      await load();
      return;
    }
    let alive = false;
    try {
      alive = await previewReachable(url);
    } catch {
      alive = false;
    }
    if (alive) {
      setNonce((n) => n + 1);
      return;
    }
    await load();
  }, [mode, url, load]);

  refreshRef.current = refresh;

  const frameUrl = state.url
    ? state.url + (state.url.includes("?") ? "&" : "?") + "v=" + nonce
    : "";
  return { ...state, frameUrl, reload: load, refresh };
}
