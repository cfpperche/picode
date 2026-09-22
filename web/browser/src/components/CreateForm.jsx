import { useEffect, useMemo, useState } from "react";
import * as Dialog from "./ResponsiveDialog.jsx";
import ConfigFields from "./ConfigFields.jsx";
import FolderField from "./FolderField.jsx";
import RepoField from "./RepoField.jsx";
import { deriveRepo, cloneDest } from "@picode/shared/domain/cloneUrl.js";
import { IconFolder, IconGit } from "./Icons.jsx";

export default function CreateForm({
  open,
  kind,
  workspaceName,
  catalog,
  cfg,
  onCfg,
  error,
  onSubmit,
  onClose,
  busy,
}) {
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
    setCloneUrl("");
    setCloneName("");
    setNameDirty(false);
    setClonePath("");
    setPathDirty(false);
  }, [open]);
  const cloneParent = useMemo(() => {
    try {
      return localStorage.getItem("picode.cloneParent") || "~/code";
    } catch {
      return "~/code";
    }
  }, [open]);
  const remote = kind === "workspace" && wsSrc === "remote";
  const title =
    kind === "workspace"
      ? "New workspace"
      : "New agent" + (workspaceName ? " in " + workspaceName : "");
  const desc =
    kind === "workspace"
      ? remote
        ? "Clone a repository into a new project folder."
        : "A project folder. Add agents and terminals inside it."
      : "Provider, model, and thinking are required.";

  function applyCloneUrl(v) {
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
  const fields = (
    <form className="form-new create-form" noValidate onSubmit={onSubmit}>
      {kind === "workspace" ? (
        <>
          <div
            className="create-seg"
            role="radiogroup"
            aria-label="Workspace source"
          >
            <label className="create-seg-opt">
              <input
                type="radio"
                name="ws-src"
                value="local"
                checked={wsSrc === "local"}
                onChange={() => setWsSrc("local")}
              />
              <span className="create-seg-face">
                <IconFolder size={13} /> Local folder
              </span>
            </label>
            <label className="create-seg-opt">
              <input
                type="radio"
                name="ws-src"
                value="remote"
                checked={wsSrc === "remote"}
                onChange={() => setWsSrc("remote")}
              />
              <span className="create-seg-face">
                <IconGit size={13} /> Clone repository
              </span>
            </label>
          </div>
          {wsSrc === "local" ? (
            <>
              <input
                name="name"
                type="text"
                placeholder="Name (e.g. My App)"
                autoComplete="off"
                autoFocus
              />
              <FolderField
                name="path"
                placeholder="Folder path (e.g. ~/code/my-app)"
                resetKey={open}
              />
            </>
          ) : (
            <>
              <RepoField
                name="url"
                value={cloneUrl}
                onValue={applyCloneUrl}
                autoFocus
                placeholder="https://github.com/org/repo or git@host:org/repo.git"
              />
              <input
                name="name"
                type="text"
                placeholder="Name"
                autoComplete="off"
                value={cloneName}
                onChange={onCloneName}
              />
              <FolderField
                name="path"
                placeholder={"Destination (e.g. " + cloneParent + "/repo)"}
                resetKey={open}
                value={clonePath}
                onChange={(v) => {
                  setClonePath(v);
                  setPathDirty(true);
                }}
              />
            </>
          )}
          <input type="hidden" name="source" value={wsSrc} />
        </>
      ) : (
        <input
          name="name"
          type="text"
          placeholder="Agent name"
          autoComplete="off"
          autoFocus
        />
      )}
      {kind !== "workspace" ? (
        <ConfigFields
          catalog={catalog}
          provider={cfg.provider}
          model={cfg.model}
          thinking={cfg.thinking}
          onChange={onCfg}
          idPrefix="create"
        />
      ) : null}
      <p className="form-error" hidden={!error}>
        {error}
      </p>
      <div className="dlg-actions">
        <button
          type="button"
          className="btn btn-ghost btn-sm"
          onClick={onClose}
        >
          Cancel
        </button>
        <button
          type="submit"
          className="btn btn-primary btn-sm"
          disabled={!!busy}
        >
          {remote ? (busy ? "Cloning…" : "Clone") : "Create"}
        </button>
      </div>
    </form>
  );

  // ResponsiveDialog (ADR-0046): a centred dialog at >=720px, a bottom
  // sheet below — one tree, the primitive decides.
  return (
    <Dialog.Root
      open={!!open}
      onOpenChange={(o) => {
        if (!o) onClose();
      }}
    >
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content
          className="dlg dlg-create"
          onCloseAutoFocus={(e) => e.preventDefault()}
        >
          <Dialog.Title className="dlg-title">{title}</Dialog.Title>
          <Dialog.Description className="dlg-body">{desc}</Dialog.Description>
          {fields}
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
