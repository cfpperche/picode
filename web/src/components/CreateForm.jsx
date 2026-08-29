import * as Dialog from "@radix-ui/react-dialog";
import { Drawer } from "vaul";
import ConfigFields from "./ConfigFields.jsx";
import FolderField from "./FolderField.jsx";
import { useMedia } from "../lib/media.js";

export default function CreateForm({
  open, kind, workspaceName, catalog, cfg, onCfg, error, onSubmit, onClose,
  sessions, onAdopt, onKind,
}) {
  const desktop = useMedia("(min-width: 720px)");
  const title = kind === "workspace"
    ? "New workspace"
    : kind === "session"
      ? "From a Pi session"
      : kind === "agent"
        ? ("New agent" + (workspaceName ? " in " + workspaceName : ""))
        : "New agent";
  const sessionBody = kind === "session" ? (
    <div className="create-form">
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
        <ul className="session-pick">
          {sessions.map((s) => (
            <li key={s.path}>
              <button type="button" className="session-pick-btn" onClick={() => onAdopt && onAdopt(s.path)}>
                <span className="session-pick-name">{s.name || s.preview || "Pi session"}</span>
                {s.cwd ? <span className="session-pick-cwd">{s.cwd}</span> : null}
              </button>
            </li>
          ))}
        </ul>
      )}
      <p className="form-error" hidden={!error}>{error}</p>
      <div className="dlg-actions">
        <button type="button" className="btn btn-ghost btn-sm" onClick={onClose}>Cancel</button>
      </div>
    </div>
  ) : null;
  const fields = kind === "session" ? sessionBody : (
    <form className="form-new create-form" noValidate onSubmit={onSubmit}>
      {kind === "workspace" ? (
        <>
          <input name="name" type="text" placeholder="Name (e.g. My App)" autoComplete="off" autoFocus />
          <FolderField name="path" placeholder="Folder path (e.g. ~/code/my-app)" resetKey={open} />
        </>
      ) : kind === "free" ? (
        <>
          <input name="name" type="text" placeholder="Name" autoComplete="off" autoFocus />
          <FolderField name="path" placeholder="Folder (optional — ~/.picode/work/name)" resetKey={open} />
        </>
      ) : (
        <input name="name" type="text" placeholder="Agent name" autoComplete="off" autoFocus />
      )}
      <ConfigFields catalog={catalog} provider={cfg.provider} model={cfg.model} thinking={cfg.thinking} onChange={onCfg} idPrefix="create" />
      <p className="form-error" hidden={!error}>{error}</p>
      {onKind ? (
        <p className="dlg-body">
          <button type="button" className="btn btn-ghost btn-sm" onClick={() => onKind("session")}>From a Pi session</button>
        </p>
      ) : null}
      <div className="dlg-actions">
        <button type="button" className="btn btn-ghost btn-sm" onClick={onClose}>Cancel</button>
        <button type="submit" className="btn btn-primary btn-sm">Create</button>
      </div>
    </form>
  );

  if (desktop) {
    return (
      <Dialog.Root open={!!open} onOpenChange={(o) => { if (!o) onClose(); }}>
        <Dialog.Portal>
          <Dialog.Overlay className="dlg-overlay" />
          <Dialog.Content className="dlg dlg-create" onCloseAutoFocus={(e) => e.preventDefault()}>
            <Dialog.Title className="dlg-title">{title}</Dialog.Title>
            <Dialog.Description className="dlg-body">
              {kind === "workspace" ? "A folder plus its first agent." : kind === "session" ? "A copy. Close the other Pi if you want — we do not touch it." : "Provider, model, and thinking are required."}
            </Dialog.Description>
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
          <Drawer.Description className="dlg-body">
            {kind === "workspace" ? "A folder plus its first agent." : kind === "session" ? "A copy. Close the other Pi if you want — we do not touch it." : "Provider, model, and thinking are required."}
          </Drawer.Description>
          {fields}
        </Drawer.Content>
      </Drawer.Portal>
    </Drawer.Root>
  );
}
