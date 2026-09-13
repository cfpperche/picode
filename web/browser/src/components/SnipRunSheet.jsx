import { useEffect, useState } from "react";
import * as Dialog from "./ResponsiveDialog.jsx";
import { api } from "@picode/shared/client/api.js";
import { SNIP_RESERVED, replaceSnipToken } from "@picode/shared/domain/snipDraft.js";
import { toast, toastError } from "../lib/toast.js";

// The fill sheet behind every snippet invoke (ADR-0130). Two doors, two
// labels (K13): an agent target is "Send snippet" (SendTurn); a terminal
// target is "Send to terminal" (ADR-0089 paste — the UI says "Sent to the
// terminal", never "the model saw it"). Composer mode adds Insert, which
// splices the draft and sends nothing.
export default function SnipRunSheet({ open, mode, snipId, slug, draft, target, onInsert, onSend, onRan, onClose }) {
  const [snip, setSnip] = useState(null);
  const [choices, setChoices] = useState(null);
  const [pickedId, setPickedId] = useState("");
  const [values, setValues] = useState({});
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);

  const effectiveId = snipId || pickedId;
  const isTerm = !!(target && target.type === "terminal");
  const primary = mode === "run" ? (isTerm ? "Send to terminal" : "Send snippet") : "Send snippet";

  useEffect(() => {
    if (!open) { setSnip(null); setChoices(null); setPickedId(""); setValues({}); setErr(""); return; }
    if (snipId) return;
    let live = true;
    api("/api/snips/picker").then((d) => { if (live) setChoices(d.snips || []); }).catch(() => { if (live) setChoices([]); });
    return () => { live = false; };
  }, [open, snipId]);

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

  async function run() {
    setBusy(true); setErr("");
    try {
      await api("/api/snips/" + encodeURIComponent(effectiveId) + "/run", {
        method: "POST", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ target, values, confirm: true }),
      });
      toast.ok(isTerm ? "Sent to the terminal." : "Sent.");
      if (onRan) onRan();
    } catch (e) {
      const msg = e.message || "Could not send.";
      setErr(msg);
      if (/running as a command|shell prompt/i.test(msg)) toast.info("The CLI left a shell here — a prompt would run as a command.");
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
                {choices.map((c) => <option key={c.id} value={c.id}>{c.title}</option>)}
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
              <button type="button" className="btn btn-primary" disabled={busy || !effectiveId} onClick={run}>{busy ? "Sending…" : primary}</button>
            ) : (
              <>
                <button type="button" className="btn btn-ghost" disabled={busy || !effectiveId} onClick={insert}>Insert snippet</button>
                <button type="button" className="btn btn-primary" disabled={busy || !effectiveId} onClick={sendComposer}>{busy ? "Sending…" : primary}</button>
              </>
            )}
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
