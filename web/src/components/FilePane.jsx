import { useEffect, useRef, useState } from "react";
import { EditorView, keymap, lineNumbers, highlightActiveLine } from "@codemirror/view";
import { EditorState } from "@codemirror/state";
import { defaultKeymap, history, historyKeymap, indentWithTab } from "@codemirror/commands";
import { oneDark } from "@codemirror/theme-one-dark";
import { api, humanizeError } from "../lib/api.js";
import { askConfirm } from "../lib/confirm.js";
import { languageFor } from "../lib/fileLang.js";

export default function FilePane({ agentId, path, onClose }) {
  const [view, setView] = useState({ kind: "load" });
  const [dirty, setDirty] = useState(false);
  const [saving, setSaving] = useState(false);
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
    const onChange = EditorView.updateListener.of((u) => {
      if (!u.docChanged) return;
      dirtyRef.current = true;
      setDirty(true);
    });
    const saveKey = keymap.of([{
      key: "Mod-s",
      run: () => { saveRef.current(); return true; },
    }]);
    const state = EditorState.create({
      doc: view.text,
      extensions: [
        lineNumbers(),
        highlightActiveLine(),
        history(),
        saveKey,
        keymap.of([...defaultKeymap, ...historyKeymap, indentWithTab]),
        onChange,
        EditorView.lineWrapping,
        ...(lang ? [lang] : []),
        ...(dark ? [oneDark] : []),
      ],
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

  const title = view.name || view.path || path || "File";
  const canSave = view.kind === "text";
  return (
    <section className="file-pane" aria-label={title}>
      <header className="file-pane-bar">
        <span className="file-pane-name" title={view.path || path}>{title}</span>
        {dirty ? <span className="file-dirty" aria-label="Unsaved" /> : null}
        {canSave ? (
          <button type="button" className="btn btn-primary btn-sm" onClick={save} disabled={!dirty || saving}>Save</button>
        ) : null}
        <button type="button" className="btn btn-ghost btn-sm" onClick={close}>Close</button>
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
        {view.kind === "text" ? <div className="file-cm" ref={hostRef} /> : null}
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
