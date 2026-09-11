import { useEffect, useState } from "react";
import { canvasNameSchema, parseForm } from "@picode/shared/contracts/schemas.js";
import * as Dialog from "../ResponsiveDialog.jsx";

// NameDialog — New canvas and Rename share one form: a name, validated by
// the Zod schema in the store's own words (never the browser's), so the
// dialog and a 400 read the same. onSubmit(name) may throw; its message
// shows under the field.
export default function NameDialog({ open, title, action, initial, onSubmit, onClose }) {
  const [name, setName] = useState(initial || "");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  useEffect(() => { if (open) { setName(initial || ""); setError(""); setBusy(false); } }, [open, initial]);
  async function submit(e) {
    e.preventDefault();
    const parsed = parseForm(canvasNameSchema, { name });
    if (!parsed.ok) { setError(parsed.error); return; }
    setBusy(true);
    try {
      await onSubmit(parsed.value.name);
      onClose();
    } catch (err) {
      setError(err && err.message ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }
  return (
    <Dialog.Root open={!!open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg dlg-canvas-name" onCloseAutoFocus={(e) => e.preventDefault()}>
          <Dialog.Title className="dlg-title">{title}</Dialog.Title>
          <Dialog.Description className="sr-only">Name for the canvas.</Dialog.Description>
          <form noValidate onSubmit={submit}>
            <input
              type="text"
              className="dlg-input"
              aria-label="Canvas name"
              placeholder="Name"
              autoFocus
              autoComplete="off"
              spellCheck={false}
              value={name}
              onChange={(e) => { setName(e.target.value); if (error) setError(""); }}
            />
            {error ? <p className="form-error cv-form-error" role="alert">{error}</p> : null}
            <div className="dlg-actions">
              <Dialog.Close asChild>
                <button type="button" className="btn btn-ghost btn-sm">Cancel</button>
              </Dialog.Close>
              <button type="submit" className="btn btn-primary btn-sm" disabled={busy}>{busy ? "Saving…" : action}</button>
            </div>
          </form>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
