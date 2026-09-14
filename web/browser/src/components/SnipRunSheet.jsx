import { useEffect, useState } from "react";
import * as Dialog from "./ResponsiveDialog.jsx";
import { api } from "@picode/shared/client/api.js";
import { SNIP_RESERVED, replaceSnipToken } from "@picode/shared/domain/snipDraft.js";
import { toast, toastError } from "../lib/toast.js";

// The fill sheet behind every snippet invoke (ADR-0130). Doors label
// themselves (K13): managed agent → "Send snippet" (SendTurn); terminal
// or interactive TUI (`via: "tui"`) → "Send to terminal" (paste — the UI
// says "Sent to the terminal", never "the model saw it"); a Command
// snippet → "Run command", which always stops at one confirm that names
// the terminal and shows the expanded command (K14: a UI invariant, not
// authz). Composer mode adds Insert, which splices the draft and sends
// nothing.
export default function SnipRunSheet({ open, mode, snipId, slug, draft, target, targetName, onlyKind, via, onInsert, onSend, onRan, onClose }) {
  const [snip, setSnip] = useState(null);
  const [choices, setChoices] = useState(null);
  const [pickedId, setPickedId] = useState("");
  const [values, setValues] = useState({});
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [confirming, setConfirming] = useState(null); // shell kind: the expanded text awaiting the explicit Run

  const effectiveId = snipId || pickedId;
  const isTerm = !!(target && target.type === "terminal");
  const isTui = via === "tui";
  const isShell = !!((snip && snip.kind === "shell") || onlyKind === "shell");
  const terminalCopy = isTerm || isTui;
  const primary = mode === "run" ? (terminalCopy ? (isShell ? "Run command" : "Send to terminal") : "Send snippet") : "Send snippet";

  useEffect(() => {
    if (!open) { setSnip(null); setChoices(null); setPickedId(""); setValues({}); setErr(""); setConfirming(null); return; }
    if (snipId) return;
    let live = true;
    api("/api/snips/picker" + (mode === "run" ? "?shell=1" : "")).then((d) => {
      if (!live) return;
      const all = d.snips || [];
      setChoices(onlyKind ? all.filter((c) => c.kind === onlyKind) : all);
    }).catch(() => { if (live) setChoices([]); });
    return () => { live = false; };
  }, [open, snipId, mode, onlyKind]);

  useEffect(() => {
    if (!open || !effectiveId) { setSnip(null); setValues({}); return; }
    let live = true;
    api("/api/snips/" + encodeURIComponent(effectiveId)).then((p) => {
      if (!live) return;
      setSnip(p);
      const next = {};
      for (const ph of p.placeholders || []) {
        if (SNIP_RESERVED.includes(ph.name)) continue;
        next[ph.name] = ph.default || "";
      }
      setValues(next);
    }).catch((e) => { if (live) setErr(e.message || "Could not load snippet."); });
    return () => { live = false; };
  }, [open, effectiveId]);

  const fields = ((snip && snip.placeholders) || []).filter((ph) => !SNIP_RESERVED.includes(ph.name));

  async function expand() {
    const d = await api("/api/snips/" + encodeURIComponent(effectiveId) + "/expand", {
      method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ values }),
    });
    if (d.missing && d.missing.length) {
      const userMiss = d.missing.filter((n) => !SNIP_RESERVED.includes(n));
      if (userMiss.length) throw new Error("Fill in " + userMiss.join(", ") + ".");
    }
    return d.text || "";
  }

  async function insert() {
    setBusy(true); setErr("");
    try {
      const text = await expand();
      onInsert(replaceSnipToken(draft, slug || (snip && snip.slug), text));
    } catch (e) { setErr(e.message || "Could not expand."); }
    finally { setBusy(false); }
  }

  async function sendComposer() {
    setBusy(true); setErr("");
    try {
      const text = await expand();
      onSend(replaceSnipToken(draft, slug || (snip && snip.slug), text));
    } catch (e) { setErr(e.message || "Could not send."); }
    finally { setBusy(false); }
  }

  async function beginRun() {
    setBusy(true); setErr("");
    try {
      if (isShell) {
        // The confirm shows exactly what would run: live context, gates
        // included, nothing delivered.
        const d = await api("/api/snips/" + encodeURIComponent(effectiveId) + "/run", {
          method: "POST", headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ target, values, confirm: true, preview: true }),
        });
        setConfirming(d.text || "");
        return;
      }
      const text = await expand();
      await deliver();
    } catch (e) { setErr(e.message || "Could not send."); }
    finally { setBusy(false); }
  }

  async function deliver() {
    await api("/api/snips/" + encodeURIComponent(effectiveId) + "/run", {
      method: "POST", headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ target, values, confirm: true }),
    });
    toast.ok(terminalCopy ? (isShell ? "Command sent to the terminal." : "Sent to the terminal.") : "Sent.");
    if (onRan) onRan();
  }

  async function run() {
    setBusy(true); setErr("");
    try {
      await deliver();
    } catch (e) {
      const msg = e.message || "Could not send.";
      setErr(msg);
      toastError(e);
    }
    finally { setBusy(false); }
  }

  const noChoices = !snipId && choices !== null && choices.length === 0;

  return (
    <Dialog.Root open={!!open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg dlg-sm snip-run">
          <Dialog.Title>{snip ? snip.title : mode === "run" ? primary : "Insert snippet"}</Dialog.Title>
          <Dialog.Description className="sr-only">Fill snippet fields then send or insert.</Dialog.Description>
          {err ? <p className="form-error" role="alert">{err}</p> : null}
          {confirming !== null ? (
            <>
              <p className="auto-hint">
                Run in <strong>{targetName || "this terminal"}</strong>:
              </p>
              <pre className="auto-prompt snip-preview">{confirming}</pre>
              <p className="auto-hint">PiCode does not quote values — quote in the snippet if the shell should see one word.</p>
              <div className="dlg-actions" data-align-row>
                <button type="button" className="btn btn-ghost" onClick={() => setConfirming(null)}>Back</button>
                <button type="button" className="btn btn-primary" disabled={busy} onClick={run}>{busy ? "Running…" : "Run command"}</button>
              </div>
            </>
          ) : (
            <>
              {!snipId && choices === null && !err ? (
                <div className="mcp-skel" aria-hidden="true"><span className="skel-line w-70" /><span className="skel-line w-50" /></div>
              ) : null}
              {noChoices ? (
                <p className="auto-hint">No snippets yet.</p>
              ) : null}
              {!snipId && choices && choices.length > 0 ? (
                <label className="auto-field">
                  <span>Snippet</span>
                  <select className="auto-select" value={pickedId} onChange={(e) => { setPickedId(e.target.value); setErr(""); }}>
                    <option value="">Choose…</option>
                    {choices.map((c) => <option key={c.id} value={c.id}>{c.title}{!onlyKind && c.kind === "shell" ? " · Command" : ""}</option>)}
                  </select>
                </label>
              ) : null}
              {snip && fields.length === 0 ? <p className="auto-hint">No fields to fill.</p> : null}
              {fields.map((ph) => (
                <label key={ph.name} className="auto-field">
                  <span>{ph.name}</span>
                  {ph.enum && ph.enum.length ? (
                    <select className="auto-select" value={values[ph.name] || ""} onChange={(e) => setValues((v) => ({ ...v, [ph.name]: e.target.value }))}>
                      <option value="">{ph.optional ? "Default" : "Choose…"}</option>
                      {ph.enum.map((opt) => <option key={opt} value={opt}>{opt}</option>)}
                    </select>
                  ) : (
                    <input className="dlg-input" value={values[ph.name] || ""} onChange={(e) => setValues((v) => ({ ...v, [ph.name]: e.target.value }))} placeholder={ph.default || ""} />
                  )}
                </label>
              ))}
              <div className="dlg-actions" data-align-row>
                <button type="button" className="btn btn-ghost" onClick={onClose}>Cancel</button>
                {mode === "run" ? (
                  <button type="button" className="btn btn-primary" disabled={busy || !effectiveId} onClick={beginRun}>{busy ? "Sending…" : isShell ? "Run command…" : primary}</button>
                ) : (
                  <>
                    <button type="button" className="btn btn-ghost" disabled={busy || !effectiveId} onClick={insert}>Insert snippet</button>
                    <button type="button" className="btn btn-primary" disabled={busy || !effectiveId} onClick={sendComposer}>{busy ? "Sending…" : primary}</button>
                  </>
                )}
              </div>
            </>
          )}
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
