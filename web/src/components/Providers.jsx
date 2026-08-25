import { useState } from "react";
import * as Dialog from "@radix-ui/react-dialog";
import PageFrame from "./PageFrame.jsx";
import { api } from "../lib/api.js";
import { toast, toastError } from "../lib/toast.js";
import { apiKeySchema, parseForm } from "../lib/schemas.js";

export default function Providers({ hidden, catalog, onSignOut, onRefresh }) {
  const list = catalog && catalog.providers ? catalog.providers : [];
  const [pick, setPick] = useState("");
  const [key, setKey] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);

  async function save(e) {
    e.preventDefault();
    const parsed = parseForm(apiKeySchema, { key });
    if (!parsed.ok) { setErr(parsed.error); return; }
    setBusy(true);
    setErr("");
    try {
      await api("/api/providers/" + encodeURIComponent(pick), {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ key: parsed.value.key }),
      });
      setKey("");
      setPick("");
      if (onRefresh) await onRefresh();
      toast.ok("Signed in to " + pick + ".");
    } catch (ex) {
      toastError(ex);
    } finally {
      setBusy(false);
    }
  }

  return (
    <PageFrame id="providers-view" title="Providers" hidden={hidden}>
      <p className="settings-desc">API keys are stored in pi's auth.json. Subscriptions (Codex, Claude Pro, xAI, OpenRouter OAuth) still use /login in the terminal.</p>
      <section className="settings-section">
        {list.length === 0 ? (
          <p className="side-empty">No providers in the catalog yet.</p>
        ) : (
          <ul className="prov-list">
            {list.map((p) => (
              <li key={p.id} className="prov-row">
                <span className="prov-id">{p.id}</span>
                <span className={"prov-auth" + (p.signedIn ? " in" : "")}>{p.signedIn ? "signed in" : "not signed in"}</span>
                {p.signedIn
                  ? <button type="button" className="btn btn-ghost btn-sm" onClick={() => onSignOut && onSignOut(p.id)}>Sign out</button>
                  : <button type="button" className="btn btn-ghost btn-sm" onClick={() => { setPick(p.id); setKey(""); setErr(""); }}>API key</button>}
              </li>
            ))}
          </ul>
        )}
      </section>

      <Dialog.Root open={!!pick} onOpenChange={(o) => { if (!o) { setPick(""); setKey(""); setErr(""); } }}>
        <Dialog.Portal>
          <Dialog.Overlay className="dlg-overlay" />
          <Dialog.Content className="dlg" onCloseAutoFocus={(e) => e.preventDefault()}>
            <Dialog.Title className="dlg-title">API key · {pick}</Dialog.Title>
            <Dialog.Description className="dlg-body">Saved in ~/.pi/agent/auth.json. The key is never shown again.</Dialog.Description>
            <form className="form-new" noValidate onSubmit={save}>
              <input
                type="password"
                autoComplete="off"
                placeholder="sk-…"
                value={key}
                onChange={(e) => setKey(e.target.value)}
              />
              <p className="form-error" hidden={!err}>{err}</p>
              <div className="dlg-actions">
                <button type="button" className="btn btn-ghost btn-sm" onClick={() => setPick("")}>Cancel</button>
                <button type="submit" className="btn btn-primary btn-sm" disabled={busy}>Save</button>
              </div>
            </form>
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>
    </PageFrame>
  );
}
