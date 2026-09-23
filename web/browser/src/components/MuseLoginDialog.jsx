import { useEffect, useRef, useState } from "react";
import * as Dialog from "./ResponsiveDialog.jsx";
import { api } from "@picode/shared/client/api.js";
import { toast } from "../lib/toast.js";
import { apiKeySchema, parseForm } from "@picode/shared/contracts/schemas.js";

// Muse's own sign-in, in PiCode's GUI (ADR-0195). It offers what Muse offers:
// the Meta account (`muse login`, a one-time code approved in the browser)
// and a Meta API key. PiCode runs Muse's own login and shows its page and
// code; Muse writes its own file. Muse keeps one credential, so a key is
// applied with Use, which keeps the account login in the vault first.
const JSON_BODY = (method, body) => ({
  method,
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify(body),
});

export default function MuseLoginDialog({ open, onClose, onSaved, onTerminalSignin }) {
  const attempt = useRef(0);
  useEffect(() => () => { attempt.current++; }, []);
  const [step, setStep] = useState("method"); // method | account | key
  const [key, setKey] = useState("");
  const [page, setPage] = useState(null);
  const [copied, setCopied] = useState(false);
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [waiting, setWaiting] = useState(false);

  function reset() {
    setStep("method");
    setKey("");
    setPage(null);
    setErr("");
    setBusy(false);
    setWaiting(false);
  }

  useEffect(() => {
    attempt.current++;
    reset();
  }, [open]);

  function cancelPending() {
    if (waiting) api("/api/muse/login", { method: "DELETE" }).catch(() => {});
  }

  function close() {
    attempt.current++;
    cancelPending();
    reset();
    if (onClose) onClose();
  }

  function back() {
    attempt.current++;
    cancelPending();
    setErr("");
    setBusy(false);
    setWaiting(false);
    setPage(null);
    setStep("method");
  }

  async function done(message) {
    reset();
    if (onClose) onClose();
    toast.ok(message);
    if (onSaved) await onSaved();
  }

  async function signIn() {
    const mine = ++attempt.current;
    const current = () => mine === attempt.current;
    setBusy(true);
    setErr("");
    try {
      const res = await api("/api/muse/login", JSON_BODY("POST", {}));
      if (!current()) return;
      setPage({ url: res.url, code: res.userCode || "" });
      setWaiting(true);
      const t0 = Date.now();
      while (Date.now() - t0 < 10 * 60 * 1000) {
        await new Promise((r) => setTimeout(r, 1000));
        if (!current()) return;
        const st = await api("/api/muse/login");
        if (!current()) return;
        if (st && !st.pending && st.done) {
          setWaiting(false);
          if (st.error) { setErr(st.error); setBusy(false); return; }
          await done("Muse is signed in with your Meta account.");
          return;
        }
      }
      setWaiting(false);
      setErr("The sign-in expired. Start it again.");
      setBusy(false);
    } catch (ex) {
      if (!current()) return;
      setErr(ex.message);
      setWaiting(false);
      setBusy(false);
    }
  }

  async function saveKey(e) {
    e.preventDefault();
    const parsed = parseForm(apiKeySchema, { key });
    if (!parsed.ok) { setErr(parsed.error); return; }
    setBusy(true);
    setErr("");
    try {
      const row = await api("/api/credentials", JSON_BODY("POST", { provider: "meta-ai", label: "", key: parsed.value.key }));
      await api("/api/credentials/meta-ai/" + encodeURIComponent(row.id) + "/activate", JSON_BODY("POST", { cli: "muse" }));
      await done("Muse uses this key now. Your account login is kept in the vault — Use brings it back.");
    } catch (ex) {
      setErr(ex.message);
      setBusy(false);
    }
  }

  async function terminal() {
    setBusy(true);
    try {
      const res = await api("/api/credentials/signin", JSON_BODY("POST", { cli: "muse" }));
      close();
      if (onTerminalSignin) onTerminalSignin(res);
    } catch (ex) {
      setErr(ex.message);
      setBusy(false);
    }
  }

  async function copyCode() {
    try {
      await navigator.clipboard.writeText(page.code);
      setCopied(true);
      setTimeout(() => setCopied(false), 1200);
    } catch {
      toast.info("Copy failed — select the code instead.");
    }
  }

  const title = step === "account" ? "Meta account" : step === "key" ? "Meta API key" : "Sign in to Muse";
  const description = step === "account"
    ? (page ? "Open the page and confirm the code matches." : "Sign in with your Meta account: you approve a one-time code in the browser, on any device.")
    : step === "key"
      ? "Muse keeps one credential: this key takes the place of your account login, which stays in the vault."
      : "Use your Meta account or a Meta API key.";

  return (
    <Dialog.Root open={open} onOpenChange={(o) => { if (!o) close(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg dlg-create" onCloseAutoFocus={(e) => e.preventDefault()}>
          <Dialog.Title className="dlg-title">{title}</Dialog.Title>
          <Dialog.Description className="dlg-body">{description}</Dialog.Description>

          {step === "method" ? (
            <div className="prov-methods">
              <button type="button" className="btn btn-primary" onClick={() => setStep("account")}>Sign in with your Meta account</button>
              <button type="button" className="btn btn-ghost" onClick={() => setStep("key")}>Use a Meta API key</button>
              <p className="form-error" hidden={!err}>{err}</p>
              <div className="dlg-actions" data-align-row data-align-wrap>
                <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={terminal}>Sign in from a terminal</button>
                <button type="button" className="btn btn-ghost btn-sm" onClick={close}>Cancel</button>
              </div>
            </div>
          ) : null}

          {step === "account" ? (
            <div>
              {page ? (
                <>
                  {page.code ? <p className="oauth-code">{page.code}</p> : null}
                  {page.code ? <button type="button" className="btn btn-ghost btn-sm cred-copy" onClick={copyCode}>{copied ? "Copied" : "Copy code"}</button> : null}
                  <a className="cred-help cred-link-block" href={page.url} target="_blank" rel="noopener noreferrer">{page.url}</a>
                </>
              ) : null}
              <p className="form-error" role="alert" hidden={!err}>{err}</p>
              <div className="dlg-actions" data-align-row data-align-wrap>
                <button type="button" className="btn btn-ghost btn-sm" onClick={back}>Back</button>
                {page ? (
                  <button type="button" className="btn btn-primary btn-sm" disabled><span className="cred-waiting">Waiting</span></button>
                ) : (
                  <button type="button" className="btn btn-primary btn-sm" disabled={busy} onClick={signIn}>Get a code</button>
                )}
              </div>
            </div>
          ) : null}

          {step === "key" ? (
            <form className="cred-form" noValidate onSubmit={saveKey}>
              <label className="cred-field">
                <span>API key</span>
                <input className="cred-input" type="password" autoComplete="off" spellCheck={false} aria-invalid={err ? true : undefined} value={key} onChange={(e) => { setErr(""); setKey(e.target.value); }} />
              </label>
              <p className="form-error" role="alert" hidden={!err}>{err}</p>
              <div className="dlg-actions" data-align-row data-align-wrap>
                <button type="button" className="btn btn-ghost btn-sm" onClick={back}>Back</button>
                <button type="submit" className="btn btn-primary btn-sm" disabled={busy}>{busy ? "Saving…" : "Save and use"}</button>
              </div>
            </form>
          ) : null}
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
