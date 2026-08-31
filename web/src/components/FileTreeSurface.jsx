import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "../lib/api.js";
import { changedDirs, changeKinds, flattenTree, mergeLevel, treeApiBase } from "../lib/fileTree.js";
import { toast, toastError } from "../lib/toast.js";
import FileTree from "./FileTree.jsx";
import WorkingDiff from "./WorkingDiff.jsx";

const SKELETON_ROWS = 12;

// The file tree of one folder (ADR-0030). The owner is what the server reads
// through; the folder it answers with is what the tab is. No polling — the
// tree refreshes when asked, like the git graph.
export default function FileTreeSurface({ owner, onKey, onOpenFile, onClose }) {
  const [levels, setLevels] = useState(null); // null = first load
  const [expanded, setExpanded] = useState(() => new Set());
  const [status, setStatus] = useState({ git: false, changes: [] });
  const [gone, setGone] = useState(false);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [diffPath, setDiffPath] = useState("");
  const [nonce, setNonce] = useState(0);
  const keyRef = useRef("");
  const busyRef = useRef(false);
  const loadRef = useRef(() => {});
  const expandedRef = useRef(new Set());

  const base = treeApiBase(owner ? owner.kind : "agent");
  const ownerId = owner ? owner.id : "";

  const fetchDir = useCallback(
    (dir) => api(`${base}${encodeURIComponent(ownerId)}/browse${dir ? "?dir=" + encodeURIComponent(dir) : ""}`),
    [base, ownerId],
  );

  const load = useCallback(
    async (openDirs) => {
      if (!ownerId) return;
      setBusy(true);
      try {
        const root = await fetchDir("");
        if (!root || root.cwdOk === false) {
          setGone(true);
          setLevels({});
          setError("");
          return;
        }
        setGone(false);
        let next = mergeLevel({}, root);
        for (const dir of openDirs) {
          try {
            next = mergeLevel(next, await fetchDir(dir));
          } catch {
            /* a vanished subdir just stops being expandable */
          }
        }
        setLevels(next);
        setError("");
        if (root.root && root.root !== keyRef.current) {
          keyRef.current = root.root;
          if (onKey) onKey(root.root);
        }
        try {
          const st = await api(`${base}${encodeURIComponent(ownerId)}/gitstatus`);
          setStatus(st && st.git ? st : { git: false, changes: [] });
        } catch {
          setStatus({ git: false, changes: [] });
        }
        setNonce((n) => n + 1);
      } catch (e) {
        // Keep the last good tree on a refetch; only a first load goes blank.
        setError(e.message || "Could not read this folder.");
      } finally {
        setBusy(false);
      }
    },
    [base, ownerId, fetchDir, onKey],
  );

  useEffect(() => {
    load([]);
    // A new owner is a new tree: collapse what the old one had open.
    setExpanded(new Set());
    setDiffPath("");
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [base, ownerId]);

  busyRef.current = busy;
  expandedRef.current = expanded;
  loadRef.current = load;

  const toggle = useCallback(
    async (path) => {
      const open = new Set(expanded);
      if (open.has(path)) {
        open.delete(path);
        setExpanded(open);
        return;
      }
      open.add(path);
      setExpanded(open);
      if (!levels || levels[path]) return;
      try {
        const page = await fetchDir(path);
        setLevels((prev) => mergeLevel(prev || {}, page));
      } catch (e) {
        open.delete(path);
        setExpanded(new Set(open));
        toast(e.message || "Could not read that folder.");
      }
    },
    [expanded, levels, fetchDir],
  );

  // Coming back to the app is the moment the working tree most likely moved
  // (an agent worked, a terminal committed) — refresh then, instead of
  // polling (ADR-0032; ADR-0030 still refuses a watcher).
  useEffect(() => {
    const kick = () => {
      if (document.hidden || busyRef.current) return;
      loadRef.current([...expandedRef.current]);
    };
    document.addEventListener("visibilitychange", kick);
    window.addEventListener("focus", kick);
    return () => {
      document.removeEventListener("visibilitychange", kick);
      window.removeEventListener("focus", kick);
    };
  }, []);

  const reveal = useCallback(async () => {
    try {
      await api(`${base}${encodeURIComponent(ownerId)}/reveal`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: "{}",
      });
    } catch (e) {
      toastError(e);
    }
  }, [base, ownerId]);

  if (!owner) return null;

  if (error && !levels) {
    return (
      <section className="ft-surface" aria-label="Files">
        <p className="ft-msg">
          {error}{" "}
          <button type="button" className="btn btn-sm" onClick={() => load([...expanded])}>
            Try again
          </button>
        </p>
      </section>
    );
  }

  if (levels === null) {
    return (
      <section className="ft-surface" aria-label="Files" aria-busy="true">
        <header className="ft-head">
          <span className="skel-line w-40" />
        </header>
        <div className="ft-body">
          {Array.from({ length: SKELETON_ROWS }, (_, i) => (
            <div key={i} className="skel-line" style={{ width: 30 + ((i * 19) % 50) + "%" }} />
          ))}
        </div>
      </section>
    );
  }

  const root = keyRef.current;
  const name = root ? root.split("/").filter(Boolean).pop() || root : "";
  const changes = status.git ? status.changes || [] : [];
  const kinds = changeKinds(changes);
  const dirtyDirs = changedDirs(changes);
  const rows = flattenTree(levels, expanded);

  return (
    <section className="ft-surface" aria-label={`Files in ${name}`}>
      <header className="ft-head">
        <h2 className="ft-title">{name}</h2>
        {status.git ? (
          <span className="ft-count">
            {changes.length === 0 ? "clean" : changes.length === 1 ? "1 change" : `${changes.length} changes`}
          </span>
        ) : null}
        <span className="ft-spacer" />
        <button type="button" className="btn btn-sm btn-ghost" title="Open this folder in your file manager" onClick={reveal}>
          Reveal
        </button>
        <button type="button" className="btn btn-sm btn-ghost" onClick={() => load([...expanded])} disabled={busy}>
          Refresh
        </button>
        {onClose ? (
          <button type="button" className="btn btn-sm btn-ghost" onClick={onClose}>
            Close
          </button>
        ) : null}
      </header>

      {gone ? (
        <p className="ft-msg">
          That folder is gone.{" "}
          <button type="button" className="btn btn-sm" onClick={() => load([...expanded])}>
            Refresh
          </button>
        </p>
      ) : (
        <div className={"ft-split" + (diffPath ? " ft-split-open" : "")}>
        <div className="ft-body">
          {changes.length > 0 ? (
            <div className="ft-changes">
              <h3 className="ft-sect">Changes ({changes.length})</h3>
              <ul className="ft-list">
                {changes.map((c) => (
                  <li key={c.path}>
                    <button type="button" className={"ft-row" + (diffPath === c.path ? " ft-row-on" : "")} onClick={() => setDiffPath(c.path === diffPath ? "" : c.path)} title={c.path}>
                      <span className={"ft-dot ft-dot-" + c.kind} title={c.kind} />
                      <span className="ft-name ft-name-path">{c.path}</span>
                    </button>
                  </li>
                ))}
              </ul>
              <h3 className="ft-sect">Files</h3>
            </div>
          ) : null}
          {rows.length === 0 ? (
            <p className="ft-msg">Empty folder.</p>
          ) : (
            <FileTree rows={rows} kinds={kinds} dirtyDirs={dirtyDirs} onToggle={toggle} onOpen={onOpenFile} />
          )}
        </div>
        {diffPath ? (
          <WorkingDiff
            owner={owner}
            path={diffPath}
            nonce={nonce}
            onClose={() => setDiffPath("")}
            onOpenFile={onOpenFile}
          />
        ) : null}
        </div>
      )}
    </section>
  );
}
