import { useEffect, useState } from "react";
import * as Dialog from "@radix-ui/react-dialog";
import { api } from "../lib/api.js";
import { IconFolder, IconPlus } from "./Icons.jsx";

export default function FolderPicker({ open, start, onPick, onClose }) {
  const [cur, setCur] = useState("");
  const [label, setLabel] = useState("");
  const [parent, setParent] = useState("");
  const [dirs, setDirs] = useState([]);
  const [places, setPlaces] = useState([]);
  const [err, setErr] = useState("");
  const [mkdir, setMkdir] = useState("");

  useEffect(() => {
    if (!open) return;
    setMkdir("");
    load(start || "");
  }, [open, start]);

  async function load(path) {
    setErr("");
    try {
      const data = await api("/api/fs?path=" + encodeURIComponent(path || ""));
      setCur(data.path || "");
      setLabel(data.label || "");
      setParent(data.parent || "");
      setDirs(data.dirs || []);
      setPlaces(data.places || []);
    } catch (e) {
      setErr(e.message || "Can't list that folder.");
    }
  }

  async function createDir() {
    const name = mkdir.trim();
    if (!name) return;
    setErr("");
    try {
      const made = await api("/api/fs/mkdir", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ path: cur + "/" + name }),
      });
      setMkdir("");
      await load((made && made.path) || cur);
    } catch (e) {
      setErr(e.message || "Can't create folder.");
    }
  }

  return (
    <Dialog.Root open={!!open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay dlg-overlay-folder" />
        <Dialog.Content className="dlg dlg-folder" onCloseAutoFocus={(e) => e.preventDefault()}>
          <Dialog.Title className="dlg-title">Choose folder</Dialog.Title>
          <Dialog.Description className="dlg-body">{label || cur || "—"}</Dialog.Description>
          {label && cur ? <p className="folder-posix">{cur}</p> : null}
          {places.length ? (
            <div className="folder-places">
              {places.map((p) => (
                <button type="button" key={p.path} className={"folder-place" + (cur === p.path ? " on" : "")} onClick={() => load(p.path)}>{p.name}</button>
              ))}
            </div>
          ) : null}
          <div className="folder-list">
            {parent ? (
              <button type="button" className="folder-row" onClick={() => load(parent)}>.. <span className="folder-row-meta">parent</span></button>
            ) : null}
            {dirs.length === 0 ? <p className="side-empty">No subfolders</p> : null}
            {dirs.map((d) => (
              <button type="button" key={d.path} className="folder-row" onClick={() => load(d.path)}>
                <IconFolder size={14} /> {d.name}
              </button>
            ))}
          </div>
          <div className="folder-mkdir">
            <input
              className="dlg-input"
              value={mkdir}
              onChange={(e) => setMkdir(e.target.value)}
              placeholder="New folder name"
              autoComplete="off"
              onKeyDown={(e) => { if (e.key === "Enter") { e.preventDefault(); createDir(); } }}
            />
            <button type="button" className="btn btn-ghost btn-sm" onClick={createDir} disabled={!mkdir.trim()}><IconPlus size={14} /> Create</button>
          </div>
          <p className="form-error" hidden={!err}>{err}</p>
          <div className="dlg-actions">
            <button type="button" className="btn btn-ghost btn-sm" onClick={onClose}>Cancel</button>
            <button type="button" className="btn btn-primary btn-sm" onClick={() => onPick(cur)} disabled={!cur}>Use this folder</button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
