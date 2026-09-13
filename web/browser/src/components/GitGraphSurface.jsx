import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { api } from "@picode/shared/client/api.js";
import { ownerBase } from "@picode/shared/domain/gitOwner.js";
import { useDebounced } from "../lib/useDebounced.js";
import { matchCommits, MIN_QUERY } from "../lib/gitgraphSearch.js";
import { walkParams, resolveSelection } from "../lib/gitgraphBranches.js";
import GitGraph from "./GitGraph.jsx";
import { useKeptScroll } from "../lib/keepScroll.js";
import GitGraphBranches from "./GitGraphBranches.jsx";
import WorkspacePicker from "./WorkspacePicker.jsx";
import { currentWorkspaceId, pickerOptions, triggerLabel } from "../lib/gitWorkspacePicker.js";
import CommitDetail from "./CommitDetail.jsx";
import UncommittedDetail from "./UncommittedDetail.jsx";
import { UNCOMMITTED, isUncommittedHash } from "../lib/gitgraph.js";
import { undoNote } from "@picode/shared/domain/graphActions.js";

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

export default function GitGraphSurface({ owner, hidden, onKey, onClose, onMenu, actionTick = 0, done = null, onUndo, workspaces = [], freeAgents = [], terminals = [], onPickWorkspace }) {
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
  // What an action delivered, and whether the repository has answered yet
  // (ADR-0096): a command travels to a terminal or an agent, so the graph
  // learns the outcome by watching git, not by a return value.
  const [pending, setPending] = useState(null);
  const keyRef = useRef("");
  // Leaving the tab must not throw away the history the reader scrolled into:
  // the graph stays mounted, so `limit`, the open commit, the search and the
  // branch filter all survive. Refresh stays manual (see below) — a reveal
  // never refetches, so the offset it restores still matches the rows.
  const rootRef = useKeptScroll(hidden, [".gg-rows"]);

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

  // One dirty row per worktree with changes of its own (ADR-0073). Anchors
  // only survive when the worktree's HEAD is inside the loaded window — the
  // pseudo-row index in its hash (`*3`) indexes exactly this filtered list.
  const anchors = useMemo(() => {
    if (!graph) return [];
    const known = new Set((graph.commits || []).map((c) => c.hash));
    return (graph.worktrees || [])
      .filter((wt) => wt.uncommitted && wt.uncommitted.count > 0 && wt.head && known.has(wt.head))
      .map((wt) => ({ hash: wt.head, wt }));
  }, [graph]);
  const anchorFor = useCallback(
    (hash) => {
      if (!isUncommittedHash(hash)) return null;
      const k = parseInt(hash.slice(UNCOMMITTED.length), 10);
      return Number.isInteger(k) ? anchors[k] || null : null;
    },
    [anchors],
  );

  // The workspace whose folder this graph is read through (ADR-0022
  // amendment): the owner's own workspace, or — for a terminal or an agent —
  // the one it lives in. Nothing when the owner has none (a free agent or
  // terminal): the trigger then wears the repository name.
  const ownerWorkspaceId = currentWorkspaceId(owner, { workspaces, freeAgents, terminals });
  const ownerWorkspace = (workspaces || []).find((w) => w && w.id === ownerWorkspaceId) || null;
  // Only folders that are repositories are offered; the same set the sidebar
  // gates "Git graph" on, so no row can answer 404.
  const wsOptions = useMemo(() => pickerOptions(workspaces), [workspaces]);

  // The completion signal is ADR-0038's cheap token endpoint — three execs,
  // no log — polled only while an action is pending and this tab is visible.
  // No standing timer: ADR-0030 and 0073 refused one, and this one stops.
  const doneRef = useRef(null);
  doneRef.current = done;
  const tokenRef = useRef("");
  tokenRef.current = graph ? graph.token || "" : "";
  const loadRef = useRef(null);
  useEffect(() => {
    // The tick is app-wide, but an action belongs to the tab it was sent
    // from: `done` is handed only to that surface, so the others neither
    // watch nor announce it.
    const record = doneRef.current;
    if (!actionTick || !record || !ownerIdRef.current) return undefined;
    // An action belongs to the owner it was sent through: if the picker moves
    // this tab to another one, the watch ends with it. The banner is cleared
    // by the owner-change effect below, and the App already drops the undo row
    // (it only hands `done` to the owner it was recorded for).
    const sentFor = record.ownerId || "";
    // The baseline is the token read just before delivery — never the last
    // load's, which a still-pending earlier action may already have moved.
    const started = record.token || tokenRef.current;
    setPending({ since: Date.now(), settled: false });
    let live = true;
    let tries = 0;
    const tick = async () => {
      if (!live) return;
      // A hidden tab neither polls nor spends its budget: the reader who
      // comes back after a minute gets a watch that still has its 30 s.
      if (document.hidden) return;
      if (sentFor && sentFor !== ownerIdRef.current) { clearInterval(timer); return; }
      tries += 1;
      try {
        const head = await api(`${baseRef.current}${encodeURIComponent(ownerIdRef.current)}/git/head`);
        if (!live) return;
        if (head && head.token && head.token !== started) {
          setPending(null);
          if (loadRef.current) loadRef.current();
          clearInterval(timer);
          return;
        }
      } catch {
        /* a poll that fails is not an outcome; the deadline still applies */
      }
      if (tries >= 30) {
        // Out of patience is not "nothing happened": show whatever state
        // the repository is in now, under the honest line.
        setPending((p) => (p ? { ...p, settled: true } : null));
        if (loadRef.current) loadRef.current();
        clearInterval(timer);
      }
    };
    const timer = setInterval(tick, 1000);
    return () => { live = false; clearInterval(timer); };
  }, [actionTick]);

  const onSearchKey = (e) => {
    if (e.key !== "Enter" || matchList.length === 0) return;
    e.preventDefault();
    setMatchIdx((i) => (i + (e.shiftKey ? -1 : 1) + matchList.length) % matchList.length);
  };

  const base = ownerBase(owner);
  const ownerId = owner ? owner.id : "";
  // The watch above outlives a re-render, so it reads the owner through refs
  // rather than closing over values the next render replaces.
  const baseRef = useRef(base);
  baseRef.current = base;
  const ownerIdRef = useRef(ownerId);
  ownerIdRef.current = ownerId;

  // A different owner is a different folder reading the same history (ADR-0022)
  // and a different place a write is delivered (ADR-0096), so what the open
  // detail and a pending action refer to no longer holds: the open commit
  // closes (an uncommitted row is indexed positionally inside the worktree
  // list) and yesterday's "Sent — …" banner goes with the owner that sent it.
  // The loaded window, the branch filter and the search stay: those belong to
  // the repository and to the reader, and the picker changes neither.
  const shownOwner = useRef(ownerId);
  useEffect(() => {
    if (shownOwner.current === ownerId) return;
    shownOwner.current = ownerId;
    setSelected("");
    setPending(null);
  }, [ownerId]);

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

  // The pending watch (above) is declared before `load` exists, so it reaches
  // it through this ref rather than being reordered around it.
  loadRef.current = () => load(limit);

  // Refresh is manual again (back to ADR-0030; the ADR-0038 token poll is
  // gone). With several agents committing, the poll kept a ~340ms graph load
  // in flight so often that busy disabled Load earlier and Refresh most of
  // the time, and the view jumped underneath the reader. The token endpoint
  // comes back for exactly one job (ADR-0096): watching for the outcome of an
  // action this surface just delivered, and stopping when it lands.

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
      <section className="gg-surface" aria-label="Git graph" hidden={!!hidden} ref={rootRef}>
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
      <section className="gg-surface" aria-label="Git graph" aria-busy="true" hidden={!!hidden} ref={rootRef}>
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
  const gitLabel = ownerWorkspace ? `Git graph for ${ownerWorkspace.name} in ${graph.name}` : `Git graph for ${graph.name}`;
  // A parent link can land outside the loaded window even after growing it
  // once; the anchor row does not exist, so there is nowhere to open inline.
  // Pseudo rows (`*<n>`) are always present by construction.
  const selectedMissing =
    Boolean(selected) && !isUncommittedHash(selected) && count > 0 &&
    !(graph.commits || []).some((c) => c.hash === selected);

  return (
    <section className="gg-surface" aria-label={gitLabel} hidden={!!hidden} ref={rootRef}>
      <header className="gg-head">
        {wsOptions.length > 1 ? (
          <WorkspacePicker
            options={wsOptions}
            value={ownerWorkspaceId}
            label={triggerLabel(ownerWorkspace, graph.name)}
            onPick={onPickWorkspace}
          />
        ) : (
          <h2 className="gg-title">{graph.name}</h2>
        )}
        <GitGraphBranches
          refs={graph.refs}
          worktrees={graph.worktrees}
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

      {pending ? (
        <p className={"gg-pending" + (pending.settled ? " gg-pending-slow" : "")} role="status" aria-live="polite">
          {pending.settled ? (
            done && done.door === "ask"
              ? <>No change yet. The agent has it — watch its tab, then Refresh.</>
              : <>Still running. Watch it in the terminal it was sent to, then Refresh.</>
          ) : (
            <><span className="gg-pending-dot" aria-hidden="true" />
              {done && done.verb ? `Sent — ${done.verb}. Waiting for the repository to change…` : "Waiting for the repository to change…"}</>
          )}
        </p>
      ) : null}
      {done && done.undo && onUndo && (!pending || pending.settled) ? (
        <p className="gg-pending gg-pending-undo" role="status">
          <span>{undoNote(done.undo)}</span>
          <button type="button" className="btn btn-sm btn-ghost" onClick={() => onUndo(done.undo)}>Undo</button>
        </p>
      ) : null}
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
          anchors={anchors}
          showRemoteBranches={showRemoteBranches}
          selected={selected}
          onSelect={(h) => setSelected(h === selected ? "" : h)}
          matches={searching ? matches : null}
          activeMatch={activeMatch}
          detailHeight={detailH}
          onSizerDown={onSizerDown}
          onEndReached={onEndReached}
          loadingEarlier={busy}
          onMenu={onMenu && graph ? (m) => onMenu({ ...m, graph, owner, root: graph.root || "" }) : null}
          detail={
            isUncommittedHash(selected) ? (
              anchorFor(selected) ? (
                <UncommittedDetail
                  owner={owner}
                  worktree={anchorFor(selected).wt}
                  onClose={() => setSelected("")}
                />
              ) : (
                <p className="gg-msg">That worktree is gone — Refresh the graph.</p>
              )
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
