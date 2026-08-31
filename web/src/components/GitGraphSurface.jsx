import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { api } from "../lib/api.js";
import { useDebounced } from "../lib/useDebounced.js";
import { matchCommits, MIN_QUERY } from "../lib/gitgraphSearch.js";
import GitGraph from "./GitGraph.jsx";
import CommitDetail from "./CommitDetail.jsx";
import UncommittedDetail from "./UncommittedDetail.jsx";
import { UNCOMMITTED } from "../lib/gitgraph.js";

const SKELETON_ROWS = 14;

// The inline detail keeps its height across commits and sessions; below the
// minimum it is a sliver, above the ceiling it is the old bottom split again.
const DETAIL_KEY = "picode.gg.detail-h";
const DETAIL_MIN = 160;

function clampDetail(n) {
  const max = Math.max(DETAIL_MIN, Math.round(window.innerHeight * 0.7));
  return Math.min(max, Math.max(DETAIL_MIN, n));
}

// The graph of one repository (ADR-0022). The owner in `owner` is what the
// server reads through; the repository it answers with is what the tab is.

export default function GitGraphSurface({ owner, onKey, onClose }) {
  const [graph, setGraph] = useState(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [selected, setSelected] = useState("");
  const [limit, setLimit] = useState(250);
  const [detailH, setDetailH] = useState(() => {
    const n = parseInt(localStorage.getItem(DETAIL_KEY) || "", 10);
    return Number.isFinite(n) ? clampDetail(n) : 280;
  });
  const [query, setQuery] = useState("");
  const keyRef = useRef("");
  const tokenRef = useRef("");
  const busyRef = useRef(false);
  busyRef.current = busy;

  // Search dims and highlights, never hides (ADR-0038): the lanes are
  // positional. Enter walks the matches without opening any of them — the
  // click stays the one gesture that opens a detail.
  const debouncedQuery = useDebounced(query);
  const searching = debouncedQuery.trim().length >= MIN_QUERY;
  const matches = useMemo(
    () => matchCommits(graph ? graph.commits : [], debouncedQuery),
    [graph, debouncedQuery],
  );
  const matchList = useMemo(
    () => (graph ? graph.commits || [] : []).filter((c) => matches.has(c.hash)).map((c) => c.hash),
    [graph, matches],
  );
  const [matchIdx, setMatchIdx] = useState(0);
  useEffect(() => { setMatchIdx(0); }, [debouncedQuery]);
  const activeMatch = searching && matchList.length ? matchList[matchIdx % matchList.length] : "";

  const onSearchKey = (e) => {
    if (e.key !== "Enter" || matchList.length === 0) return;
    e.preventDefault();
    setMatchIdx((i) => (i + (e.shiftKey ? -1 : 1) + matchList.length) % matchList.length);
  };

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
        if (g && g.token) tokenRef.current = g.token;
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

  // Auto-refresh (ADR-0038, superseding 0030's manual-only for the graph):
  // poll a cheap token and refetch only when it moves. The surface only
  // mounts while its tab is selected, so unmounting stops the poll; the
  // hidden-document guard covers a backgrounded browser. Errors are ignored —
  // the Refresh button stays the valve.
  useEffect(() => {
    if (!ownerId) return undefined;
    let stop = false;
    const tick = async () => {
      if (stop || document.hidden || busyRef.current) return;
      try {
        const h = await api(`${base}${encodeURIComponent(ownerId)}/git/head`);
        if (!stop && h && h.token && tokenRef.current && h.token !== tokenRef.current) {
          tokenRef.current = h.token;
          load(limit);
        }
      } catch { /* ignore; manual Refresh still works */ }
    };
    const id = setInterval(tick, 5000);
    const onVis = () => { if (!document.hidden) tick(); };
    document.addEventListener("visibilitychange", onVis);
    return () => {
      stop = true;
      clearInterval(id);
      document.removeEventListener("visibilitychange", onVis);
    };
  }, [base, ownerId, load, limit]);

  // A parent link can point below the loaded window. Growing it once is the
  // polite attempt; past that, the answer is the Load earlier button, not a
  // fetch loop.
  const grewFor = useRef("");
  const selectCommit = useCallback(
    (h) => {
      if (!h) return;
      setSelected(h);
      const commits = (graph && graph.commits) || [];
      if (graph && graph.more && grewFor.current !== h && !commits.some((c) => c.hash === h)) {
        grewFor.current = h;
        setLimit((l) => l * 2);
      }
    },
    [graph],
  );

  function onSizerDown(e) {
    e.preventDefault();
    const startY = e.clientY;
    const startH = detailH;
    let latest = startH;
    const move = (ev) => {
      latest = clampDetail(Math.round(startH + (ev.clientY - startY)));
      setDetailH(latest);
    };
    const up = () => {
      try { localStorage.setItem(DETAIL_KEY, String(latest)); } catch { /* ignore */ }
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", up);
    };
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", up);
  }

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
  // A parent link can land outside the loaded window even after growing it
  // once; the anchor row does not exist, so there is nowhere to open inline.
  const selectedMissing =
    Boolean(selected) && selected !== UNCOMMITTED && count > 0 &&
    !(graph.commits || []).some((c) => c.hash === selected);

  return (
    <section className="gg-surface" aria-label={`Git graph for ${graph.name}`}>
      <header className="gg-head">
        <h2 className="gg-title">{graph.name}</h2>
        <span className="gg-count">
          {count}
          {graph.more ? "+" : ""} {count === 1 ? "commit" : "commits"}
        </span>
        <span className="gg-spacer" />
        {count > 0 ? (
          <span className="gg-search-wrap">
            <input
              type="search"
              className="gg-search"
              placeholder="Search commits"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={onSearchKey}
              aria-label="Search commits by message, author or hash"
            />
            {searching ? (
              <span className="gg-search-count">
                {matchList.length ? `${(matchIdx % matchList.length) + 1}/${matchList.length}` : "0"}
              </span>
            ) : null}
          </span>
        ) : null}
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
      {selectedMissing ? (
        <p className="gg-warn">
          That commit is earlier than the loaded window — Load earlier to reach it.
        </p>
      ) : null}

      {count === 0 ? (
        <p className="gg-msg">
          No commits yet. Make the first one, then Refresh.
        </p>
      ) : (
        <GitGraph
          graph={graph}
          selected={selected}
          onSelect={(h) => setSelected(h === selected ? "" : h)}
          matches={searching ? matches : null}
          activeMatch={activeMatch}
          detailHeight={detailH}
          onSizerDown={onSizerDown}
          detail={
            selected === UNCOMMITTED ? (
              <UncommittedDetail owner={owner} onClose={() => setSelected("")} />
            ) : selected && !selectedMissing ? (
              <CommitDetail
                owner={owner}
                hash={selected}
                onClose={() => setSelected("")}
                onSelectCommit={selectCommit}
              />
            ) : null
          }
        />
      )}
    </section>
  );
}
