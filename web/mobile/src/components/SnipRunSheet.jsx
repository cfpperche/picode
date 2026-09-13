import { useEffect, useState } from "react";
import * as Dialog from "./MobileSheet.jsx";
import { api } from "@picode/shared/client/api.js";
import { SNIP_RESERVED, replaceSnipToken } from "@picode/shared/domain/snipDraft.js";
import { toast, toastError } from "../lib/toast.js";

// The mobile twin of the desktop fill sheet (ADR-0130): same doors, same
// labels, MobileSheet at every width (ADR-0072). Composer mode adds
// Insert, which splices the draft and sends nothing. A Command snippet
// stops at one confirm that shows the exact command.
export default function SnipRunSheet({ open, mode, snipId, slug, draft, target, targetName, onlyKind, onInsert, onSend, onRan, onClose }) {
  const [snip, setSnip] = useState(null);
  const [choices, setChoices] = useState(null);
  const [pickedId, setPickedId] = useState("");
  const [values, setValues] = useState({});
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [confirming, setConfirming] = useState(null);

  const effectiveId = snipId || pickedId;
  const isTerm = !!(target && target.type === "terminal");
  const isShell = !!((snip && snip.kind === "shell") || onlyKind === "shell");
  const primary = mode === "run" ? (isTerm ? (isShell ? "Run command" : "Send to terminal") : "Send snippet") : "Send snippet";

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

  async function deliver() {
    await api("/api/snips/" + encodeURIComponent(effectiveId) + "/run", {
      method: "POST", headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ target, values, confirm: true }),
    });
    toast.ok(isTerm ? (isShell ? "Command sent to the terminal." : "Sent to the terminal.") : "Sent.");
    if (onRan) onRan();
  }

  async function beginRun() {
    setBusy(true); setErr("");
    try {
      if (isShell) {
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

  async function run() {
    setBusy(true); setErr("");
    try {
      await deliver();
    } catch (e) { setErr(e.message || "Could not send."); }
    finally { setBusy(false); }
  }

  const noChoices = !snipId && choices !== null && choices.length === 0;

  return (
    <Dialog.Root open={!!open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg m-snip-run">
          <Dialog.Title className="dlg-title">{snip ? snip.title : mode === "run" ? primary : "Insert snippet"}</Dialog.Title>
          {err ? <p className="form-error" role="alert">{err}</p> : null}
          {confirming !== null ? (
            <>
              <p className="m-term-actions-name">Run in {targetName || "this terminal"}:</p>
              <pre className="m-snip-preview">{confirming}</pre>
              <p className="m-snip-note">PiCode does not quote values — quote in the snippet if the shell should see one word.</p>
              <div className="m-snip-actions">
                <button type="button" className="btn" onClick={() => setConfirming(null)}>Back</button>
                <button type="button" className="btn btn-primary" disabled={busy} onClick={run}>{busy ? "Running…" : "Run command"}</button>
              </div>
            </>
          ) : (
            <>
              {noChoices ? <p className="m-snip-note">No snippets yet.</p> : null}
              {!snipId && choices && choices.length > 0 ? (
                <label className="m-snip-field">
                  <span>Snippet</span>
                  <select className="dlg-input" value={pickedId} onChange={(e) => { setPickedId(e.target.value); setErr(""); }}>
                    <option value="">Choose…</option>
                    {choices.map((c) => <option key={c.id} value={c.id}>{c.title}{!onlyKind && c.kind === "shell" ? " · Command" : ""}</option>)}
                  </select>
                </label>
              ) : null}
              {snip && fields.length === 0 ? <p className="m-snip-note">No fields to fill.</p> : null}
              {fields.map((ph) => (
                <label key={ph.name} className="m-snip-field">
                  <span>{ph.name}</span>
                  {ph.enum && ph.enum.length ? (
                    <select className="dlg-input" value={values[ph.name] || ""} onChange={(e) => setValues((v) => ({ ...v, [ph.name]: e.target.value }))}>
                      <option value="">{ph.optional ? "Default" : "Choose…"}</option>
                      {ph.enum.map((opt) => <option key={opt} value={opt}>{opt}</option>)}
                    </select>
                  ) : (
                    <input className="dlg-input" value={values[ph.name] || ""} onChange={(e) => setValues((v) => ({ ...v, [ph.name]: e.target.value }))} placeholder={ph.default || ""} />
                  )}
                </label>
              ))}
              <div className="m-snip-actions">
                <button type="button" className="btn" onClick={onClose}>Cancel</button>
                {mode === "run" ? (
                  <button type="button" className="btn btn-primary" disabled={busy || !effectiveId} onClick={beginRun}>{busy ? "Sending…" : isShell ? "Run command…" : primary}</button>
                ) : (
                  <>
                    <button type="button" className="btn" disabled={busy || !effectiveId} onClick={async () => {
                      setBusy(true); setErr("");
                      try {
                        const text = await expand();
                        onInsert(replaceSnipToken(draft, slug || (snip && snip.slug), text));
                      } catch (e) { setErr(e.message || "Could not expand."); }
                      finally { setBusy(false); }
                    }}>Insert</button>
                    <button type="button" className="btn btn-primary" disabled={busy || !effectiveId} onClick={async () => {
                      setBusy(true); setErr("");
                      try {
                        const text = await expand();
                        await onSend(replaceSnipToken(draft, slug || (snip && snip.slug), text));
                      } catch (e) { setErr(e.message || "Could not send."); }
                      finally { setBusy(false); }
                    }}>{busy ? "Sending…" : "Send snippet"}</button>
                  </>
                )}
              </div>
            </>
          )}
          <Dialog.Close asChild><button type="button" className="btn btn-sm m-snip-done">Done</button></Dialog.Close>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
