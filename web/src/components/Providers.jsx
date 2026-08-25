import { useEffect, useMemo, useState } from "react";
import * as Dialog from "@radix-ui/react-dialog";
import { Command } from "cmdk";
import PageFrame from "./PageFrame.jsx";
import { api } from "../lib/api.js";
import { toast, toastError } from "../lib/toast.js";
import { apiKeySchema, parseForm } from "../lib/schemas.js";
import { go } from "../lib/routes.js";

export default function Providers({ hidden, catalog, onSignOut, onRefresh, wantAdd }) {
  const list = catalog && catalog.providers ? catalog.providers : [];
  const signed = list.filter((p) => p.signedIn);
  const [add, setAdd] = useState(false);
  const [step, setStep] = useState("pick"); // pick | method | key | oauth
  const [pick, setPick] = useState(null);
  const [key, setKey] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (hidden || !wantAdd) return;
    openAdd();
  }, [hidden, wantAdd]);

  const available = useMemo(
    () => list.filter((p) => !p.signedIn).sort((a, b) => a.id.localeCompare(b.id)),
    [list],
  );

  function openAdd() {
    setPick(null);
    setStep("pick");
    setKey("");
    setErr("");
    setAdd(true);
  }

  function closeAdd() {
    setAdd(false);
    setPick(null);
    setKey("");
    setErr("");
    if (wantAdd) go("providers");
  }

  function chooseProvider(p) {
    setPick(p);
    setErr("");
    if (p.login === "oauth") setStep("oauth");
    else if (p.login === "both") setStep("method");
    else setStep("key");
  }

  async function save(e) {
    e.preventDefault();
    const parsed = parseForm(apiKeySchema, { key });
    if (!parsed.ok) { setErr(parsed.error); return; }
    setBusy(true);
    setErr("");
    try {
      await api("/api/providers/" + encodeURIComponent(pick.id), {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ key: parsed.value.key }),
      });
      toast.ok("Signed in to " + pick.id + ".");
      closeAdd();
      if (onRefresh) await onRefresh();
    } catch (ex) {
      toastError(ex);
    } finally {
      setBusy(false);
    }
  }

  const title = !pick ? "Add provider" : step === "method" ? pick.id : step === "oauth" ? pick.id : "API key · " + pick.id;

  return (
    <PageFrame id="providers-view" title="Providers" hidden={hidden}>
      <p className="settings-desc">Credentials stay in pi's auth.json. API keys are saved here. Account / subscription login is still the TUI until pi adds RPC login.</p>
      <div className="set-row" style={{ marginBottom: 12 }}>
        <span />
        <button type="button" className="btn btn-primary btn-sm" onClick={openAdd}>Add provider</button>
      </div>
      <section className="settings-section">
        {signed.length === 0 ? (
          <p className="side-empty">No providers yet. Add a provider to sign in.</p>
        ) : (
          <ul className="prov-list">
            {signed.map((p) => (
              <li key={p.id} className="prov-row">
                <span className="prov-id">{p.id}</span>
                <span className="prov-auth in">{p.authType === "oauth" ? "account" : "api key"}</span>
                {p.login !== "oauth" ? (
                  <button type="button" className="btn btn-ghost btn-sm" onClick={() => { setPick(p); setStep("key"); setKey(""); setErr(""); setAdd(true); }}>Replace</button>
                ) : null}
                <button type="button" className="btn btn-ghost btn-sm" onClick={() => onSignOut && onSignOut(p.id)}>Sign out</button>
              </li>
            ))}
          </ul>
        )}
      </section>

      <Dialog.Root open={add} onOpenChange={(o) => { if (!o) closeAdd(); }}>
        <Dialog.Portal>
          <Dialog.Overlay className="dlg-overlay" />
          <Dialog.Content className="dlg dlg-create" onCloseAutoFocus={(e) => e.preventDefault()}>
            <Dialog.Title className="dlg-title">{title}</Dialog.Title>
            <Dialog.Description className="dlg-body">
              {step === "pick" ? "Search the same set as TUI /login." : step === "method" ? "This provider accepts an account or an API key." : step === "oauth" ? "Account login is not in pi RPC yet." : "Saved in ~/.pi/agent/auth.json. The key is never shown again."}
            </Dialog.Description>

            {step === "pick" ? (
              <Command loop className="prov-pick">
                <Command.Input className="combo-input" placeholder="Search providers" />
                <Command.List className="prov-pick-list">
                  <Command.Empty className="combo-empty">No matches</Command.Empty>
                  {available.map((p) => (
                    <Command.Item key={p.id} value={p.id + " " + p.login} className="cockpit-opt" onSelect={() => chooseProvider(p)}>
                      <span>{p.id}</span>
                      <span className="combo-hint">{p.login === "both" ? "account or api key" : p.login === "oauth" ? "account" : "api key"}</span>
                    </Command.Item>
                  ))}
                </Command.List>
              </Command>
            ) : null}

            {step === "method" ? (
              <div className="prov-methods">
                <button type="button" className="btn btn-primary" onClick={() => setStep("key")}>Sign in with an API key</button>
                <button type="button" className="btn btn-ghost" onClick={() => setStep("oauth")}>Sign in with an account</button>
                <div className="dlg-actions">
                  <button type="button" className="btn btn-ghost btn-sm" onClick={() => { setPick(null); setStep("pick"); }}>Back</button>
                </div>
              </div>
            ) : null}

            {step === "oauth" ? (
              <div>
                <p className="settings-desc">Pi has no RPC login. Use /login {pick ? pick.id : ""} in the terminal for subscriptions (Codex, Claude Pro, Copilot, xAI, OpenRouter, Radius).</p>
                <div className="dlg-actions">
                  <button type="button" className="btn btn-ghost btn-sm" onClick={() => { if (pick && pick.login === "both") setStep("method"); else { setPick(null); setStep("pick"); } }}>Back</button>
                  <button type="button" className="btn btn-primary btn-sm" onClick={closeAdd}>Close</button>
                </div>
              </div>
            ) : null}

            {step === "key" ? (
              <form className="form-new" noValidate onSubmit={save}>
                <input type="password" autoComplete="off" placeholder="sk-…" value={key} onChange={(e) => setKey(e.target.value)} />
                <p className="form-error" hidden={!err}>{err}</p>
                <div className="dlg-actions">
                  <button type="button" className="btn btn-ghost btn-sm" onClick={() => { if (pick && pick.login === "both") setStep("method"); else { setPick(null); setStep("pick"); } }}>Back</button>
                  <button type="submit" className="btn btn-primary btn-sm" disabled={busy}>Save</button>
                </div>
              </form>
            ) : null}
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>
    </PageFrame>
  );
}
