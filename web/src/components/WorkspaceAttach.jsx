import { useEffect, useState } from "react";
import * as Dialog from "@radix-ui/react-dialog";
import { api } from "../lib/api.js";
import { IconFile, IconFolder } from "./Icons.jsx";

export default function WorkspaceAttach({ open, agentId, onPick, onClose }) {
  const [dir, setDir] = useState("");
  const [parent, setParent] = useState("");
  const [dirs, setDirs] = useState([]);
  const [files, setFiles] = useState([]);
  const [filter, setFilter] = useState("");
  const [err, setErr] = useState("");

  useEffect(() => {
    if (!open || !agentId) return;
    setFilter("");
    load("");
  }, [open, agentId]);

  async function load(next) {
    setErr("");
    try {
      const data = await api("/api/agents/" + encodeURIComponent(agentId) + "/browse?dir=" + encodeURIComponent(next || ""));
      if (!data.cwdOk) {
        setErr("Can't read this workspace.");
        setDirs([]);
        setFiles([]);
        return;
      }
      setDir(data.dir || "");
      setParent(data.parent === undefined ? "" : data.parent);
      setDirs(data.dirs || []);
      setFiles(data.files || []);
    } catch (e) {
      setErr(e.message || "Can't list files.");
    }
  }

  const q = filter.trim().toLowerCase();
  const shownDirs = q ? dirs.filter((d) => d.name.toLowerCase().includes(q)) : dirs;
  const shownFiles = q ? files.filter((f) => f.name.toLowerCase().includes(q)) : files;

  return (
    <Dialog.Root open={!!open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay dlg-overlay-folder" />
        <Dialog.Content className="dlg dlg-folder" onCloseAutoFocus={(e) => e.preventDefault()}>
          <Dialog.Title className="dlg-title">Attach file</Dialog.Title>
          <Dialog.Description className="folder-here">
            <span className="folder-here-icon" aria-hidden="true"><IconFolder size={13} /></span>
            <span className="folder-here-text">
              <span className="folder-here-name">{dir || "Workspace"}</span>
            </span>
          </Dialog.Description>
          <input
            className="dlg-input"
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            placeholder="Filter files"
            autoComplete="off"
          />
          <div className="folder-list">
            {dir ? (
              <button type="button" className="folder-row" onClick={() => load(parent)}>
                .. <span className="folder-row-meta">parent</span>
              </button>
            ) : null}
            {shownDirs.map((d) => (
              <button type="button" key={"d-" + d.path} className="folder-row" onClick={() => load(d.path)}>
                <IconFolder size={14} /> {d.name}
              </button>
            ))}
            {shownFiles.map((f) => (
              <button type="button" key={"f-" + f.path} className="folder-row" onClick={() => onPick(f)}>
                <IconFile size={14} /> {f.name}
              </button>
            ))}
            {!shownDirs.length && !shownFiles.length ? <p className="side-empty">No files</p> : null}
          </div>
          <p className="form-error" hidden={!err}>{err}</p>
          <div className="dlg-actions">
            <button type="button" className="btn btn-ghost btn-sm" onClick={onClose}>Cancel</button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
