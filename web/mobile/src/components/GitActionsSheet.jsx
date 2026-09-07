import { useRef, useState } from "react";
import * as Sheet from "./MobileSheet.jsx";
import { commitMessageSchema, parseForm } from "@picode/shared/contracts/schemas.js";
import { gitActions, gitActionCommand, askGitPrompt, askedNote, askChannelHint } from "../lib/git/actions.js";
import { ACTION_LABELS } from "../lib/git/model.js";

// Every action is reviewable before it enters the established terminal or
// agent channel. The host callback owns terminal selection and server guards.
export default function GitActionsSheet({ open, owner, root, status, agents = [], blocked, onClose, onOpenTerminal, onAskAgent }) {
  const [action, setAction] = useState("");
  const [mode, setMode] = useState("prepare");
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState("");
  const submitting = useRef(false);
  const offered = gitActions(status);
  if (offered.length && !status?.detached) offered.push("pr");
  const agent = agents.find(a => mode === "ask:" + a.id);
  const commit = action === "commit" || action === "commit-push";
  const options = { root, branch: status?.branch, upstream: status?.upstream, message: message.trim() };
  const preview = action ? (agent ? askGitPrompt(action, options) : gitActionCommand(action, { ...options, message: message.trim() || "…" })) : "";
  const unavailable = blocked || !root || !offered.includes(action) || (mode.startsWith("ask:") && !agent);
  const close = () => { if (!submitting.current) { setError(""); setNotice(""); onClose(); } };

  async function submit(e) {
    e.preventDefault();
    if (submitting.current || unavailable) return;
    let cleanMessage = message.trim();
    if (commit && (!agent || cleanMessage)) {
      const parsed = parseForm(commitMessageSchema, { message });
      if (!parsed.ok) { setError(parsed.error); return; }
      cleanMessage = parsed.value.message;
    }
    submitting.current = true;
    setBusy(true); setError(""); setNotice("");
    try {
      const args = { ...options, message: cleanMessage };
      if (agent) {
        if (!onAskAgent) throw new Error("The agent channel is unavailable.");
        const result = await onAskAgent(agent, askGitPrompt(action, args), root, action);
        setNotice(askedNote(agent.name, action, result));
      } else {
        if (!onOpenTerminal) throw new Error("The terminal channel is unavailable.");
        await onOpenTerminal(owner, root, gitActionCommand(action, args), { run: mode === "run" });
        setNotice("Terminal ready.");
      }
    } catch (e) {
      setError(e?.message || "Could not send this action. Try again.");
    } finally {
      submitting.current = false; setBusy(false);
    }
  }

  return <Sheet.Root open={!!open} onOpenChange={value => { if (!value) close(); }}>
    <Sheet.Portal><Sheet.Overlay className="dlg-overlay" /><Sheet.Content className="dlg m-git-sheet" aria-describedby="m-git-action-description">
      <div className="m-git-sheet-heading"><Sheet.Title className="dlg-title">{action ? ACTION_LABELS[action] : "Git actions"}</Sheet.Title><button type="button" className="btn btn-ghost" disabled={busy} onClick={close}>Close</button></div>
      <Sheet.Description id="m-git-action-description" className="dlg-body">{status?.branch || "This repository"}</Sheet.Description>
      {notice ? <div className="m-git-state" role="status"><p>{notice}</p><button type="button" className="btn btn-primary" onClick={close}>Done</button></div> : !action ? <div className="m-git-action-list">
        {offered.length ? offered.map(id => <button type="button" className="m-git-action" key={id} onClick={() => { setAction(id); setError(""); }}>{ACTION_LABELS[id]}<span aria-hidden="true">›</span></button>) : <div className="m-git-state"><p>No Git actions are available for this folder.</p><button className="btn" type="button" onClick={close}>Back</button></div>}
      </div> : <form noValidate onSubmit={submit}>
        <fieldset disabled={busy} className="m-git-action-fields">
          <label className="m-git-field"><span>Send through</span><select value={mode} onChange={e => { setMode(e.target.value); setError(""); }}>
            <option value="prepare">Prepare in terminal</option><option value="run">Run when idle</option>
            {agents.map(a => <option key={a.id} value={"ask:" + a.id}>Ask {a.name}{askChannelHint(a) ? " · " + askChannelHint(a) : ""}</option>)}
          </select></label>
          <p className="m-git-hint">{agent ? `${agent.name} receives this request in its own turn.` : mode === "run" ? "Runs when no agent is working here; otherwise it is prepared for you." : "The command waits in your terminal for you to press Enter."}</p>
          {commit ? <label className="m-git-field"><span>Commit message{agent ? " (optional)" : ""}</span><input autoComplete="off" value={message} onChange={e => { setMessage(e.target.value); setError(""); }} placeholder={agent ? "Let the agent write it" : "What changed, in one line"} aria-invalid={!!error} /></label> : null}
          <details className="m-git-preview"><summary>{agent ? "Review request" : "Review command"}</summary><pre>{preview}</pre></details>
        </fieldset>
        {blocked ? <p className="form-error" role="alert">This folder changed. Close this sheet and follow the working folder before continuing.</p> : null}
        {mode.startsWith("ask:") && !agent ? <p className="form-error" role="alert">That agent is no longer available. Choose a terminal or another agent.</p> : null}
        {busy ? <p className="m-git-pending" role="status"><span aria-hidden="true" />Sending this action…</p> : null}
        {error ? <p className="form-error" role="alert">{error}</p> : null}
        <div className="dlg-actions" data-align-row>
          <button type="button" className="btn btn-ghost" disabled={busy} onClick={() => { setAction(""); setError(""); }}>Back</button>
          <button type="submit" className="btn btn-primary" disabled={busy || unavailable}>{busy ? "Sending…" : agent ? `Ask ${agent.name}` : mode === "run" ? "Run when idle" : "Prepare"}</button>
        </div>
      </form>}
    </Sheet.Content></Sheet.Portal>
  </Sheet.Root>;
}
