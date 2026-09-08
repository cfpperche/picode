import { useEffect, useMemo, useRef, useState } from "react";
import * as Dialog from "./ResponsiveDialog.jsx";
import { api } from "@picode/shared/client/api.js";
import { ownerBase } from "@picode/shared/domain/gitOwner.js";
import { FIELD_LABELS, LABELS, gateFor } from "@picode/shared/domain/graphActions.js";
import { useDebounced } from "../lib/useDebounced.js";

// One form for every write action the git graph offers (ADR-0096).
//
// It composes nothing. The exact command comes from the server
// (POST .../git/compose), so what the reader is promised here and what the
// terminal receives are the same string by construction — and the browser
// never builds a shell fragment of its own.
//
// Three doors, as ADR-0078 shipped them: prepare the command in a terminal
// and let the human press Enter, let PiCode press it when the interlock finds
// the repository idle, or ask an agent that lives here to do it in its own
// turn. A tier C action taken through the run door — the one place where
// nobody reads the command before it runs — asks for a typed phrase first.

const DOOR_LABELS = {
  prepare: "Prepare it in a terminal",
  run: "Run it when nobody is working here",
};

// A full object name is unreadable in a sentence; the graph shows seven
// characters everywhere else, and so does this.
function shortTarget(t) {
  const s = String(t || "");
  return /^[0-9a-f]{40,64}$/i.test(s) ? s.slice(0, 7) : s;
}

function fieldLabel(action, field) {
  const per = FIELD_LABELS[action] || {};
  return per[field] || (field === "name" ? "Name" : "Message");
}

