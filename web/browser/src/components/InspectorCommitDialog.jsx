import { useEffect, useRef, useState } from "react";
import * as Dialog from "./ResponsiveDialog.jsx";
import { commitMessageSchema, parseForm } from "@picode/shared/contracts/schemas.js";
import { askGitPrompt, gitActionCommand } from "../lib/inspector.js";

// The commit form of the Inspector's Git menu (ADR-0078). It never runs
// git: it composes `git add -A && git commit -m '…'` (and a push) from a
// one-line, Zod-validated message and hands the exact command to the
// owner's terminal, where the human reads it and presses Enter. With `ask`
// set (stage 3) the same form addresses a running agent instead: the
// message becomes optional — left empty, the agent writes it from the
// changes — and the preview shows the prompt the agent will receive.
export default function InspectorCommitDialog({ open, push, status, run = false, ask = null, root = "", onClose, onPrepare, onAsk }) {
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
  const asking = !!(ask && ask.id);
  const preview = asking
    ? askGitPrompt(action, { root, branch, upstream, message: message.trim() })
    : gitActionCommand(action, { branch, upstream, message: message.trim() || "…" });
  function submit() {
    if (asking && !message.trim()) {
      onAsk(askGitPrompt(action, { root, branch, upstream }), action);
      return;
    }
    const parsed = parseForm(commitMessageSchema, { message });
    if (!parsed.ok) {
      setError(parsed.error);
      return;
    }
    if (asking) {
      onAsk(askGitPrompt(action, { root, branch, upstream, message: parsed.value.message }), action);
      return;
    }
    onPrepare(gitActionCommand(action, { branch, upstream, message: parsed.value.message }));
  }
  const verb = push ? "commit and push" : "commit";
  const title = asking ? `Ask ${ask.name} to ${verb}` : push ? "Commit and push" : "Commit";
  const lead = asking
    ? `${ask.name} does it in its own turn. Add a message, or leave it empty and ${ask.name} writes one from the changes.`
    : run
      ? "The command runs in your terminal when no agent is working here; otherwise it is prepared for you to press Enter."
      : "The command opens in your terminal with this message; you press Enter to run it.";
  return (
    <Dialog.Root open={!!open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg insp-commit-dlg" onOpenAutoFocus={(e) => { e.preventDefault(); inputRef.current?.focus(); }}>
          <Dialog.Title className="dlg-title">{title}</Dialog.Title>
          <Dialog.Description className="dlg-body">{lead}</Dialog.Description>
          <form noValidate onSubmit={(e) => { e.preventDefault(); submit(); }}>
            <input
              ref={inputRef}
              className="dlg-input"
              placeholder={asking ? "Optional: the message you want" : "What changed, in one line"}
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
              <button type="submit" className="btn btn-primary btn-sm">{asking ? `Ask ${ask.name}` : run ? "Run in terminal" : "Prepare in terminal"}</button>
            </div>
          </form>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
