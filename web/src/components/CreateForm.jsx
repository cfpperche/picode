import { useEffect, useMemo, useState } from "react";
import * as Dialog from "@radix-ui/react-dialog";
import { Drawer } from "vaul";
import ConfigFields from "./ConfigFields.jsx";
import FolderField from "./FolderField.jsx";
import { useMedia } from "../lib/media.js";
import { deriveRepo, cloneDest } from "../lib/cloneUrl.js";

export default function CreateForm({
  open, kind, workspaceName, catalog, cfg, onCfg, error, onSubmit, onClose,
  sessions, onAdopt, onKind, busy,
}) {
  const desktop = useMedia("(min-width: 720px)");
  // A workspace comes from a local folder or a remote repository (ADR-0034):
  // one choice inside the same form, not a second feature.
  const [wsSrc, setWsSrc] = useState("local");
  const [cloneUrl, setCloneUrl] = useState("");
  const [cloneName, setCloneName] = useState("");
  const [nameDirty, setNameDirty] = useState(false);
  const [clonePath, setClonePath] = useState("");
  const [pathDirty, setPathDirty] = useState(false);
  useEffect(() => {
    if (!open) return;
    setWsSrc("local");
    setCloneUrl(""); setCloneName(""); setNameDirty(false);
    setClonePath(""); setPathDirty(false);
  }, [open]);
  const cloneParent = useMemo(() => {
    try { return localStorage.getItem("picode.cloneParent") || "~/code"; } catch { return "~/code"; }
  }, [open]);
  const remote = kind === "workspace" && wsSrc === "remote";
  const title = kind === "workspace"
    ? "New workspace"
    : kind === "session"
      ? "From a Pi session"
      : kind === "agent"
        ? ("New agent" + (workspaceName ? " in " + workspaceName : ""))
        : "New agent";
  const desc = kind === "workspace"
    ? (remote
      ? "Clone a repository into a new project folder."
      : "A project folder. Add agents and terminals inside it.")
    : kind === "session"
      ? "A copy. The original stays."
      : "Provider, model, and thinking are required.";

  function onCloneUrl(e) {
    const v = e.target.value;
    setCloneUrl(v);
    // Derive name and destination from the URL while the user hasn't
    // touched those fields; a manual edit stops the derivation.
    const d = deriveRepo(v);
    if (!d.name) return;
    if (!nameDirty) setCloneName(d.name);
    if (!pathDirty) setClonePath(cloneDest(cloneParent, d.name));
  }
  function onCloneName(e) {
    const v = e.target.value;
    setCloneName(v);
    setNameDirty(true);
    if (!pathDirty && v.trim()) setClonePath(cloneDest(cloneParent, v.trim()));
  }
  const fields = kind === "session" ? (
    <SessionPicker sessions={sessions} error={error} onAdopt={onAdopt} onKind={onKind} onClose={onClose} />
  ) : (
    <form className="form-new create-form" noValidate onSubmit={onSubmit}>
      {kind === "workspace" ? (
        <>
          <div className="create-seg" role="radiogroup" aria-label="Workspace source">
            <label className="create-seg-opt">
              <input type="radio" name="ws-src" value="local" checked={wsSrc === "local"} onChange={() => setWsSrc("local")} />
              <span className="create-seg-face">Local folder</span>
            </label>
            <label className="create-seg-opt">
              <input type="radio" name="ws-src" value="remote" checked={wsSrc === "remote"} onChange={() => setWsSrc("remote")} />
              <span className="create-seg-face">Clone repository</span>
            </label>
          </div>
          {wsSrc === "local" ? (
            <>
              <input name="name" type="text" placeholder="Name (e.g. My App)" autoComplete="off" autoFocus />
              <FolderField name="path" placeholder="Folder path (e.g. ~/code/my-app)" resetKey={open} />
            </>
          ) : (
            <>
              <input
                name="url" type="text" autoComplete="off" autoFocus spellCheck={false}
                placeholder="https://github.com/org/repo or git@host:org/repo.git"
                value={cloneUrl} onChange={onCloneUrl}
              />
              <input
                name="name" type="text" placeholder="Name" autoComplete="off"
                value={cloneName} onChange={onCloneName}
              />
              <FolderField
                name="path" placeholder={"Destination (e.g. " + cloneParent + "/repo)"} resetKey={open}
                value={clonePath} onChange={(v) => { setClonePath(v); setPathDirty(true); }}
              />
            </>
          )}
          <input type="hidden" name="source" value={wsSrc} />
        </>
      ) : kind === "free" ? (
        <>
          <input name="name" type="text" placeholder="Name" autoComplete="off" autoFocus />
          <FolderField name="path" placeholder="Folder (optional — ~/.picode/work/name)" resetKey={open} />
        </>
      ) : (
        <input name="name" type="text" placeholder="Agent name" autoComplete="off" autoFocus />
      )}
      {kind !== "workspace" ? (
        <ConfigFields catalog={catalog} provider={cfg.provider} model={cfg.model} thinking={cfg.thinking} onChange={onCfg} idPrefix="create" />
      ) : null}
      <p className="form-error" hidden={!error}>{error}</p>
      {onKind && kind !== "workspace" ? (
        <p className="dlg-body">
          <button type="button" className="btn btn-ghost btn-sm" onClick={() => onKind("session")}>From a Pi session</button>
        </p>
      ) : null}
      <div className="dlg-actions">
        <button type="button" className="btn btn-ghost btn-sm" onClick={onClose}>Cancel</button>
        <button type="submit" className="btn btn-primary btn-sm" disabled={!!busy}>
          {remote ? (busy ? "Cloning…" : "Clone") : "Create"}
        </button>
      </div>
    </form>
  );

  if (desktop) {
    return (
      <Dialog.Root open={!!open} onOpenChange={(o) => { if (!o) onClose(); }}>
        <Dialog.Portal>
          <Dialog.Overlay className="dlg-overlay" />
          <Dialog.Content className={"dlg dlg-create" + (kind === "session" ? " dlg-create-session" : "")} onCloseAutoFocus={(e) => e.preventDefault()}>
            <Dialog.Title className="dlg-title">{title}</Dialog.Title>
            <Dialog.Description className="dlg-body">{desc}</Dialog.Description>
            {fields}
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>
    );
  }

  return (
    <Drawer.Root open={!!open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <Drawer.Portal>
        <Drawer.Overlay className="create-overlay" />
        <Drawer.Content className="create-drawer">
          <div className="create-handle" aria-hidden="true" />
          <Drawer.Title className="dlg-title">{title}</Drawer.Title>
          <Drawer.Description className="dlg-body">{desc}</Drawer.Description>
          {fields}
        </Drawer.Content>
      </Drawer.Portal>
    </Drawer.Root>
  );
}

function SessionPicker({ sessions, error, onAdopt, onKind, onClose }) {
  const [q, setQ] = useState("");
  useEffect(() => { setQ(""); }, [sessions]);
  const shown = useMemo(() => {
    const list = sessions || [];
    const n = q.trim().toLowerCase();
    if (!n) return list;
    return list.filter((s) => [s.name, s.preview, s.cwd].some((x) => String(x || "").toLowerCase().includes(n)));
  }, [sessions, q]);
  return (
    <div className="form-new create-form session-picker">
      {sessions == null ? (
        <div className="file-skel" aria-hidden="true">
          <div className="skel-line w-80" />
          <div className="skel-line w-50" />
        </div>
      ) : sessions.length === 0 ? (
        <p className="side-empty">No Pi sessions on this machine.{" "}
          <button type="button" className="btn btn-primary btn-sm" onClick={() => onKind && onKind("free")}>New agent</button>
        </p>
      ) : (
        <>
          <input
            type="search"
            className="session-pick-filter"
            placeholder="Search"
            value={q}
            onChange={(e) => setQ(e.target.value)}
            autoFocus
            autoComplete="off"
          />
          {shown.length === 0 ? (
            <p className="side-empty">No matching sessions.{" "}
              <button type="button" className="btn btn-ghost btn-sm" onClick={() => setQ("")}>Clear</button>
            </p>
          ) : (
            <ul className="session-pick">
              {shown.map((s) => (
                <li key={s.path}>
                  <button type="button" className="session-pick-btn" title={(s.name || s.preview || "") + "\n" + (s.cwd || "")} onClick={() => onAdopt && onAdopt(s.path)}>
                    <span className="session-pick-name">{s.name || s.preview || "Pi session"}</span>
                    {s.cwd ? <span className="session-pick-cwd">{s.cwd}</span> : null}
                  </button>
                </li>
              ))}
            </ul>
          )}
        </>
      )}
      <p className="form-error" hidden={!error}>{error}</p>
      <div className="dlg-actions">
        <button type="button" className="btn btn-ghost btn-sm" onClick={onClose}>Cancel</button>
      </div>
    </div>
  );
}
