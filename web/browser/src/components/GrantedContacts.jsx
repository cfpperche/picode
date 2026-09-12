import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { normalizeCanvasDetail, normalizeCanvasList } from "@picode/shared/domain/canvas.js";
import { edgeGrant, peerIndex, removeConfirm } from "@picode/shared/domain/canvasGrants.js";
import { askConfirm } from "../lib/confirm.js";
import { appHash } from "../lib/routes.js";

// GrantedContacts — the contacts a link drawn on a canvas granted, listed
// where they can be audited without a plane (ADR-0116's Consequences: "a
// non-spatial list of every live edge in the Messages view, because the
// canvas is not the only place a grant may be audited").
//
// **This is Messages' own section, not the Canvas reaching out of its app.**
// ADR-0109's 2026-09-11 amendment closes the host's doors to an app, and
// this is the one case it declares as host business rather than a leak: the
// grant is the host's — it lives in `peer_connections` and decides who the
// mailbox lets talk — and a canvas is only where a human happened to draw
// one. So the copy is Messages' copy, the rows are contacts, and nothing
// here imports from `components/canvas/`: the shared domain modules
// (`canvas.js`, `canvasGrants.js`) are the whole dependency, the same ones
// the app reads.
//
// It sits under the workspace's participants because it answers the same
// question from the other side: Participants says who may talk *inside* one
// folder, this says which pairs the owner granted by hand — including the
// pairs that cross folders, which no workspace row can show. Every row names
// both ends and their kinds, the canvas the line lives on, and whether it
// grants **right now**; Remove revokes exactly as the canvas does, because
// both call the same DELETE and both read the same `canvasGrants.js`.
//
// It is not scoped to the workspace picker above it. An edge is per pair, and
// half of its value is pairing two folders: filtering it by one folder would
// hide the rows that matter most.
//
// Reads: the canvas list plus one detail per canvas (the detail carries
// `edges` and `panels` in one request), and `GET /api/communication` for the
// enrolment. Refreshed by the feed — `canvas.*` for the lines, `peer.*` for
// what they grant — never on a timer.
const CANVAS_CAP = 50;
const WATCHED = /^(canvas\.|peer\.|agent\.(updated|deleted)|terminal\.(updated|deleted)|feed\.(open|reset))/;
const DEBOUNCE_MS = 400;

export default function GrantedContacts({ hidden }) {
  const [data, setData] = useState(null); // { canvases: [{canvas, panels, edges}], peers }
  const [error, setError] = useState("");
  const [busy, setBusy] = useState("");
  const live = useRef(false);
  const generation = useRef(0);
  const timer = useRef(0);

  const refresh = useCallback(async () => {
    const gen = ++generation.current;
    try {
      const [listRaw, peers] = await Promise.all([api("/api/canvases"), api("/api/communication")]);
      const list = normalizeCanvasList(listRaw).slice(0, CANVAS_CAP);
      // A canvas that vanished between the list and its detail is simply not
      // in the answer; one that fails to read must not blank the rest.
      const details = await Promise.all(list.map((m) => api("/api/canvases/" + encodeURIComponent(m.id)).then(normalizeCanvasDetail).catch(() => null)));
      if (!live.current || gen !== generation.current) return;
      setData({ canvases: details.filter(Boolean), peers });
      setError("");
    } catch (e) {
      if (live.current && gen === generation.current) setError(e.message || "Couldn’t read granted contacts.");
    }
  }, []);

  useEffect(() => {
    if (hidden) return undefined;
    live.current = true;
    refresh();
    // `agent.updated` is a busy row and each refresh is a read per canvas, so
    // the feed marks the list dirty and one timer does the work. The owner
    // events are in the set because ADR-0104 invalidates a connection whose
    // recorded session moved, which no `peer.*` row announces.
    const off = subscribeFeed((ev) => {
      if (!WATCHED.test(ev.type) || timer.current) return;
      timer.current = setTimeout(() => { timer.current = 0; refresh(); }, DEBOUNCE_MS);
    });
    return () => { live.current = false; generation.current++; if (timer.current) { clearTimeout(timer.current); timer.current = 0; } off(); };
  }, [hidden, refresh]);

  const rows = useMemo(() => {
    if (!data) return [];
    const ix = peerIndex(data.peers);
    const out = [];
    for (const det of data.canvases) {
      for (const edge of det.edges) {
        // No names of its own: an end falls back to the peer's own label,
        // which is the name the rest of this view already uses for it.
        const grant = edgeGrant(edge, det.panels, ix);
        if (!grant) continue;
        out.push({ id: edge.id, canvasId: det.canvas.id, canvas: det.canvas.name, grant });
      }
    }
    return out;
  }, [data]);

  async function remove(row) {
    const ask = removeConfirm(row.grant);
    if (ask && !(await askConfirm(ask))) return;
    setBusy(row.id);
    try {
      await api("/api/canvases/" + encodeURIComponent(row.canvasId) + "/edges/" + encodeURIComponent(row.id), { method: "DELETE" });
      setError("");
    } catch (e) {
      if (!(e && e.status === 404)) setError(e.message || "Couldn’t remove this contact.");
    } finally {
      setBusy("");
      refresh();
    }
  }

  return (
    <section className="peer-body peer-links" aria-label="Granted contacts">
      <div className="peer-history-heading">
        <div>
          <h4>Granted contacts</h4>
          <p>Pairs you granted by drawing a link between two panels on a canvas. Each pair can message each other here — never read each other’s history.</p>
        </div>
        <button className="btn btn-ghost" type="button" disabled={!!busy} onClick={refresh}>Refresh</button>
      </div>
      {error ? (
        <div className="cli-notice is-error" role="alert"><span>{error}</span><button className="btn btn-ghost" type="button" onClick={refresh}>Try again</button></div>
      ) : null}
      {!data ? (
        <div className="cli-loading" aria-label="Loading granted contacts"><div /><div /><div /></div>
      ) : !rows.length ? (
        <div className="cli-notice">
          <span>No contacts granted this way yet. Draw a link between two panels on a canvas to let those two sessions message each other.</span>
          <a className="btn btn-primary" href={appHash("canvas")}>Open Canvas</a>
        </div>
      ) : (
        <ul className="peer-participant-list">
          {rows.map((row) => {
            const [a, b] = row.grant.ends;
            return (
              <li key={row.id} className={"peer-participant peer-link" + (row.grant.grants ? "" : " is-broken")}>
                <div className="peer-link-ends">
                  <strong>{a.name} ↔ {b.name}</strong>
                  <small>{kindOf(a)} and {kindOf(b)} · on {row.canvas}</small>
                </div>
                <span className="peer-participant-state" role="status">
                  {row.grant.grants ? "They can message each other" : row.grant.reason}
                </span>
                <button className="btn btn-ghost" type="button" disabled={busy === row.id} onClick={() => remove(row)}>
                  {busy === row.id ? "Removing…" : "Remove"}
                </button>
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}

// The same words the participants list above uses for the same two kinds.
const kindOf = (end) => (end.kind === "agent" ? "Pi agent" : end.cli || "Agent CLI");
