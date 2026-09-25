import { useCallback, useEffect, useImperativeHandle, useMemo, useRef, useState, useSyncExternalStore } from "react";
import { EditorView } from "@codemirror/view";
import { Compartment, EditorState } from "@codemirror/state";
import { api } from "@picode/shared/client/api.js";
import { askConfirm } from "../lib/confirm.js";
import { languageFor } from "../lib/fileLang.js";
import { fileEditorExtensions } from "../lib/fileEditor.js";
import { previewKind, previewEmpty } from "@picode/shared/domain/filePreview.js";
import { usePreviewTicket } from "../lib/usePreviewTicket.js";
import { createDocumentGuard, createFileDocument } from "../lib/fileDocument.js";
import { fileMessage, ownerFileURL, readFile } from "../lib/fileIO.js";
import { holdDocument, releaseDocument } from "../lib/fileDocs.js";
import { useKeptScroll } from "../lib/keepScroll.js";
import { useSplitSync } from "../lib/useSplitSync.js";
import { markdownLive } from "../lib/mdLive.js";
import { headingPos } from "../lib/mdLivePlan.js";
import { resolveDocLink } from "@picode/shared/domain/mdDocument.js";
import FilePreview from "./FilePreview.jsx";
import FileLeaveDialog from "./FileLeaveDialog.jsx";
import { IconExpand, IconCollapse } from "./Icons.jsx";

const FILE_MIN = 240;
const FILE_MAX = 800;
const FILE_KEY = "picode-file-w";
// How a markdown file opens (Preview, Split or Raw) is the reader's habit, not
// the file's: the last choice is remembered for the next markdown file.
const MD_VIEW_KEY = "picode-md-view";
// Split needs two readable columns; below this the pane shows the preview.
const SPLIT_MIN = 880;

function initialMode(kind) {
  if (!kind) return "raw";
  if (kind !== "markdown") return "preview";
  try {
    const saved = localStorage.getItem(MD_VIEW_KEY);
    if (saved === "live" || saved === "split" || saved === "raw") return saved;
  } catch { /* storage unavailable */ }
  return "preview";
}

