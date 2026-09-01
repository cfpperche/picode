import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { api } from "../lib/api.js";
import { useDebounced } from "../lib/useDebounced.js";
import { matchCommits, MIN_QUERY } from "../lib/gitgraphSearch.js";
import { walkParams, resolveSelection } from "../lib/gitgraphBranches.js";
import GitGraph from "./GitGraph.jsx";
import GitGraphBranches from "./GitGraphBranches.jsx";
import CommitDetail from "./CommitDetail.jsx";
import UncommittedDetail from "./UncommittedDetail.jsx";
import { UNCOMMITTED } from "../lib/gitgraph.js";

const SKELETON_ROWS = 14;
const DEFAULT_LIMIT = 250;

// The inline detail keeps its height across commits and sessions; below the
// minimum it is a sliver, above the ceiling it is the old bottom split again.
const DETAIL_KEY = "picode.gg.detail-h";
const DETAIL_MIN = 160;

// Branch selection is per repository (keyed by graph.key, only known after
// the first response — same timing keyRef/onKey already handle below); the
// remotes toggle reads as one global feel-of-the-app preference, like the
// detail height.
const BRANCHES_KEY = "picode.gg.branches";
const REMOTES_KEY = "picode.gg.show-remotes";

function readStoredBranches(repoKey) {
  if (!repoKey) return [];
  try {
    const map = JSON.parse(localStorage.getItem(BRANCHES_KEY) || "{}");
    return Array.isArray(map[repoKey]) ? map[repoKey] : [];
  } catch {
    return [];
  }
}
function writeStoredBranches(repoKey, list) {
  if (!repoKey) return;
  try {
    const map = JSON.parse(localStorage.getItem(BRANCHES_KEY) || "{}");
    if (list.length) map[repoKey] = list;
    else delete map[repoKey];
    localStorage.setItem(BRANCHES_KEY, JSON.stringify(map));
  } catch {
    /* ignore */
  }
}

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
  const [limit, setLimit] = useState(DEFAULT_LIMIT);
  const [selectedBranches, setSelectedBranches] = useState([]);
  const [showRemoteBranches, setShowRemoteBranches] = useState(
    () => localStorage.getItem(REMOTES_KEY) !== "0",
  );
  const [detailH, setDetailH] = useState(() => {
    const n = parseInt(localStorage.getItem(DETAIL_KEY) || "", 10);
    return Number.isFinite(n) ? clampDetail(n) : 280;
  });
  const [query, setQuery] = useState("");
  const keyRef = useRef("");

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

  // onKey lives in a ref so `load` stays stable across parent re-renders. The
  // App re-renders on every sidebar poll and hands down a fresh onKey closure;
  // with onKey in load's deps that meant a full refetch per App render — the
  // graph flickered as if it still auto-refreshed.
  const onKeyRef = useRef(onKey);
  onKeyRef.current = onKey;

  const load = useCallback(
    async (want) => {
      if (!ownerId) return;
      setBusy(true);
      try {
        const params = new URLSearchParams({ limit: String(want) });
        const { branches, remotes } = walkParams(selectedBranches, showRemoteBranches);
        for (const name of branches) params.append("branches", name);
        if (!remotes) params.set("remotes", "0");
        const g = await api(`${base}${encodeURIComponent(ownerId)}/git?${params}`);
        setGraph(g);
        setError("");
        if (g && g.key && g.key !== keyRef.current) {
          keyRef.current = g.key;
          if (onKeyRef.current) onKeyRef.current(g.key, g.name);
          // The repo's saved selection can only be checked against its refs
          // now that we have them — a first load for any repo is always
          // unfiltered. A saved selection that still resolves triggers one
          // more, now-filtered fetch; a repo with nothing saved never pays
          // for the extra round trip.
          const resolved = resolveSelection(readStoredBranches(g.key), g.refs);
          if (resolved.length) {
            setSelectedBranches(resolved);
            setLimit(DEFAULT_LIMIT);
          }
        }
      } catch (e) {
        // Keep the last good graph on a refetch; only a first load goes blank.
        setError(e.message || "Could not read this repository.");
      } finally {
        setBusy(false);
      }
    },
    [base, ownerId, selectedBranches, showRemoteBranches],
  );

  useEffect(() => {
    load(limit);
  }, [load, limit]);

  // Refresh is manual again (back to ADR-0030; the ADR-0038 token poll is
  // gone). With several agents committing, the poll kept a ~340ms graph load
  // in flight so often that busy disabled Load earlier and Refresh most of
  // the time, and the view jumped underneath the reader.

  // Earlier commits load on demand: reaching the bottom of the scroll doubles
  // the window (no button — the scrollbar is the request). The count==limit
  // guard stops the growth once the server clamps a request, so a huge repo
  // cannot put a wiggle-the-scrollbar refetch loop at the bottom.
  const onEndReached = useCallback(() => {
    if (busy || !graph || !graph.more) return;
    if ((graph.commits || []).length < limit) return;
    setLimit(limit * 2);
  }, [busy, graph, limit]);

  // Both handlers reset limit to the default: a restrictive filter should
  // not immediately re-request whatever large window scrolling had grown,
  // against what may now be much shorter history. The refetch itself needs
  // no extra wiring — load's identity already changes with these two states,
  // so the existing `load(limit)` effect below re-runs on its own, and every
  // later scroll-triggered limit-doubling resends whatever filter is active.
  const onChangeBranches = useCallback((next) => {
    setSelectedBranches(next);
    writeStoredBranches(keyRef.current, next);
    setLimit(DEFAULT_LIMIT);
  }, []);
  const onToggleRemotes = useCallback((next) => {
    setShowRemoteBranches(next);
    try { localStorage.setItem(REMOTES_KEY, next ? "1" : "0"); } catch { /* ignore */ }
    setLimit(DEFAULT_LIMIT);
  }, []);

  // A parent link can point below the loaded window. Growing it once is the
  // polite attempt; past that, the answer is scrolling to the bottom, not a
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
        <GitGraphBranches
          refs={graph.refs}
          selected={selectedBranches}
          showRemotes={showRemoteBranches}
          onChange={onChangeBranches}
          onToggleRemotes={onToggleRemotes}
        />
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
          That commit is earlier than the loaded window — scroll to the bottom to load more history.
        </p>
      ) : null}

      {count === 0 ? (
        <p className="gg-msg">
          No commits yet. Make the first one, then Refresh.
        </p>
      ) : (
        <GitGraph
          graph={graph}
          showRemoteBranches={showRemoteBranches}
          selected={selected}
          onSelect={(h) => setSelected(h === selected ? "" : h)}
          matches={searching ? matches : null}
          activeMatch={activeMatch}
          detailHeight={detailH}
          onSizerDown={onSizerDown}
          onEndReached={onEndReached}
          loadingEarlier={busy}
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
