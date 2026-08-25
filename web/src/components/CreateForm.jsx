import * as Dialog from "@radix-ui/react-dialog";
import { Drawer } from "vaul";
import ConfigFields from "./ConfigFields.jsx";
import { useMedia } from "../lib/media.js";

export default function CreateForm({
  open, kind, workspaceName, catalog, cfg, onCfg, error, onSubmit, onClose,
}) {
  const desktop = useMedia("(min-width: 720px)");
  const title = kind === "workspace"
    ? "New workspace"
    : kind === "agent"
      ? ("New agent" + (workspaceName ? " in " + workspaceName : ""))
      : "New agent";
  const fields = (
    <form className="form-new create-form" onSubmit={onSubmit}>
      {kind === "workspace" ? (
        <>
          <input name="name" type="text" placeholder="Name (e.g. My App)" autoComplete="off" autoFocus />
          <input name="path" type="text" placeholder="Folder path (e.g. ~/code/my-app)" autoComplete="off" />
        </>
      ) : kind === "free" ? (
        <>
          <input name="name" type="text" placeholder="Name" autoComplete="off" autoFocus />
          <input name="path" type="text" placeholder="Folder (optional — ~/.picode/work/name)" autoComplete="off" />
        </>
      ) : (
        <input name="name" type="text" placeholder="Agent name" autoComplete="off" autoFocus />
      )}
      <ConfigFields catalog={catalog} provider={cfg.provider} model={cfg.model} thinking={cfg.thinking} onChange={onCfg} idPrefix="create" />
      <p className="form-error" hidden={!error}>{error}</p>
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
              {kind === "workspace" ? "A folder plus its first agent." : "Provider, model, and thinking are required."}
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
            {kind === "workspace" ? "A folder plus its first agent." : "Provider, model, and thinking are required."}
          </Drawer.Description>
          {fields}
        </Drawer.Content>
      </Drawer.Portal>
    </Drawer.Root>
  );
}
