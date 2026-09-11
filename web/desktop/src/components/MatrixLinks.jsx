import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { normalizeMatrixDetail, normalizeMatrixList } from "@picode/shared/domain/matrix.js";
import { edgeGrant, peerIndex, removeConfirm } from "@picode/shared/domain/matrixGrants.js";
import { askConfirm } from "../lib/confirm.js";
import { appHash } from "../lib/routes.js";

// MatrixLinks — every live Matrix edge, listed where it can be audited
// without a plane (ADR-0116's Consequences: "a non-spatial list of every
// live edge in the Messages view, because the canvas is not the only place
// a grant may be audited").
//
// It sits under the workspace's participants because it answers the same
// question from the other side: Participants says who may talk *inside* one
// folder, this says which pairs the owner has linked by hand — including the
// pairs that cross folders, which no workspace row can show. Every row names
// both ends and their kinds, the matrix the line lives on, and whether it
// grants **right now**; Remove revokes exactly as the canvas does, because
// both call the same DELETE and both read the same `matrixGrants.js`.
//
// It is not scoped to the workspace picker above it. An edge is per pair, and
// half of its value is pairing two folders: filtering it by one folder would
// hide the rows that matter most.
//
// Reads: the matrix list plus one detail per matrix (the detail carries
// `edges` and `panels` in one request), and `GET /api/communication` for the
// enrolment. Refreshed by the feed — `matrix.*` for the lines, `peer.*` for
// what they grant — never on a timer.
const MATRIX_CAP = 50;

export default function MatrixLinks({ hidden }) {
  const [data, setData] = useState(null); // { matrices: [{matrix, panels, edges}], peers }
  const [error, setError] = useState("");
  const [busy, setBusy] = useState("");
  const live = useRef(false);
  const generation = useRef(0);

  const refresh = useCallback(async () => {
    const gen = ++generation.current;
    try {
      const [listRaw, peers] = await Promise.all([api("/api/matrices"), api("/api/communication")]);
      const list = normalizeMatrixList(listRaw).slice(0, MATRIX_CAP);
      // A matrix that vanished between the list and its detail is simply not
      // in the answer; one that fails to read must not blank the rest.
      const details = await Promise.all(list.map((m) => api("/api/matrices/" + encodeURIComponent(m.id)).then(normalizeMatrixDetail).catch(() => null)));
      if (!live.current || gen !== generation.current) return;
      setData({ matrices: details.filter(Boolean), peers });
      setError("");
    } catch (e) {
      if (live.current && gen === generation.current) setError(e.message || "Couldn’t read matrix links.");
    }
  }, []);

  useEffect(() => {
    if (hidden) return undefined;
    live.current = true;
    refresh();
    const off = subscribeFeed((ev) => {
      if (/^(matrix\.|peer\.|agent\.deleted|terminal\.deleted|feed\.(open|reset))/.test(ev.type)) refresh();
    });
    return () => { live.current = false; generation.current++; off(); };
  }, [hidden, refresh]);

  const rows = useMemo(() => {
    if (!data) return [];
    const ix = peerIndex(data.peers);
    const out = [];
    for (const det of data.matrices) {
      for (const edge of det.edges) {
        // No names of its own: an end falls back to the peer's own label,
        // which is the name the rest of this view already uses for it.
        const grant = edgeGrant(edge, det.panels, ix);
        if (!grant) continue;
        out.push({ id: edge.id, matrixId: det.matrix.id, matrix: det.matrix.name, grant });
      }
    }
    return out;
  }, [data]);

  async function remove(row) {
    const ask = removeConfirm(row.grant);
    if (ask && !(await askConfirm(ask))) return;
    setBusy(row.id);
    try {
      await api("/api/matrices/" + encodeURIComponent(row.matrixId) + "/edges/" + encodeURIComponent(row.id), { method: "DELETE" });
      setError("");
    } catch (e) {
      if (!(e && e.status === 404)) setError(e.message || "Couldn’t remove this link.");
    } finally {
      setBusy("");
      refresh();
    }
  }

  return (
    <section className="peer-body peer-links" aria-label="Matrix links">
      <div className="peer-history-heading">
        <div>
          <h4>Matrix links</h4>
          <p>Every link drawn between two panels. A link lets those two sessions message each other — it never lets one read the other’s history.</p>
        </div>
        <button className="btn btn-ghost" type="button" disabled={!!busy} onClick={refresh}>Refresh</button>
      </div>
      {error ? (
        <div className="cli-notice is-error" role="alert"><span>{error}</span><button className="btn btn-ghost" type="button" onClick={refresh}>Try again</button></div>
      ) : null}
      {!data ? (
        <div className="cli-loading" aria-label="Loading matrix links"><div /><div /><div /></div>
      ) : !rows.length ? (
        <div className="cli-notice">
          <span>No links yet. Draw one between two panels on a matrix canvas to let those sessions message each other.</span>
          <a className="btn btn-primary" href={appHash("matrix")}>Open Matrix</a>
        </div>
      ) : (
        <ul className="peer-participant-list">
          {rows.map((row) => {
            const [a, b] = row.grant.ends;
            return (
              <li key={row.id} className={"peer-participant peer-link" + (row.grant.grants ? "" : " is-broken")}>
                <div className="peer-link-ends">
                  <strong>{a.name} ↔ {b.name}</strong>
                  <small>{kindOf(a)} and {kindOf(b)} · on {row.matrix}</small>
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
