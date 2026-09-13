import { useEffect, useState } from "react";
import * as Dialog from "./ResponsiveDialog.jsx";
import { api } from "@picode/shared/client/api.js";
import { SNIP_RESERVED, replaceSnipToken } from "@picode/shared/domain/snipDraft.js";
import { toastError } from "../lib/toast.js";

export default function SnipRunSheet({ open, mode, snipId, slug, draft, target, onInsert, onSend, onRan, onClose }) {
  const [snip, setSnip] = useState(null);
  const [values, setValues] = useState({});
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!open || !snipId) { setSnip(null); setValues({}); setErr(""); return; }
    let live = true;
    api("/api/snips/" + encodeURIComponent(snipId)).then((p) => {
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
  }, [open, snipId]);

  const fields = (snip && snip.placeholders || []).filter((ph) => !SNIP_RESERVED.includes(ph.name));
  const title = mode === "run" ? "Send snippet" : "Insert snippet";

  async function expand() {
    const d = await api("/api/snips/" + encodeURIComponent(snipId) + "/expand", {
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
      await api("/api/snips/" + encodeURIComponent(snipId) + "/run", {
        method: "POST", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ target, values, confirm: true }),
      });
      if (onRan) onRan();
    } catch (e) { toastError(e); setErr(e.message || "Could not send."); }
    finally { setBusy(false); }
  }

  return (
    <Dialog.Root open={!!open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg dlg-sm snip-run">
          <Dialog.Title>{snip ? snip.title : title}</Dialog.Title>
          <Dialog.Description className="sr-only">Fill snippet fields then send or insert.</Dialog.Description>
          {err ? <p className="form-error" role="alert">{err}</p> : null}
          {!snip && !err ? <div className="mcp-skel" aria-hidden="true"><span className="skel-line w-70" /><span className="skel-line w-50" /></div> : null}
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
              <button type="button" className="btn btn-primary" disabled={busy || !snip} onClick={run}>{busy ? "Sending…" : "Send snippet"}</button>
            ) : (
              <>
                <button type="button" className="btn btn-ghost" disabled={busy || !snip} onClick={insert}>Insert snippet</button>
                <button type="button" className="btn btn-primary" disabled={busy || !snip} onClick={sendComposer}>{busy ? "Sending…" : "Send snippet"}</button>
              </>
            )}
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
