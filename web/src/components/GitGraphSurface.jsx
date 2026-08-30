import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "../lib/api.js";
import GitGraph from "./GitGraph.jsx";
import CommitDetail from "./CommitDetail.jsx";

const SKELETON_ROWS = 14;

// The graph of one repository (ADR-0022). The owner in `owner` is what the
// server reads through; the repository it answers with is what the tab is.

export default function GitGraphSurface({ owner, onKey, onClose }) {
  const [graph, setGraph] = useState(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [selected, setSelected] = useState("");
  const [limit, setLimit] = useState(250);
  const keyRef = useRef("");

  const base = owner && owner.kind === "term" ? "/api/terminals/" : "/api/agents/";
  const ownerId = owner ? owner.id : "";

  const load = useCallback(
    async (want) => {
      if (!ownerId) return;
      setBusy(true);
      try {
        const g = await api(`${base}${encodeURIComponent(ownerId)}/git?limit=${want}`);
        setGraph(g);
        setError("");
        if (g && g.key && g.key !== keyRef.current) {
          keyRef.current = g.key;
          if (onKey) onKey(g.key, g.name);
        }
      } catch (e) {
        // Keep the last good graph on a refetch; only a first load goes blank.
        setError(e.message || "Could not read this repository.");
      } finally {
        setBusy(false);
      }
    },
    [base, ownerId, onKey],
  );

  useEffect(() => {
    load(limit);
  }, [load, limit]);

  if (!owner) return null;

  if (error && !graph) {
    return (
      <section className="gg-surface" aria-label="Git graph">
        <p className="gg-msg">
          {error}{" "}
          <button type="button" className="btn btn-sm" onClick={() => load(limit)}>
            Try again
          </button>
        </p>
      </section>
    );
  }

  if (!graph) {
    return (
      <section className="gg-surface" aria-label="Git graph" aria-busy="true">
        <header className="gg-head">
          <span className="gg-skel gg-skel-title" />
        </header>
        <div className="gg-rows gg-rows-skel">
          <ol className="gg-list">
            {Array.from({ length: SKELETON_ROWS }, (_, i) => (
              <li key={i}>
                <span className="gg-row gg-row-skel">
                  <span className="gg-skel gg-skel-dot" />
                  <span className="gg-skel gg-skel-line" style={{ width: 30 + ((i * 17) % 45) + "%" }} />
                  <span className="gg-skel gg-skel-meta" />
                </span>
              </li>
            ))}
          </ol>
        </div>
      </section>
    );
  }

  const count = (graph.commits || []).length;

  return (
    <section className="gg-surface" aria-label={`Git graph for ${graph.name}`}>
      <header className="gg-head">
        <h2 className="gg-title">{graph.name}</h2>
        <span className="gg-count">
          {count}
          {graph.more ? "+" : ""} {count === 1 ? "commit" : "commits"}
        </span>
        <span className="gg-spacer" />
        {graph.more ? (
          <button type="button" className="btn btn-sm btn-ghost" onClick={() => setLimit(limit * 2)} disabled={busy}>
            Load earlier
          </button>
        ) : null}
        <button type="button" className="btn btn-sm btn-ghost" onClick={() => load(limit)} disabled={busy}>
          Refresh
        </button>
        {onClose ? (
          <button type="button" className="btn btn-sm btn-ghost" onClick={onClose}>
            Close
          </button>
        ) : null}
      </header>

      {error ? <p className="gg-warn">{error}</p> : null}

      {count === 0 ? (
        <p className="gg-msg">
          No commits yet. Make the first one, then Refresh.
        </p>
      ) : (
        <div className={"gg-split" + (selected ? " gg-split-open" : "")}>
          <GitGraph graph={graph} selected={selected} onSelect={(h) => setSelected(h === selected ? "" : h)} />
          {selected ? (
            <CommitDetail owner={owner} hash={selected} onClose={() => setSelected("")} />
          ) : null}
        </div>
      )}
    </section>
  );
}