export default function GitActionDialog({ open, owner, root, item, agents = [], run = false, onClose, onDeliver, onAsk }) {
  const [name, setName] = useState("");
  const [message, setMessage] = useState("");
  const [door, setDoor] = useState("prepare");
  const [typed, setTyped] = useState("");
  const [composed, setComposed] = useState(null);
  // The one gesture no other client can make (ADR-0096 phase 4): the worktree
  // and the agent that lives in it, from the row under the cursor.
  const [alsoAgent, setAlsoAgent] = useState(false);
  const [agentName, setAgentName] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const firstRef = useRef(null);
  const action = item ? item.action : "";
  const needs = useMemo(() => (item && item.needs) || [], [item]);
  const wantsName = needs.includes("name");
  const wantsMessage = needs.includes("message");

  useEffect(() => {
    if (!open) return;
    setName(item && item.name ? item.name : "");
    setMessage("");
    setTyped("");
    setError("");
    setComposed(null);
    setDoor(run ? "run" : "prepare");
    setAlsoAgent(false);
    setAgentName("");
  }, [open, item, run]);

  // The preview is a server answer, so it follows the fields rather than
  // leading them: a debounce keeps a typed message from asking on every key.
  const debouncedName = useDebounced(name);
  const debouncedMessage = useDebounced(message);
  useEffect(() => {
    if (!open || !action || !owner) return;
    let live = true;
    // A message is required before the server will compose; showing its
    // refusal as an error while the field is still empty would be noise.
    if ((wantsName && !debouncedName.trim()) || (wantsMessage && !debouncedMessage.trim())) {
      setComposed(null);
      return undefined;
    }
    api(ownerBase(owner) + encodeURIComponent(owner.id) + "/git/compose", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ action, target: item.target || "", name: debouncedName.trim(), message: debouncedMessage, root }),
    })
      .then((res) => { if (live) { setComposed(res); setError(""); } })
      .catch((e) => { if (live) { setComposed(null); setError(e?.message || "That command cannot be composed."); } });
    return () => { live = false; };
  }, [open, action, owner, item, root, debouncedName, debouncedMessage, wantsName, wantsMessage]);

  const asking = door.startsWith("ask:");
  const agent = asking ? agents.find((a) => a.id === door.slice(4)) : null;
  const gate = composed && !asking
    ? gateFor({ tier: composed.tier, door, action, target: item ? item.target : "", name: name.trim(), branch: composed.branch })
    : { typed: "" };
  const preview = composed ? (asking ? composed.prompt : composed.command) : "";
  const ready = !!composed && (!gate.typed || typed.trim() === gate.typed) && !busy;

  async function submit() {
    if (!ready) return;
    setBusy(true);
    try {
      if (asking && agent) await onAsk(agent, composed.prompt, action, composed.verb, composed.head || "");
      else await onDeliver(composed.command, {
        run: door === "run",
        verb: composed.verb,
        tier: composed.tier,
        // The position an undo is recorded against: read at composition,
        // not from whatever the graph showed when the menu opened.
        head: composed.head || "",
        name: name.trim(),
        alsoAgent: alsoAgent && action === "create-worktree",
        agentName: agentName.trim(),
      });
      onClose();
    } catch (e) {
      setError(e?.message || "That action could not be sent.");
    } finally {
      setBusy(false);
    }
  }

  if (!item) return null;
  const title = LABELS[action] ? LABELS[action].replace(/…$/, "") : action;
  return (
    <Dialog.Root open={!!open} onOpenChange={(o) => { if (!o && !busy) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg gg-action-dlg" onOpenAutoFocus={(e) => {
            // Only steer focus when there is a field to steer it to; an
            // action with none keeps Radix's default, which lands on the
            // first control — otherwise focus escapes to the page behind.
            if (firstRef.current) { e.preventDefault(); firstRef.current.focus(); }
          }}>
          <Dialog.Title className="dlg-title">{title}</Dialog.Title>
          <Dialog.Description className="dlg-body">
            {item.target ? <>On <code className="gg-code-inline">{shortTarget(item.target)}</code></> : <>In this repository</>}
            {composed && composed.tier === "C" ? " — this one publishes or destroys work." : "."}
          </Dialog.Description>
          <form noValidate onSubmit={(e) => { e.preventDefault(); submit(); }}>
            <fieldset className="gg-action-fields" disabled={busy}>
              {wantsName ? (
                <label className="gg-action-field">
                  <span>{fieldLabel(action, "name")}</span>
                  <input
                    ref={firstRef}
                    className="dlg-input"
                    value={name}
                    autoComplete="off"
                    maxLength={200}
                    onChange={(e) => setName(e.target.value)}
                  />
                </label>
              ) : null}
              {wantsMessage ? (
                <label className="gg-action-field">
                  <span>{fieldLabel(action, "message")}</span>
                  <input
                    ref={wantsName ? undefined : firstRef}
                    className="dlg-input"
                    value={message}
                    autoComplete="off"
                    maxLength={400}
                    placeholder="What changed, in one line"
                    onChange={(e) => setMessage(e.target.value)}
                  />
                </label>
              ) : null}
              <label className="gg-action-field">
                <span>Send through</span>
                <select className="dlg-input" value={door} onChange={(e) => { setDoor(e.target.value); setTyped(""); }}>
                  <option value="prepare">{DOOR_LABELS.prepare}</option>
                  <option value="run">{DOOR_LABELS.run}</option>
                  {agents.map((a) => (
                    <option key={a.id} value={"ask:" + a.id}>Ask {a.name}</option>
                  ))}
                </select>
              </label>
            </fieldset>
            {action === "create-worktree" ? (
              <div className="gg-action-also">
                <label className="dlg-choice">
                  <input type="checkbox" checked={alsoAgent} disabled={busy} onChange={(e) => setAlsoAgent(e.target.checked)} />
                  <span>Also start an agent in it, once the folder exists.</span>
                </label>
                {alsoAgent ? (
                  <label className="gg-action-field gg-action-indent">
                    <span>Agent name</span>
                    <input
                      className="dlg-input"
                      value={agentName}
                      autoComplete="off"
                      maxLength={80}
                      placeholder={name.trim() || "the worktree's name"}
                      disabled={busy}
                      onChange={(e) => setAgentName(e.target.value)}
                    />
                  </label>
                ) : null}
              </div>
            ) : null}
            {gate.typed ? (
              <div className="dlg-typed">
                <p className="dlg-typed-hint">
                  PiCode presses Enter on this one. Type <strong>{gate.typed}</strong> to confirm.
                </p>
                <input
                  className="dlg-input"
                  value={typed}
                  autoComplete="off"
                  aria-label={`Type ${gate.typed} to confirm`}
                  onChange={(e) => setTyped(e.target.value)}
                />
              </div>
            ) : null}
            {error ? <p className="form-error" role="alert">{error}</p> : null}
            <p className="gg-action-preview">
              {preview
                ? <code className="gg-code">{preview}</code>
                : <span className="gg-action-waiting">{wantsName || wantsMessage ? "Fill the field above to see the exact command." : "Composing…"}</span>}
            </p>
            <div className="dlg-actions">
              <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={onClose}>Cancel</button>
              <button type="submit" className={"btn btn-sm " + (composed && composed.tier === "C" ? "btn-danger" : "btn-primary")} disabled={!ready}>
                {asking ? `Ask ${agent ? agent.name : "the agent"}` : door === "run" ? "Run in terminal" : "Prepare in terminal"}
              </button>
            </div>
          </form>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
