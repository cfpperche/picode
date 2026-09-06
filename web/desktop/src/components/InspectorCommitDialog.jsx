import { useEffect, useRef, useState } from "react";
import * as Dialog from "./ResponsiveDialog.jsx";
import { commitMessageSchema, parseForm } from "@picode/shared/contracts/schemas.js";
import { gitActionCommand } from "../lib/inspector.js";

// The commit form of the Inspector's Git menu (ADR-0078). It never runs
// git: it composes `git add -A && git commit -m '…'` (and a push) from a
// one-line, Zod-validated message and hands the exact command to the
// owner's terminal, where the human reads it and presses Enter.
export default function InspectorCommitDialog({ open, push, status, run = false, onClose, onPrepare }) {
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const inputRef = useRef(null);
  useEffect(() => {
    if (open) {
      setMessage("");
      setError("");
    }
  }, [open]);
  const branch = (status && status.branch) || "";
  const upstream = (status && status.upstream) || "";
  const action = push ? "commit-push" : "commit";
  const preview = gitActionCommand(action, { branch, upstream, message: message.trim() || "…" });
  function submit() {
    const parsed = parseForm(commitMessageSchema, { message });
    if (!parsed.ok) {
      setError(parsed.error);
      return;
    }
    onPrepare(gitActionCommand(action, { branch, upstream, message: parsed.value.message }));
  }
  return (
    <Dialog.Root open={!!open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg insp-commit-dlg" onOpenAutoFocus={(e) => { e.preventDefault(); inputRef.current?.focus(); }}>
          <Dialog.Title className="dlg-title">{push ? "Commit and push" : "Commit"}</Dialog.Title>
          <Dialog.Description className="dlg-body">{run ? "The command runs in your terminal when no agent is working here; otherwise it is prepared for you to press Enter." : "The command opens in your terminal with this message; you press Enter to run it."}</Dialog.Description>
          <form noValidate onSubmit={(e) => { e.preventDefault(); submit(); }}>
            <input
              ref={inputRef}
              className="dlg-input"
              placeholder="What changed, in one line"
              value={message}
              maxLength={400}
              aria-label="Commit message"
              aria-invalid={error ? "true" : undefined}
              aria-describedby={error ? "insp-commit-error" : undefined}
              onChange={(e) => { setMessage(e.target.value); if (error) setError(""); }}
            />
            {error ? <p id="insp-commit-error" className="form-error insp-commit-error" role="alert">{error}</p> : null}
            <p className="insp-commit-preview"><code className="insp-code">{preview}</code></p>
            <div className="dlg-actions">
              <button type="button" className="btn btn-ghost btn-sm" onClick={onClose}>Cancel</button>
              <button type="submit" className="btn btn-primary btn-sm">{run ? "Run in terminal" : "Prepare in terminal"}</button>
            </div>
          </form>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