// `docKey` hands the open document to a registry outside React
// (lib/fileDocs.js) so a body that moves host — a canvas panel being
// maximized, then restored — keeps its unsaved text.
// Without it the document is this component's, as it has always been.
// `onDirty` reports that text upward: the canvas pins a dirty editor so the
// viewport never unmounts it silently.
export default function FilePane({ agentId, termId, wsId, path, onClose, variant, root = "", worktree = "", nonce = 0, hidden = false, controllerRef, docKey = "", onDirty, onSaved, onViewDiff, onRefreshRoot, onOpenPath }) {
  const ownerKind = termId ? "term" : wsId ? "workspace" : "agent";
  const ownerId = termId || wsId || agentId;
  const savedRef = useRef(onSaved);
  savedRef.current = onSaved;
  const doc = useMemo(() => {
    const owner = { kind: ownerKind, id: ownerId };
    const make = () => createFileDocument({
      read: (signal) => readFile(owner, path, root, signal, worktree),
      write: async (text, mtime) => {
        const page = await api(ownerFileURL(owner, "text", "", root, worktree), {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ path, text, mtime }),
        });
        savedRef.current?.();
        return page;
      },
      release: (page) => { if (page.src) URL.revokeObjectURL(page.src); },
    });
    return docKey ? holdDocument(docKey, make) : make();
  }, [ownerKind, ownerId, path, root, worktree, docKey]);
  const view = useSyncExternalStore(doc.subscribe, doc.getSnapshot);
  // Images a markdown file points at are read from the same tree, through the
  // same file API — never a filesystem path in the page.
  const assetUrl = useCallback(
    (p, resource = "blob") => ownerFileURL({ kind: ownerKind, id: ownerId }, resource, p, root, worktree),
    [ownerKind, ownerId, root, worktree],
  );
  const rootRef = useKeptScroll(hidden, [".cm-scroller", ".file-preview"]);
  const [width, setWidth] = useState(() => {
    const n = parseInt(localStorage.getItem(FILE_KEY) || "", 10);
    return Number.isFinite(n) ? Math.min(FILE_MAX, Math.max(FILE_MIN, n)) : 420;
  });
  const [resizing, setResizing] = useState(false);
  const [expanded, setExpanded] = useState(false);
  const kind = previewKind(path);
  const [mode, setModeState] = useState(() => initialMode(kind));
  const setMode = (next) => {
    setModeState(next);
    if (kind === "markdown") { try { localStorage.setItem(MD_VIEW_KEY, next); } catch { /* storage unavailable */ } }
  };
  const bodyRef = useRef(null);
  const previewHostRef = useRef(null);
  const [bodyWidth, setBodyWidth] = useState(0);
  const [cmTick, setCmTick] = useState(0);
  // Live (markdown rendered inside the editor) is one compartment of the same
  // editor, so switching to and from Raw keeps the text, the cursor and the
  // undo history.
  const liveComp = useMemo(() => new Compartment(), []);
  const liveOnRef = useRef(false);
  const liveCtxRef = useRef(null);
  const shownRef = useRef(mode);
  // A document too large for the text read still has a page to show: the
  // ticket serves it from disk, so the pane renders preview-only — no Raw,
  // and the usual "too large to display" banner would be a lie.
  const previewOnly = kind === "html" && view.kind === "msg" && /too large/i.test(view.error || "");
  // One capability ticket per open HTML preview (ADR-0136). Unsaved editor
  // text travels as the ticket's overlay (the hook PUTs it), so the preview
  // shows what the pane holds while the assets still come from disk.
  const html = usePreviewTicket({
    ownerKind,
    ownerId,
    path,
    root,
    worktree,
    text: view.text,
    dirty: view.dirty,
    enabled: kind === "html" && mode === "preview" && (previewOnly || !previewEmpty(view.text)),
  });
  const [leaving, setLeaving] = useState(false);
  const leaveResolver = useRef(null);
  const hostRef = useRef(null);
  const cmRef = useRef(null);
  const modeRef = useRef(mode);
  const hiddenRef = useRef(hidden);
  modeRef.current = mode;
  hiddenRef.current = hidden;
  const embedded = variant === "embedded";
  const tab = variant === "tab";
  const guard = useMemo(() => createDocumentGuard(doc, () => new Promise((resolve) => {
    leaveResolver.current = resolve;
    setLeaving(true);
  })), [doc]);

  useImperativeHandle(controllerRef, () => ({ beforeLeave: guard }), [guard]);
  useEffect(() => { void doc.refresh(); }, [doc, nonce]);
  useEffect(() => () => {
    if (docKey) releaseDocument(docKey);
    else doc.dispose();
    leaveResolver.current?.("cancel");
  }, [doc, docKey]);
  const dirtyRef = useRef(onDirty);
  dirtyRef.current = onDirty;
  useEffect(() => { dirtyRef.current?.(view.dirty); }, [view.dirty]);
  // Unmounting clears the flag only when the document dies with this
  // component. With a `docKey` the text outlives it — a canvas panel being
  // maximized unmounts one body and mounts another on the same document —
  // and clearing here would drop the "Unsaved" chip, and the pin that
  // protects it, on the way past.
  useEffect(() => () => { if (!docKey) dirtyRef.current?.(false); }, [docKey]);
  useEffect(() => { setModeState(initialMode(previewKind(path))); }, [path]);

  useEffect(() => {
    if (!view.dirty && !view.saving) return;
    const protect = (event) => { event.preventDefault(); event.returnValue = ""; };
    window.addEventListener("beforeunload", protect);
    return () => window.removeEventListener("beforeunload", protect);
  }, [view.dirty, view.saving]);

  useEffect(() => {
    if (view.kind !== "text" || !hostRef.current) return;
    const cm = new EditorView({
      state: EditorState.create({
        doc: doc.getSnapshot().text,
        extensions: fileEditorExtensions({
          lang: languageFor(path),
          dark: document.documentElement.dataset.theme !== "light",
          onDoc: () => doc.edit(cm.state.doc.toString()),
          onSave: () => { void doc.save(); },
        }).concat(liveComp.of(shownRef.current === "live" ? markdownLive(() => liveCtxRef.current) : [])),
      }),
      parent: hostRef.current,
    });
    cmRef.current = cm;
    liveOnRef.current = shownRef.current === "live";
    setCmTick((t) => t + 1);
    // In the tree, selection keeps its keyboard focus in the navigation.
    if (!embedded && !hiddenRef.current && modeRef.current === "raw") cm.focus();
    return () => { cm.destroy(); cmRef.current = null; };
  }, [doc, view.kind, view.revision, path, embedded]);

  useEffect(() => {
    const el = bodyRef.current;
    if (!el) return undefined;
    const ro = new ResizeObserver(([entry]) => setBodyWidth(entry.contentRect.width));
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  function pickLeave(choice) {
    const resolve = leaveResolver.current;
    leaveResolver.current = null;
    setLeaving(false);
    resolve?.(choice);
  }

  async function reload() {
    if (view.saving) return;
    if (doc.getSnapshot().dirty && !await askConfirm({
      title: "Reload file?", message: "Discard your edits and read the file again?", confirmLabel: "Reload", danger: true,
    })) return;
    await doc.refresh({ discard: true });
  }

  async function close() {
    if (await guard()) onClose?.();
  }

  function onSizerDown(e) {
    e.preventDefault();
    const startX = e.clientX;
    const startW = width;
    let latest = startW;
    setResizing(true);
    const move = (ev) => {
      latest = Math.min(FILE_MAX, Math.max(FILE_MIN, Math.round(startW - (ev.clientX - startX))));
      setWidth(latest);
    };
    const up = () => {
      setResizing(false);
      try { localStorage.setItem(FILE_KEY, String(latest)); } catch { /* ignore */ }
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", up);
    };
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", up);
  }

  useEffect(() => {
    if (!expanded) return;
    const onKey = (e) => { if (e.key === "Escape") { e.preventDefault(); setExpanded(false); } };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [expanded]);

  const canSave = view.kind === "text";
  const showPreview = !!kind && (canSave || view.kind === "bin" || previewOnly);
  const splitOk = kind === "markdown" && canSave && bodyWidth >= SPLIT_MIN;
  // A remembered Split on a pane too narrow for it reads as Preview until the
  // pane is wide again; the choice itself is kept.
  const shown = mode === "split" && !splitOk ? "preview" : mode;
  shownRef.current = shown;
  liveCtxRef.current = {
    path,
    assetUrl,
    openLink(href) {
      const to = resolveDocLink(path, href);
      const cm = cmRef.current;
      if (to.kind === "external") window.open(to.href, "_blank", "noopener,noreferrer");
      else if (to.kind === "file") onOpenPath?.(to.path);
      else if (to.kind === "anchor" && cm) {
        const pos = headingPos(cm.state, to.id);
        if (pos >= 0) {
          cm.dispatch({ selection: { anchor: pos }, effects: EditorView.scrollIntoView(pos, { y: "start", yMargin: 16 }) });
          cm.focus();
        }
      }
    },
  };
  useEffect(() => {
    const cm = cmRef.current;
    const want = shown === "live";
    if (!cm || liveOnRef.current === want) return;
    liveOnRef.current = want;
    cm.dispatch({ effects: liveComp.reconfigure(want ? markdownLive(() => liveCtxRef.current) : []) });
  }, [shown, cmTick, liveComp]);
  useEffect(() => {
    if (!hidden && shown !== "preview") cmRef.current?.requestMeasure();
  }, [hidden, shown, bodyWidth]);
  useSplitSync({ enabled: shown === "split" && !hidden, cmRef, hostRef: previewHostRef, cmTick });
  const saveError = canSave && view.dirty && view.error;
  const conflict = /changed on disk|folder changed/i.test(view.error);
  const moved = /folder changed/i.test(view.error) && onRefreshRoot;
  return (
    <section
      className={"file-pane" + (resizing ? " resizing" : "") + (expanded ? " expanded" : "") + (tab ? " file-pane-tab" : "") + (embedded ? " file-pane-embedded" : "")}
      aria-label={path || "File"} ref={rootRef} aria-busy={view.kind === "load" || view.saving} style={tab || embedded || expanded ? undefined : { width }}
    >
      {tab || embedded || expanded ? null : <div className="file-pane-sizer" title="Drag to resize" onPointerDown={onSizerDown} />}
      <header className="file-pane-bar">
        <span className="file-pane-name" title={path}>{path}</span>
        {view.dirty ? <span className="file-dirty" aria-label="Unsaved" /> : null}
        <div className="file-pane-actions">
          {showPreview && canSave ? (
            <div className="chip-group" role="group" aria-label="File display" data-align-row>
              <button type="button" className="cockpit-chip" aria-pressed={shown === "preview"} onClick={() => setMode("preview")}>Preview</button>
              {kind === "markdown" && canSave ? <button type="button" className="cockpit-chip" aria-pressed={shown === "live"} title="Edit with formatting shown; markdown appears on the line you edit" onClick={() => setMode("live")}>Live</button> : null}
              {splitOk ? <button type="button" className="cockpit-chip" aria-pressed={shown === "split"} title="Source and preview side by side" onClick={() => setMode("split")}>Split</button> : null}
              {canSave ? <button type="button" className="cockpit-chip" aria-pressed={shown === "raw"} onClick={() => setMode("raw")}>Raw</button> : null}
            </div>
          ) : null}
          <div className="file-pane-commands" data-align-row>
            {kind === "html" && mode === "preview" && !view.dirty && (previewOnly || !previewEmpty(view.text)) ? (
              <button type="button" className="btn btn-sm btn-ghost" disabled={html.status === "loading"} onClick={() => { void html.refresh(); }}>Reload</button>
            ) : null}
            {kind === "html" && mode === "preview" && !view.dirty && html.status === "ready" ? (
              <button type="button" className="btn btn-sm btn-ghost" onClick={() => window.open(html.url, "_blank", "noopener,noreferrer")}>Open in browser</button>
            ) : null}
            {onViewDiff ? <button type="button" className="btn btn-sm btn-ghost" onClick={onViewDiff}>View diff</button> : null}
            {canSave ? <button type="button" className="btn btn-primary btn-sm" onClick={() => doc.save()} disabled={!view.dirty || view.saving}>{view.saving ? "Saving…" : "Save"}</button> : null}
            {/* Close only where there is something to close to: the tree's
                detail pane passes onClose, a canvas panel does not — and a
                button that does nothing is worse than no button. */}
            {tab || !onClose ? null : <button type="button" className="btn btn-ghost btn-sm" onClick={close} aria-label="Close file panel">Close</button>}
            {tab || embedded ? null : (
              <button type="button" className="file-pane-expand" title={expanded ? "Collapse" : "Expand"} aria-label={expanded ? "Collapse file pane" : "Expand file pane"} onClick={() => setExpanded((v) => !v)}>
                {expanded ? <IconCollapse /> : <IconExpand />}
              </button>
            )}
          </div>
        </div>
      </header>
      {view.error && !previewOnly ? (
        <p className="file-pane-notice" role="status">
          <span>{fileMessage(view.error)}</span>
          <button type="button" className="btn btn-sm btn-ghost" disabled={view.saving || view.refreshing} onClick={moved ? onRefreshRoot : saveError && !conflict ? () => doc.save() : reload}>
            {moved ? "Refresh tree" : saveError && !conflict ? "Retry save" : "Reload"}
          </button>
        </p>
      ) : null}
      <div className={"file-pane-body" + (shown === "split" ? " file-pane-split" : "")} ref={bodyRef}>
        {view.kind === "load" ? (
          <div className="file-skel" aria-hidden="true">
            <div className="skel-line w-80" /><div className="skel-line w-90" /><div className="skel-line w-50" /><div className="skel-line w-70" />
          </div>
        ) : null}
        {canSave ? <div className="file-cm" ref={hostRef} hidden={shown === "preview"} /> : null}
        {(shown === "preview" || shown === "split") && showPreview ? (<div className="file-pane-view" ref={previewHostRef}>{
          kind === "html" && !previewOnly && previewEmpty(view.text) ? (
            <p className="file-pane-msg">Nothing to preview.</p>
          ) : (
            <FilePreview kind={kind} text={view.text} src={view.src} html={kind === "html" ? html : undefined} path={path} assetUrl={assetUrl} onOpenPath={onOpenPath} sourceLines={shown === "split"} />
          )
        }</div>) : null}
      </div>
      <FileLeaveDialog open={leaving} path={path} onPick={pickLeave} />
    </section>
  );
}
