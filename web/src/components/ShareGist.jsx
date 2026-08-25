import * as Dialog from "@radix-ui/react-dialog";
import { toast } from "../lib/toast.js";

export default function ShareGist({ open, gist, viewer, onClose }) {
  function copy(url) {
    navigator.clipboard.writeText(url).then(() => toast.ok("Copied link.")).catch(() => toast.error("Clipboard blocked."));
  }
  return (
    <Dialog.Root open={open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg dlg-create" onCloseAutoFocus={(e) => e.preventDefault()}>
          <Dialog.Title className="dlg-title">Shared</Dialog.Title>
          <Dialog.Description className="dlg-body">Secret gist. Anyone with the link can read it.</Dialog.Description>
          {viewer ? (
            <p className="share-link">
              <a href={viewer} target="_blank" rel="noreferrer">{viewer}</a>
              <button type="button" className="btn btn-ghost btn-sm" onClick={() => copy(viewer)}>Copy</button>
            </p>
          ) : null}
          {gist ? (
            <p className="share-link">
              <a href={gist} target="_blank" rel="noreferrer">{gist}</a>
              <button type="button" className="btn btn-ghost btn-sm" onClick={() => copy(gist)}>Copy</button>
            </p>
          ) : null}
          <div className="dlg-actions">
            <button type="button" className="btn btn-primary btn-sm" onClick={onClose}>Close</button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
