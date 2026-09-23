import { useEffect, useRef, useState } from "react";
import * as Dialog from "./ResponsiveDialog.jsx";
import { api } from "@picode/shared/client/api.js";
import { toast } from "../lib/toast.js";
import { apiKeySchema, parseForm } from "@picode/shared/contracts/schemas.js";

// Grok's own sign-in, in PiCode's GUI (ADR-0192). It offers what `grok login`
// offers — the browser, a device code — and an xAI API key. PiCode runs Grok's
// own login and shows its page and code here; Grok writes its own file. A key
// becomes the login Grok uses only when chosen, since Grok's own session
// outranks it: saving one files that session in the vault and signs Grok out.
const JSON_BODY = (method, body) => ({
  method,
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify(body),
});

export default function GrokLoginDialog({ open, onClose, onSaved, onTerminalSignin }) {
  const attempt = useRef(0);
  useEffect(() => () => { attempt.current++; }, []);
  const [step, setStep] = useState("method"); // method | browser | device | key
  const [key, setKey] = useState("");
  const [page, setPage] = useState(null); // { url, code }
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
    if (waiting) api("/api/grok/login", { method: "DELETE" }).catch(() => {});
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

  async function done(message, file) {
    reset();
    if (onClose) onClose();
    if (file) {
      try {
        await api("/api/credentials/import", JSON_BODY("POST", { cli: "grok" }));
      } catch (ex) {
        toast.warn("Grok is signed in, but saving it to the vault failed: " + ex.message);
      }
    }
    toast.ok(message);
    if (onSaved) await onSaved();
  }

  async function signIn(type) {
    const mine = ++attempt.current;
    const current = () => mine === attempt.current;
    setBusy(true);
    setErr("");
    try {
      const res = await api("/api/grok/login", JSON_BODY("POST", { type }));
      if (!current()) return;
      setPage({ url: res.url, code: res.userCode || "" });
      if (type === "oauth" && res.url) window.open(res.url, "_blank", "noopener");
      setWaiting(true);
      const t0 = Date.now();
      while (Date.now() - t0 < 10 * 60 * 1000) {
        await new Promise((r) => setTimeout(r, 1000));
        if (!current()) return;
        const st = await api("/api/grok/login");
        if (!current()) return;
        if (st && !st.pending && st.done) {
          setWaiting(false);
          if (st.error) { setErr(st.error); setBusy(false); return; }
          await done("Grok is signed in.", true);
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
      const row = await api("/api/credentials", JSON_BODY("POST", { provider: "xai", label: "", key: parsed.value.key }));
      await api("/api/credentials/xai/" + encodeURIComponent(row.id) + "/activate", JSON_BODY("POST", { cli: "grok" }));
      await done("Grok will use this key in new terminals. Its own sign-in is kept in the vault — Use brings it back.", false);
    } catch (ex) {
      setErr(ex.message);
      setBusy(false);
    }
  }

  async function terminal() {
    setBusy(true);
    try {
      const res = await api("/api/credentials/signin", JSON_BODY("POST", { cli: "grok" }));
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

  const title = step === "browser" ? "Grok account" : step === "device" ? "Device code" : step === "key" ? "xAI API key" : "Sign in to Grok";
  const description = step === "browser"
    ? (waiting ? "Finish sign-in in the browser tab." : "SuperGrok or X Premium. The sign-in opens in a browser tab on this computer.")
    : step === "device"
      ? (page ? "Open the link on any device and confirm the code." : "Sign in from another device — a phone, another computer — with a one-time code.")
      : step === "key"
        ? "Billed by API usage. Grok uses this key in new terminals; its own sign-in is kept in the vault."
        : "Use your Grok account or an xAI API key.";

  return (
    <Dialog.Root open={open} onOpenChange={(o) => { if (!o) close(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg dlg-create" onCloseAutoFocus={(e) => e.preventDefault()}>
          <Dialog.Title className="dlg-title">{title}</Dialog.Title>
          <Dialog.Description className="dlg-body">{description}</Dialog.Description>

          {step === "method" ? (
            <div className="prov-methods">
              <button type="button" className="btn btn-primary" onClick={() => setStep("browser")}>Sign in with Grok</button>
              <button type="button" className="btn btn-ghost" onClick={() => setStep("device")}>Sign in with a device code</button>
              <button type="button" className="btn btn-ghost" onClick={() => setStep("key")}>Use an xAI API key</button>
              <p className="form-error" hidden={!err}>{err}</p>
              <div className="dlg-actions" data-align-row data-align-wrap>
                <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={terminal}>Sign in from a terminal</button>
                <button type="button" className="btn btn-ghost btn-sm" onClick={close}>Cancel</button>
              </div>
            </div>
          ) : null}

          {step === "browser" ? (
            <div>
              {waiting && page && page.url ? (
                <p className="cred-help">
                  <a className="cred-help" href={page.url} target="_blank" rel="noopener noreferrer">Open the sign-in page again</a>
                  {" · Not on this computer? Go back and use a device code."}
                </p>
              ) : null}
              <p className="form-error" role="alert" hidden={!err}>{err}</p>
              <div className="dlg-actions" data-align-row data-align-wrap>
                <button type="button" className="btn btn-ghost btn-sm" onClick={back}>Back</button>
                <button type="button" className="btn btn-primary btn-sm" disabled={busy || waiting} onClick={() => signIn("oauth")}>{waiting ? <span className="cred-waiting">Waiting</span> : "Continue in browser"}</button>
              </div>
            </div>
          ) : null}

          {step === "device" ? (
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
                  <button type="button" className="btn btn-primary btn-sm" disabled={busy} onClick={() => signIn("device")}>Get a code</button>
                )}
              </div>
            </div>
          ) : null}

          {step === "key" ? (
            <form className="cred-form" noValidate onSubmit={saveKey}>
              <label className="cred-field">
                <span>API key</span>
                <input className="cred-input" type="password" autoComplete="off" spellCheck={false} placeholder="xai-…" aria-invalid={err ? true : undefined} value={key} onChange={(e) => { setErr(""); setKey(e.target.value); }} />
              </label>
              <a className="cred-help" href="https://console.x.ai" target="_blank" rel="noopener noreferrer">Create a key in the xAI console</a>
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
