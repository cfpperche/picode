import { useEffect, useRef, useState } from "react";
import { EditorView } from "@codemirror/view";
import { EditorState } from "@codemirror/state";
import { api, humanizeError } from "../lib/api.js";
import { askConfirm } from "../lib/confirm.js";
import { languageFor } from "../lib/fileLang.js";
import { fileEditorExtensions } from "../lib/fileEditor.js";
import { IconExpand, IconCollapse, IconX } from "./Icons.jsx";
import ShellTerm from "./ShellTerm.jsx";

const FILE_MIN = 240;
const FILE_MAX = 800;
const FILE_KEY = "picode-file-w";

export default function FilePane({ agentId, path, shell, tab, onTab, onCloseFile, onCloseShell, onOpenShell }) {
  const [view, setView] = useState({ kind: "load" });
  const [dirty, setDirty] = useState(false);
  const [saving, setSaving] = useState(false);
  const [width, setWidth] = useState(() => {
    const n = parseInt(localStorage.getItem(FILE_KEY) || "", 10);
    return Number.isFinite(n) ? Math.min(FILE_MAX, Math.max(FILE_MIN, n)) : 420;
  });
  const [resizing, setResizing] = useState(false);
  const [expanded, setExpanded] = useState(false);
  const hostRef = useRef(null);
  const cmRef = useRef(null);
  const mtimeRef = useRef(0);
  const dirtyRef = useRef(false);
  const saveRef = useRef(async () => {});
  dirtyRef.current = dirty;

  useEffect(() => {
    if (!agentId || !path) return;
    let stop = false;
    setView({ kind: "load" });
    setDirty(false);
    dirtyRef.current = false;
    api("/api/agents/" + agentId + "/text?path=" + encodeURIComponent(path))
      .then((page) => {
        if (stop) return;
        mtimeRef.current = Number(page.mtime) || 0;
        setView({ kind: "text", path: page.path || path, name: page.name || "", text: page.text || "" });
      })
      .catch((err) => {
        if (stop) return;
        const raw = err && err.message ? err.message : String(err);
        setView({ kind: "msg", path, name: path.split("/").pop() || path, text: fileMsg(raw) });
      });
    return () => { stop = true; };
  }, [agentId, path]);

  useEffect(() => {
    if (view.kind !== "text" || !hostRef.current) return;
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
  }, [view.kind, view.path, view.text, path]);

  async function save() {
    if (view.kind !== "text" || !dirtyRef.current || saving) return;
    const cm = cmRef.current;
    if (!cm) return;
    setSaving(true);
    try {
      const page = await api("/api/agents/" + agentId + "/text", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ path: view.path || path, text: cm.state.doc.toString(), mtime: mtimeRef.current }),
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
    api("/api/agents/" + agentId + "/text?path=" + encodeURIComponent(path))
      .then((page) => {
        mtimeRef.current = Number(page.mtime) || 0;
        setView({ kind: "text", path: page.path || path, name: page.name || "", text: page.text || "" });
      })
      .catch((err) => {
        const raw = err && err.message ? err.message : String(err);
        setView({ kind: "msg", path, name: path.split("/").pop() || path, text: fileMsg(raw) });
      });
  }

  async function closeFileTab() {
    if (dirtyRef.current) {
      const ok = await askConfirm({
        title: "Discard changes?",
        message: "Close without saving?",
        confirmLabel: "Discard",
        danger: true,
      });
      if (!ok) return;
    }
    onCloseFile();
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

  const title = view.name || view.path || path || "File";
  const canSave = tab === "file" && view.kind === "text";
  const hasFile = !!path;
  const hasShell = !!(shell && (shell.session || shell.error));
  const fileOn = tab === "file" || !hasShell;
  return (
    <section className={"file-pane" + (resizing ? " resizing" : "") + (expanded ? " expanded" : "")} aria-label={fileOn ? title : "Terminal"} style={expanded ? undefined : { width }}>
      {expanded ? null : <div className="file-pane-sizer" title="Drag to resize" onPointerDown={onSizerDown} />}
      <header className="file-pane-bar">
        <div className="file-pane-tabs">
          {hasFile ? (
            <button type="button" className={"file-etab" + (fileOn ? " on" : "")} onClick={() => onTab && onTab("file")}>
              <span className="file-etab-name" title={view.path || path}>{title}</span>
              {dirty ? <span className="file-dirty" aria-label="Unsaved" /> : null}
              <span className="file-etab-x" role="button" title="Close" onClick={(e) => { e.stopPropagation(); closeFileTab(); }}><IconX size={12} /></span>
            </button>
          ) : null}
          {hasShell ? (
            <button type="button" className={"file-etab" + (!fileOn ? " on" : "")} title="Stays running if you close this tab" onClick={() => onTab && onTab("shell")}>
              <span className="file-etab-name">Terminal</span>
              <span className="file-etab-x" role="button" title="Close" onClick={(e) => { e.stopPropagation(); onCloseShell && onCloseShell(); }}><IconX size={12} /></span>
            </button>
          ) : null}
          {!hasShell && onOpenShell ? (
            <button type="button" className="file-etab file-etab-add" title="Terminal" onClick={onOpenShell}>+</button>
          ) : null}
        </div>
        {canSave ? (
          <button type="button" className="btn btn-primary btn-sm" onClick={save} disabled={!dirty || saving}>Save</button>
        ) : null}
        <button
          type="button"
          className="file-pane-expand"
          title={expanded ? "Collapse" : "Expand"}
          aria-label={expanded ? "Collapse file pane" : "Expand file pane"}
          onClick={() => setExpanded((v) => !v)}
        >
          {expanded ? <IconCollapse /> : <IconExpand />}
        </button>
      </header>
      <div className="file-pane-body">
        {hasFile ? (
          <div className="file-pane-page" hidden={!fileOn}>
            {view.kind === "load" ? (
              <div className="file-skel" aria-hidden="true">
                <div className="skel-line w-80" />
                <div className="skel-line w-90" />
                <div className="skel-line w-50" />
                <div className="skel-line w-70" />
              </div>
            ) : null}
            {view.kind === "text" ? <div className="file-cm" ref={hostRef} /> : null}
            {view.kind === "msg" ? <p className="file-pane-msg">{view.text}</p> : null}
          </div>
        ) : null}
        {hasShell ? (
          <div className="file-pane-page" hidden={fileOn}>
            {shell.error ? (
              <p className="file-pane-msg">
                {shell.error}{" "}
                <a href="#/system">Open System</a>
              </p>
            ) : (
              <ShellTerm agentId={agentId} session={shell.session} active={!fileOn} />
            )}
          </div>
        ) : null}
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
