import { useCallback, useEffect, useRef, useState } from "react";
import { listDevServers } from "@picode/shared/client/devservers.js";
import { IconGlobe, IconReload } from "./Icons.jsx";

// The Servers panel: what is running on this machine right now, and one click
// to open it in PiCode's own browser surface. It is the one rail panel that
// is not about the anchor's folder — a dev server belongs to a terminal or an
// agent, and the row says which.
//
// The read is a poll (5 s) rather than a feed event: what is listening is
// machine state, not a store mutation, and the daemon probes a page title at
// most once per port per TTL. Refresh asks for a fresh probe by hand.
const POLL_MS = 5000;

export default function InspectorServers({ onOpen, hidden }) {
  const [state, setState] = useState({ servers: [], error: "", loaded: false });
  const [busy, setBusy] = useState(false);
  const gen = useRef(0);

  const load = useCallback(async (refresh) => {
    const mine = ++gen.current;
    if (refresh) setBusy(true);
    try {
      const page = await listDevServers({ refresh: !!refresh });
      if (mine !== gen.current) return;
      setState({ servers: (page && page.servers) || [], error: "", loaded: true });
    } catch (e) {
      if (mine !== gen.current) return;
      setState((s) => ({ ...s, error: e.message || "Could not list servers.", loaded: true }));
    } finally {
      if (mine === gen.current) setBusy(false);
    }
  }, []);

  useEffect(() => {
    if (hidden) return undefined;
    void load(false);
    const tick = setInterval(() => {
      if (!document.hidden) void load(false);
    }, POLL_MS);
    return () => { clearInterval(tick); gen.current++; };
  }, [hidden, load]);

  const open = (url, title) => { if (onOpen) onOpen(url, title || ""); };

  return (
    <div className="insp-servers">
      <div className="insp-servers-head">
        <span className="insp-servers-title">Running on this machine</span>
        <button type="button" className="insp-btn" title={busy ? "Refreshing…" : "Refresh"} aria-label="Refresh servers" disabled={busy} onClick={() => load(true)}>
          <IconReload size={14} className={busy ? "insp-spin" : undefined} />
        </button>
      </div>
      {state.error ? (
        <p className="insp-notice" role="status">
          <span>{state.error}</span>
          <button type="button" className="btn btn-sm" onClick={() => load(true)} disabled={busy}>Try again</button>
        </p>
      ) : null}
      {!state.error && state.servers.length === 0 ? (
        <p className="insp-msg">
          <span>{state.loaded ? "Nothing is listening yet. Start a dev server in a terminal and it shows up here." : "Looking…"}</span>
          {state.loaded && onOpen ? (
            <button type="button" className="btn btn-sm" onClick={() => open("")}>Open a URL…</button>
          ) : null}
        </p>
      ) : null}
      {state.servers.map((s) => (
        <button key={s.port} type="button" className="insp-server" onClick={() => open(s.url, s.title || s.tool)} title={"Open " + s.url + " in PiCode"}>
          <span className="insp-server-glyph" aria-hidden="true"><IconGlobe size={15} /></span>
          <span className="insp-server-main">
            <span className="insp-server-top">
              <span className="insp-server-port">{s.port}</span>
              <span className="insp-server-name">{s.title || s.tool || "unnamed page"}</span>
            </span>
            <span className="insp-server-sub">{serverSub(s)}</span>
          </span>
          <span className="insp-server-open">Open</span>
        </button>
      ))}
    </div>
  );
}

// serverSub is the quieter half of a row: where it answers first (the part a
// reader came for), then who owns the port. A server nobody in PiCode started
// is still worth a row — the panel says so instead of guessing.
function serverSub(s) {
  const who = s.ownerName ? (s.ownerKind === "agent" ? `agent "${s.ownerName}"` : `terminal "${s.ownerName}"`) : "not started here";
  const mine = s.workspace ? `${who} · ${s.workspace}` : who;
  return `${s.url} · ${mine}`;
}
