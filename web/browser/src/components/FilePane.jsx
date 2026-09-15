import { useEffect, useImperativeHandle, useMemo, useRef, useState, useSyncExternalStore } from "react";
import { EditorView } from "@codemirror/view";
import { EditorState } from "@codemirror/state";
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
import FilePreview from "./FilePreview.jsx";
import FileLeaveDialog from "./FileLeaveDialog.jsx";
import { IconExpand, IconCollapse } from "./Icons.jsx";

const FILE_MIN = 240;
const FILE_MAX = 800;
const FILE_KEY = "picode-file-w";

// `docKey` hands the open document to a registry outside React
// (lib/fileDocs.js) so a body that moves host — a canvas panel being
// maximized, then restored — keeps its unsaved text.
// Without it the document is this component's, as it has always been.
// `onDirty` reports that text upward: the canvas pins a dirty editor so the
// viewport never unmounts it silently.
export default function FilePane({ agentId, termId, wsId, path, onClose, variant, root = "", nonce = 0, hidden = false, controllerRef, docKey = "", onDirty, onSaved, onViewDiff, onRefreshRoot }) {
  const ownerKind = termId ? "term" : wsId ? "workspace" : "agent";
  const ownerId = termId || wsId || agentId;
  const savedRef = useRef(onSaved);
  savedRef.current = onSaved;
  const doc = useMemo(() => {
    const owner = { kind: ownerKind, id: ownerId };
    const make = () => createFileDocument({
      read: (signal) => readFile(owner, path, root, signal),
      write: async (text, mtime) => {
        const page = await api(ownerFileURL(owner, "text", "", root), {
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
  }, [ownerKind, ownerId, path, root, docKey]);
  const view = useSyncExternalStore(doc.subscribe, doc.getSnapshot);
  const rootRef = useKeptScroll(hidden, [".cm-scroller", ".file-preview"]);
  const [width, setWidth] = useState(() => {
    const n = parseInt(localStorage.getItem(FILE_KEY) || "", 10);
    return Number.isFinite(n) ? Math.min(FILE_MAX, Math.max(FILE_MIN, n)) : 420;
  });
  const [resizing, setResizing] = useState(false);
  const [expanded, setExpanded] = useState(false);
  const kind = previewKind(path);
  const [mode, setMode] = useState(kind ? "preview" : "raw");
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
  useEffect(() => { setMode(previewKind(path) ? "preview" : "raw"); }, [path]);

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
        }),
      }),
      parent: hostRef.current,
    });
    cmRef.current = cm;
    // In the tree, selection keeps its keyboard focus in the navigation.
    if (!embedded && !hiddenRef.current && modeRef.current === "raw") cm.focus();
    return () => { cm.destroy(); cmRef.current = null; };
  }, [doc, view.kind, view.revision, path, embedded]);

  useEffect(() => {
    if (!hidden && mode === "raw") cmRef.current?.requestMeasure();
  }, [hidden, mode]);

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
              <button type="button" className="cockpit-chip" aria-pressed={mode === "preview"} onClick={() => setMode("preview")}>Preview</button>
              {canSave ? <button type="button" className="cockpit-chip" aria-pressed={mode === "raw"} onClick={() => setMode("raw")}>Raw</button> : null}
            </div>
          ) : null}
          <div className="file-pane-commands" data-align-row>
            {kind === "html" && mode === "preview" && !view.dirty && (previewOnly || !previewEmpty(view.text)) ? (
              <button type="button" className="btn btn-sm btn-ghost" disabled={html.status === "loading"} onClick={() => { void html.reload(); }}>Reload</button>
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
      <div className="file-pane-body">
        {view.kind === "load" ? (
          <div className="file-skel" aria-hidden="true">
            <div className="skel-line w-80" /><div className="skel-line w-90" /><div className="skel-line w-50" /><div className="skel-line w-70" />
          </div>
        ) : null}
        {canSave ? <div className="file-cm" ref={hostRef} hidden={mode !== "raw"} /> : null}
        {mode === "preview" && showPreview ? (
          kind === "html" && !previewOnly && previewEmpty(view.text) ? (
            <p className="file-pane-msg">Nothing to preview.</p>
          ) : (
            <FilePreview kind={kind} text={view.text} src={view.src} html={kind === "html" ? html : undefined} />
          )
        ) : null}
      </div>
      <FileLeaveDialog open={leaving} path={path} onPick={pickLeave} />
    </section>
  );
}
