import { useCallback, useEffect, useRef, useState } from "react";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { api } from "@picode/shared/client/api.js";
import { changedDirs, changeKinds, fitTreeWidth, flattenTree, mergeLevel } from "../lib/fileTree.js";
import { ownerFileURL } from "../lib/fileIO.js";
import { useKeptScroll } from "../lib/keepScroll.js";
import { shortPath } from "@picode/shared/domain/repoLine.js";
import { toastError } from "../lib/toast.js";
import WorkspacePicker from "./WorkspacePicker.jsx";
import { pickerOptions, triggerLabel, workspaceForOwner } from "@picode/shared/domain/workspacePicker.js";
import FileTree from "./FileTree.jsx";
import FilePane from "./FilePane.jsx";
import WorkingDiff from "./WorkingDiff.jsx";

const TREE_KEY = "picode-ft-w";
const REVEAL_STALE_MS = 10_000;

// One mounted surface per canonical folder; Files and Changes share one
// local selection and detail pane (ADR-0074). The owner still authorizes it.
export default function FileTreeSurface({ owner, tabId, hidden, onKey, registerCloseGuard, onClose, workspaces = [], freeAgents = [], terminals = [], onPickWorkspace }) {
  const [levels, setLevels] = useState(null);
  const [expanded, setExpanded] = useState(() => new Set());
  const [panel, setPanel] = useState("files");
  const [status, setStatus] = useState({ git: false, changes: [] });
  const [gone, setGone] = useState(false);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [selection, setSelection] = useState(null);
  const [closing, setClosing] = useState(false);
  const closeTimer = useRef(null);
  const [nonce, setNonce] = useState(0);
  const [treeW, setTreeW] = useState(() => {
    const n = parseInt(localStorage.getItem(TREE_KEY) || "", 10);
    return Number.isFinite(n) ? Math.min(720, Math.max(220, n)) : 320;
  });
  const [available, setAvailable] = useState(1000);
  const [resizing, setResizing] = useState(false);
  const keyRef = useRef("");
  const busyRef = useRef(false);
  const loadRef = useRef(() => {});
  const expandedRef = useRef(expanded);
  const hiddenRef = useRef(hidden);
  const selectionRef = useRef(selection);
  const paneRef = useRef(null);
  const surfaceRef = useRef(null);
  const splitRef = useRef(null);
  const rootRef = useKeptScroll(hidden, [".ft-body", ".gg-detail-body"]);
  const attachRoot = useCallback((node) => { surfaceRef.current = node; rootRef(node); }, [rootRef]);
  const lastLoadRef = useRef(Date.now());
  const lifetimeRef = useRef(0);
  const navigationRef = useRef(0);
  const onKeyRef = useRef(onKey);
  onKeyRef.current = onKey;
  expandedRef.current = expanded;
  hiddenRef.current = hidden;
  selectionRef.current = selection;
  const ownerKind = owner?.kind;
  const ownerId = owner?.id;
  const beforeLeave = useCallback(() => paneRef.current?.beforeLeave() ?? Promise.resolve(true), []);

  useEffect(() => registerCloseGuard?.(tabId, beforeLeave), [tabId, registerCloseGuard, beforeLeave]);
  useEffect(() => {
    const observer = new ResizeObserver(([entry]) => { if (entry.contentRect.width > 0) setAvailable(entry.contentRect.width); });
    if (splitRef.current) observer.observe(splitRef.current);
    return () => observer.disconnect();
  }, []);

  async function select(path, mode = "file") {
    clearTimeout(closeTimer.current);
    setClosing(false);
    const previous = selectionRef.current;
    if (previous?.path === path && previous?.mode === mode) return;
    const request = ++navigationRef.current;
    if (!await beforeLeave() || request !== navigationRef.current) return;
    const next = path ? { path, mode } : null;
    selectionRef.current = next;
    setSelection(next);
  }

  function closeDetail() {
    // FilePane already obtained the user's decision before calling this.
    const path = selectionRef.current?.path;
    navigationRef.current++;
    setClosing(true);
    clearTimeout(closeTimer.current);
    closeTimer.current = setTimeout(() => {
      selectionRef.current = null;
      setSelection(null);
      setClosing(false);
      if (path) surfaceRef.current?.querySelector(`.ft-row[data-path="${CSS.escape(path)}"]`)?.focus();
    }, window.matchMedia("(prefers-reduced-motion: reduce)").matches ? 0 : 150);
  }

  const load = useCallback(async (manual = false) => {
    if (!ownerId || busyRef.current) return;
    const context = { kind: ownerKind, id: ownerId };
    const generation = lifetimeRef.current;
    busyRef.current = true;
    setBusy(true);
    try {
      const root = await api(ownerFileURL(context, "browse", "", manual ? "" : keyRef.current));
      if (generation !== lifetimeRef.current) return;
      if (!root || root.cwdOk === false) {
        setGone(true);
        setError("");
        return;
      }
      const moved = keyRef.current && root.root !== keyRef.current;
      if (moved) {
        if (!manual) { setError("This folder changed. Refresh the file tree."); return; }
        if (!await beforeLeave() || generation !== lifetimeRef.current) return;
        navigationRef.current++;
        clearTimeout(closeTimer.current);
        setClosing(false);
        selectionRef.current = null;
        setSelection(null);
        expandedRef.current = new Set();
        setExpanded(new Set());
        setPanel("files");
      }
      setGone(false);
      setError("");
      keyRef.current = root.root;
      // Only the initial response or an explicit Refresh may retarget a tab.
      onKeyRef.current?.(root.root);
      let next = mergeLevel({}, root);
      const pages = await Promise.all([...expandedRef.current].map(async (dir) => {
        try { return [dir, await api(ownerFileURL(context, "browse", dir, root.root))]; }
        catch { return [dir, null]; }
      }));
      if (generation !== lifetimeRef.current) return;
      for (const [, page] of pages) if (page) next = mergeLevel(next, page);
      const failedDirs = new Set(pages.filter(([, page]) => !page).map(([dir]) => dir));
      if (failedDirs.size) {
        setExpanded((previous) => new Set([...previous].filter((dir) => !failedDirs.has(dir))));
        setError("Could not refresh a folder. Expand it to try again.");
      }
      setLevels((prev) => {
        const merged = moved ? next : { ...prev, ...next };
        for (const [dir, page] of pages) if (!page) delete merged[dir];
        return merged;
      });
      try {
        const st = await api(ownerFileURL(context, "gitstatus", "", root.root));
        if (generation !== lifetimeRef.current) return;
        setStatus(st?.git ? st : { git: false, changes: [] });
        if (!st?.git) setPanel("files");
      } catch (e) {
        if (generation === lifetimeRef.current) setError(e.message || "Could not read changes.");
      }
      if (generation === lifetimeRef.current) setNonce((n) => n + 1);
    } catch (e) {
      if (generation === lifetimeRef.current) setError(e.message || "Could not read this folder.");
    } finally {
      if (generation === lifetimeRef.current) {
        busyRef.current = false;
        setBusy(false);
        lastLoadRef.current = Date.now();
      }
    }
  }, [ownerKind, ownerId, beforeLeave]);
  loadRef.current = load;

  useEffect(() => {
    void loadRef.current();
    return () => { lifetimeRef.current++; navigationRef.current++; busyRef.current = false; clearTimeout(closeTimer.current); };
  }, [ownerKind, ownerId]);

  useEffect(() => subscribeFeed((ev) => {
    const hit = ev.type === "git.updated" && ev.data?.path === keyRef.current;
    const reconcile = (ev.type === "feed.open" || ev.type === "feed.reset") && keyRef.current;
    if ((hit || reconcile) && !hiddenRef.current) void loadRef.current();
  }), []);

  async function toggle(path) {
    const open = new Set(expandedRef.current);
    if (open.has(path)) open.delete(path);
    else open.add(path);
    expandedRef.current = open;
    setExpanded(open);
    if (!open.has(path) || levels?.[path]) return;
    const generation = lifetimeRef.current;
    const root = keyRef.current;
    try {
      const page = await api(ownerFileURL(owner, "browse", path, root));
      if (generation === lifetimeRef.current && root === keyRef.current) setLevels((prev) => mergeLevel(prev || {}, page));
    } catch (e) {
      if (generation !== lifetimeRef.current) return;
      setExpanded((previous) => { const next = new Set(previous); next.delete(path); return next; });
      setError(e.message || "Could not read that folder.");
    }
  }

  useEffect(() => {
    const kick = () => { if (!document.hidden && !hiddenRef.current) void loadRef.current(); };
    document.addEventListener("visibilitychange", kick);
    window.addEventListener("focus", kick);
    return () => { document.removeEventListener("visibilitychange", kick); window.removeEventListener("focus", kick); };
  }, []);
  useEffect(() => {
    if (!hidden && Date.now() - lastLoadRef.current > REVEAL_STALE_MS) void loadRef.current();
  }, [hidden]);

  async function reveal() {
    try {
      await api(ownerFileURL(owner, "reveal", "", keyRef.current), {
        method: "POST", headers: { "Content-Type": "application/json" }, body: "{}",
      });
    } catch (e) { toastError(e); }
  }

  const actualWidth = fitTreeWidth(treeW, available);
  function rememberWidth(value) {
    const next = Math.min(720, Math.max(220, value));
    setTreeW(next);
    try { localStorage.setItem(TREE_KEY, String(next)); } catch { /* preference is optional */ }
  }
  function onSizerDown(e) {
    e.preventDefault();
    e.currentTarget.setPointerCapture(e.pointerId);
    const startX = e.clientX;
    const startW = actualWidth;
    let latest = startW;
    setResizing(true);
    const move = (ev) => { latest = fitTreeWidth(startW + ev.clientX - startX, available); setTreeW(latest); };
    const stop = () => {
      setResizing(false);
      rememberWidth(latest);
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", stop);
      window.removeEventListener("pointercancel", stop);
    };
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", stop);
    window.addEventListener("pointercancel", stop);
  }

  if (!owner) return null;
  const name = keyRef.current ? shortPath(keyRef.current) : "Files";
  // Which workspace's folder this tree is reading (ADR-0030 amendment), and
  // which ones it could be pointed at. A plain call, not useMemo: the early
  // return above sits before it, and a filter+sort over a handful of
  // workspaces costs nothing per render.
  const ownerWorkspace = workspaceForOwner(owner, { workspaces, freeAgents, terminals });
  const wsOptions = pickerOptions(workspaces, { hint: "path" });
  const changes = status.git ? status.changes || [] : [];
  const kinds = changeKinds(changes);
  const rows = flattenTree(levels || {}, expanded);
  const refresh = () => load(true);
  return (
    <section className="ft-surface" aria-label={`Files in ${name}`} hidden={!!hidden} ref={attachRoot}>
      <header className="ft-head">
        {/* The folder line was static text; it is the control now (ADR-0030
            amendment): the workspace whose folder this tree reads, and a picker
            of the others. Fewer than two choices keeps the plain line — a
            dropdown that can only pick what is already picked is chrome. */}
        {wsOptions.length > 1 ? (
          <WorkspacePicker
            options={wsOptions}
            value={ownerWorkspace ? ownerWorkspace.id : ""}
            label={triggerLabel(ownerWorkspace, name)}
            onPick={onPickWorkspace}
            ariaLabel="Workspace whose folder this tree reads"
          />
        ) : (
          <h2 className="ft-title" title={name}>{name}</h2>
        )}
        {status.git ? <span className="ft-count">{changes.length === 0 ? "clean" : changes.length === 1 ? "1 change" : `${changes.length} changes`}</span> : null}
        <span className="ft-spacer" />
        <div className="ft-head-actions" data-align-row>
          <button type="button" className="btn btn-sm btn-ghost" title="Open this folder in your file manager" onClick={reveal}>Reveal</button>
          <button type="button" className="btn btn-sm btn-ghost" onClick={refresh} disabled={busy}>{busy ? "Refreshing…" : "Refresh"}</button>
          {onClose ? <button type="button" className="btn btn-sm btn-ghost" onClick={onClose} aria-label="Close file tree">Close</button> : null}
        </div>
      </header>
      {error ? <p className="ft-msg ft-notice" role="status"><span>{error}</span><button type="button" className="btn btn-sm" onClick={refresh} disabled={busy}>Try again</button></p> : null}
      <div ref={splitRef} className={"ft-split" + (selection ? " ft-split-open" : "") + (resizing ? " resizing" : "")}>
        <div className="ft-body" style={selection ? { width: actualWidth } : undefined}>
          {status.git && !gone ? (
            <nav className="ft-tabs" role="tablist" aria-label="File list view" onKeyDown={(e) => {
              if (!["ArrowLeft", "ArrowRight", "Home", "End"].includes(e.key)) return;
              e.preventDefault();
              const next = e.key === "Home" ? "files" : e.key === "End" ? "changes" : panel === "files" ? "changes" : "files";
              setPanel(next);
              e.currentTarget.querySelector(`[data-panel="${next}"]`)?.focus();
            }}>
              <button type="button" role="tab" className="ft-tab" data-panel="files" tabIndex={panel === "files" ? 0 : -1} aria-selected={panel === "files"} onClick={() => setPanel("files")}>Files</button>
              <button type="button" role="tab" className="ft-tab" data-panel="changes" tabIndex={panel === "changes" ? 0 : -1} aria-selected={panel === "changes"} onClick={() => setPanel("changes")}>
                Changes{changes.length > 0 ? <span className="ft-tab-badge">{changes.length}</span> : null}
              </button>
            </nav>
          ) : null}
          {gone ? <p className="ft-msg">That folder is gone. <button type="button" className="btn btn-sm" onClick={refresh}>Refresh</button></p> : levels === null ? (
            error ? null : <div className="ft-skeleton" aria-label="Loading files" aria-busy="true">{Array.from({ length: 12 }, (_, i) => <div key={i} className="skel-line" style={{ width: 30 + ((i * 19) % 50) + "%" }} />)}</div>
          ) : status.git && panel === "changes" ? (
            changes.length === 0 ? <p className="ft-msg">No changes. <button type="button" className="btn btn-sm" onClick={() => setPanel("files")}>View files</button></p> : (
              <ul className="ft-list" aria-label="Changed files">
                {changes.map((c) => <li key={c.path}>
                  <button type="button" data-path={c.path} className={"ft-row" + (selection?.path === c.path ? " ft-row-on" : "")} aria-pressed={selection?.path === c.path} onClick={() => select(c.path, "diff")} title={c.path}>
                    <span className={"ft-dot ft-dot-" + c.kind} title={c.kind} /><span className="ft-name ft-name-path">{c.path}</span>
                  </button>
                </li>)}
              </ul>
            )
          ) : rows.length === 0 ? <p className="ft-msg">Empty folder. <button type="button" className="btn btn-sm" onClick={refresh}>Refresh</button></p> : (
            <FileTree rows={rows} kinds={kinds} dirtyDirs={changedDirs(changes)} selectedPath={selection?.path} onToggle={toggle} onOpen={(path) => select(path)} onRefresh={refresh} />
          )}
        </div>
        {selection ? <>
          <div className="ft-sizer" role="separator" aria-label="File tree width" aria-orientation="vertical" aria-valuemin={160} aria-valuemax={Math.max(160, Math.min(720, available - 260))} aria-valuenow={actualWidth} tabIndex={0} title="Drag to resize" onPointerDown={onSizerDown} onKeyDown={(e) => {
            if (e.key === "ArrowLeft" || e.key === "ArrowRight") { e.preventDefault(); rememberWidth(actualWidth + (e.key === "ArrowLeft" ? -20 : 20)); }
          }} />
          <div className={"ft-detail" + (closing ? " ft-detail-closing" : "")} key={`${selection.mode}:${selection.path}`}>
            {selection.mode === "diff" ? (
              <WorkingDiff owner={owner} root={keyRef.current} path={selection.path} nonce={nonce} onClose={closeDetail} onOpenFile={(path) => select(path)} />
            ) : (
              <FilePane
                key={`${keyRef.current}:${selection.path}`}
                agentId={ownerKind === "agent" ? ownerId : ""} termId={ownerKind === "term" ? ownerId : ""} wsId={ownerKind === "workspace" ? ownerId : ""}
                root={keyRef.current} path={selection.path} variant="embedded" nonce={nonce} hidden={hidden}
                controllerRef={paneRef} onClose={closeDetail} onSaved={() => loadRef.current()} onRefreshRoot={refresh}
                onViewDiff={kinds.has(selection.path) ? () => select(selection.path, "diff") : undefined}
                onOpenPath={(path) => select(path)}
              />
            )}
          </div>
        </> : null}
      </div>
    </section>
  );
}
