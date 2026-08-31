import { useEffect, useRef, useState } from "react";
import { EditorView } from "@codemirror/view";
import { EditorState } from "@codemirror/state";
import { api, humanizeError } from "../lib/api.js";
import { askConfirm } from "../lib/confirm.js";
import { languageFor } from "../lib/fileLang.js";
import { fileEditorExtensions } from "../lib/fileEditor.js";
import { previewKind, isBlobKind, fileBlobUrl } from "../lib/filePreview.js";
import FilePreview from "./FilePreview.jsx";
import { IconExpand, IconCollapse } from "./Icons.jsx";

const FILE_MIN = 240;
const FILE_MAX = 800;
const FILE_KEY = "picode-file-w";

function fileTextUrl(agentId, termId, wsId, path) {
  const base = termId
    ? "/api/terminals/" + encodeURIComponent(termId) + "/text"
    : wsId
      ? "/api/workspaces/" + encodeURIComponent(wsId) + "/text"
      : "/api/agents/" + encodeURIComponent(agentId) + "/text";
  return base + "?path=" + encodeURIComponent(path);
}

export default function FilePane({ agentId, termId, wsId, path, onClose, variant }) {
  const [view, setView] = useState({ kind: "load" });
  const [dirty, setDirty] = useState(false);
  const [saving, setSaving] = useState(false);
  const [width, setWidth] = useState(() => {
    const n = parseInt(localStorage.getItem(FILE_KEY) || "", 10);
    return Number.isFinite(n) ? Math.min(FILE_MAX, Math.max(FILE_MIN, n)) : 420;
  });
  const [resizing, setResizing] = useState(false);
  const [expanded, setExpanded] = useState(false);
  const kind = previewKind(path);
  const [mode, setMode] = useState(kind ? "preview" : "raw");
  const hostRef = useRef(null);
  const cmRef = useRef(null);
  const mtimeRef = useRef(0);
  const dirtyRef = useRef(false);
  const saveRef = useRef(async () => {});
  dirtyRef.current = dirty;

  const ownerId = termId || wsId || agentId;
  useEffect(() => {
    if (!ownerId || !path) return;
    let stop = false;
    const blobRef = { current: "" };
    setView({ kind: "load" });
    setDirty(false);
    dirtyRef.current = false;
    const name = path.split("/").pop() || path;
    if (isBlobKind(previewKind(path))) {
      fetch(fileBlobUrl(agentId, termId, wsId, path))
        .then(async (res) => {
          if (stop) return;
          if (!res.ok) {
            let msg = res.statusText;
            try { msg = (await res.json()).error || msg; } catch { /* keep */ }
            throw new Error(msg);
          }
          const buf = await res.blob();
          blobRef.current = URL.createObjectURL(buf);
          if (stop) { URL.revokeObjectURL(blobRef.current); return; }
          setView({ kind: "bin", path, name, src: blobRef.current });
        })
        .catch((err) => {
          if (stop) return;
          const raw = err && err.message ? err.message : String(err);
          setView({ kind: "msg", path, name, text: fileMsg(raw) });
        });
    } else {
      api(fileTextUrl(agentId, termId, wsId, path))
        .then((page) => {
          if (stop) return;
          mtimeRef.current = Number(page.mtime) || 0;
          setView({ kind: "text", path: page.path || path, name: page.name || "", text: page.text || "" });
        })
        .catch((err) => {
          if (stop) return;
          const raw = err && err.message ? err.message : String(err);
          setView({ kind: "msg", path, name, text: fileMsg(raw) });
        });
    }
    return () => {
      stop = true;
      if (blobRef.current) URL.revokeObjectURL(blobRef.current);
    };
  }, [agentId, termId, ownerId, path]);

  useEffect(() => {
    setMode(previewKind(path) ? "preview" : "raw");
  }, [path]);

  useEffect(() => {
    if (view.kind !== "text" || mode === "preview" || !hostRef.current) return;
    const lang = languageFor(view.path || path);
    const dark = document.documentElement.dataset.theme !== "light";
    const state = EditorState.create({
      doc: view.text,
      extensions: fileEditorExtensions({
        lang,
        dark,
        onDoc: () => { dirtyRef.current = true; setDirty(true); },
        onSave: () => { saveRef.current(); },
      }),
    });
    const cm = new EditorView({ state, parent: hostRef.current });
    cmRef.current = cm;
    cm.focus();
    return () => {
      cm.destroy();
      cmRef.current = null;
    };
  }, [view.kind, view.path, view.text, path, mode]);

  async function save() {
    if (view.kind !== "text" || !dirtyRef.current || saving) return;
    const text = cmRef.current ? cmRef.current.state.doc.toString() : view.text;
    if (text == null) return;
    setSaving(true);
    try {
      const page = await api(fileTextUrl(agentId, termId, wsId, path).split("?")[0], {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ path: view.path || path, text, mtime: mtimeRef.current }),
      });
      mtimeRef.current = Number(page.mtime) || 0;
      dirtyRef.current = false;
      setDirty(false);
    } catch (err) {
      const raw = err && err.message ? err.message : String(err);
      if (String(raw).toLowerCase().includes("changed on disk")) {
        const ok = await askConfirm({
          title: "File changed",
          message: "This file changed on disk. Open it again to see the new version.",
          confirmLabel: "Open",
        });
        if (ok) reload();
        return;
      }
      setView({ kind: "msg", path: view.path || path, name: view.name, text: fileMsg(raw) });
    } finally {
      setSaving(false);
    }
  }
  saveRef.current = save;

  function reload() {
    setView({ kind: "load" });
    setDirty(false);
    dirtyRef.current = false;
    api(fileTextUrl(agentId, termId, wsId, path))
      .then((page) => {
        mtimeRef.current = Number(page.mtime) || 0;
        setView({ kind: "text", path: page.path || path, name: page.name || "", text: page.text || "" });
      })
      .catch((err) => {
        const raw = err && err.message ? err.message : String(err);
        setView({ kind: "msg", path, name: path.split("/").pop() || path, text: fileMsg(raw) });
      });
  }

  async function close() {
    if (dirtyRef.current) {
      const ok = await askConfirm({
        title: "Discard changes?",
        message: "Close without saving?",
        confirmLabel: "Discard",
        danger: true,
      });
      if (!ok) return;
    }
    onClose();
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
    const onKey = (e) => {
      if (e.key === "Escape") { e.preventDefault(); setExpanded(false); }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [expanded]);

  function pickMode(next) {
    if (next === mode) return;
    if (mode === "raw" && cmRef.current) {
      const t = cmRef.current.state.doc.toString();
      setView((v) => (v.kind === "text" ? { ...v, text: t } : v));
    }
    setMode(next);
  }

  // Like the tree header: the tab strip carries the basename, the header
  // confirms exactly which file — so it shows the path, not just the name.
  const title = view.path || path || "File";
  const canSave = view.kind === "text";
  const showPreview = !!kind && (view.kind === "text" || view.kind === "bin");
  const tab = variant === "tab";
  return (
    <section className={"file-pane" + (resizing ? " resizing" : "") + (expanded ? " expanded" : "") + (tab ? " file-pane-tab" : "")} aria-label={title} style={tab || expanded ? undefined : { width }}>
      {tab || expanded ? null : <div className="file-pane-sizer" title="Drag to resize" onPointerDown={onSizerDown} />}
      <header className="file-pane-bar">
        <span className="file-pane-name" title={view.path || path}>{title}</span>
        {dirty ? <span className="file-dirty" aria-label="Unsaved" /> : null}
        {showPreview ? (
          <div className="chip-group" data-align-row>
            <button type="button" className="cockpit-chip" role="radio" aria-checked={mode === "preview"} onClick={() => pickMode("preview")}>Preview</button>
            <button type="button" className="cockpit-chip" role="radio" aria-checked={mode === "raw"} onClick={() => pickMode("raw")}>Raw</button>
            {canSave ? <button type="button" className="cockpit-chip" onClick={save} disabled={!dirty || saving}>Save</button> : null}
          </div>
        ) : canSave ? (
          <button type="button" className="btn btn-primary btn-sm" onClick={save} disabled={!dirty || saving}>Save</button>
        ) : null}
        {tab ? null : <button type="button" className="btn btn-ghost btn-sm" onClick={close}>Close</button>}
        {tab ? null : (
        <button
          type="button"
          className="file-pane-expand"
          title={expanded ? "Collapse" : "Expand"}
          aria-label={expanded ? "Collapse file pane" : "Expand file pane"}
          onClick={() => setExpanded((v) => !v)}
        >
          {expanded ? <IconCollapse /> : <IconExpand />}
        </button>
        )}
      </header>
      <div className="file-pane-body">
        {view.kind === "load" ? (
          <div className="file-skel" aria-hidden="true">
            <div className="skel-line w-80" />
            <div className="skel-line w-90" />
            <div className="skel-line w-50" />
            <div className="skel-line w-70" />
          </div>
        ) : null}
        {view.kind === "text" && mode === "raw" ? <div className="file-cm" ref={hostRef} /> : null}
        {view.kind === "bin" && mode === "raw" ? <p className="file-pane-msg">Can't show this file.</p> : null}
        {mode === "preview" && kind && (view.kind === "text" || view.kind === "bin") ? (
          <FilePreview kind={kind} text={view.text} src={view.src} />
        ) : null}
        {view.kind === "msg" ? <p className="file-pane-msg">{view.text}</p> : null}
      </div>
    </section>
  );
}

function fileMsg(raw) {
  const s = String(raw || "").toLowerCase();
  if (s.includes("gone") || s.includes("not found") || s.includes("no such file")) return "That file is gone.";
  if (s.includes("too large")) return "This file is too large.";
  if (s.includes("can't show") || s.includes("can't write") || s.includes("unsupported")) return "Can't show this file.";
  if (s.includes("folder")) return "That's a folder.";
  if (s.includes("escapes")) return "That path is outside this project.";
  if (s.includes("changed on disk")) return "This file changed on disk.";
  return humanizeError(raw);
}
