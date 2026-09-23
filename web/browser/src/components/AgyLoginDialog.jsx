import { useEffect, useRef, useState } from "react";
import * as Dialog from "./ResponsiveDialog.jsx";
import { api } from "@picode/shared/client/api.js";
import { toast } from "../lib/toast.js";

// Antigravity's Google sign-in, in PiCode's GUI (ADR-0197). Antigravity has no
// login command: PiCode starts its own sign-in, shows Google's page, and the
// person pastes back the code that page ends on. Antigravity gives that a
// minute, so the dialog counts it down and offers a new link when it runs out.
const JSON_BODY = (method, body) => ({
  method,
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify(body),
});

// No terminal door here: Antigravity has no sign-in command to open (its
// login happens inside its own TUI when it starts).
export default function AgyLoginDialog({ open, onClose, onSaved }) {
  const attempt = useRef(0);
  useEffect(() => () => { attempt.current++; }, []);
  // Antigravity has one door, so the dialog opens on it.
  const [step, setStep] = useState("google");
  const [link, setLink] = useState("");
  const [code, setCode] = useState("");
  const [seconds, setSeconds] = useState(0);
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [waiting, setWaiting] = useState(false);
  const [sent, setSent] = useState(false);

  function reset() {
    setStep("google");
    setLink("");
    setCode("");
    setSeconds(0);
    setErr("");
    setBusy(false);
    setWaiting(false);
    setSent(false);
  }

  useEffect(() => {
    attempt.current++;
    reset();
  }, [open]);

  function cancelPending() {
    if (waiting) api("/api/agy/login", { method: "DELETE" }).catch(() => {});
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
    setLink("");
    setSent(false);
    close();
  }

  async function getLink() {
    const mine = ++attempt.current;
    const current = () => mine === attempt.current;
    setBusy(true);
    setErr("");
    setCode("");
    setSent(false);
    try {
      const res = await api("/api/agy/login", JSON_BODY("POST", {}));
      if (!current()) return;
      setLink(res.url);
      setSeconds(res.seconds || 60);
      window.open(res.url, "_blank", "noopener");
      setWaiting(true);
      setBusy(false);
      while (current()) {
        await new Promise((r) => setTimeout(r, 1000));
        if (!current()) return;
        const st = await api("/api/agy/login");
        if (!current()) return;
        if (st && st.pending) { setSeconds(st.seconds || 0); continue; }
        setWaiting(false);
        if (st && st.error) { setErr(st.error); setSent(false); return; }
        reset();
        if (onClose) onClose();
        toast.ok("Antigravity is signed in with your Google account.");
        if (onSaved) await onSaved();
        return;
      }
    } catch (ex) {
      if (!current()) return;
      setErr(ex.message);
      setWaiting(false);
      setBusy(false);
    }
  }

  async function sendCode(e) {
    e.preventDefault();
    const value = code.trim();
    if (!value) { setErr("Paste the code the page shows."); return; }
    setErr("");
    try {
      await api("/api/agy/login/code", JSON_BODY("POST", { code: value }));
      setSent(true);
    } catch (ex) {
      setErr(ex.message);
    }
  }

  const title = step === "google" ? "Google account" : "Sign in to Antigravity";
  const description = step === "google"
    ? (waiting
      ? (sent ? "Checking the code…" : "Sign in on the Google page, then paste the code it shows here within " + seconds + "s.")
      : "Antigravity signs in with your Google account. You paste back the code Google's page shows, within a minute.")
    : "Antigravity signs in with your Google account.";

  return (
    <Dialog.Root open={open} onOpenChange={(o) => { if (!o) close(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg dlg-create" onCloseAutoFocus={(e) => e.preventDefault()}>
          <Dialog.Title className="dlg-title">{title}</Dialog.Title>
          <Dialog.Description className="dlg-body">{description}</Dialog.Description>

          {step === "method" ? (
            <div className="prov-methods">
              <button type="button" className="btn btn-primary" onClick={() => setStep("google")}>Sign in with Google</button>
              <p className="form-error" hidden={!err}>{err}</p>
              <div className="dlg-actions" data-align-row data-align-wrap>
                <button type="button" className="btn btn-ghost btn-sm" onClick={close}>Cancel</button>
              </div>
            </div>
          ) : null}

          {step === "google" ? (
            waiting ? (
              <form className="cred-form" noValidate onSubmit={sendCode}>
                <a className="cred-help" href={link} target="_blank" rel="noopener noreferrer">Open the Google page again</a>
                <label className="cred-field">
                  <span>Code from the page</span>
                  <input className="cred-input" autoComplete="off" spellCheck={false} aria-invalid={err ? true : undefined} value={code} disabled={sent} onChange={(e) => { setErr(""); setCode(e.target.value); }} />
                </label>
                <p className="form-error" role="alert" hidden={!err}>{err}</p>
                <div className="dlg-actions" data-align-row data-align-wrap>
                  <button type="button" className="btn btn-ghost btn-sm" onClick={back}>Back</button>
                  <button type="submit" className="btn btn-primary btn-sm" disabled={sent}>{sent ? <span className="cred-waiting">Signing in</span> : "Sign in"}</button>
                </div>
              </form>
            ) : (
              <div>
                <p className="form-error" role="alert" hidden={!err}>{err}</p>
                <div className="dlg-actions" data-align-row data-align-wrap>
                  <button type="button" className="btn btn-ghost btn-sm" onClick={back}>Back</button>
                  <button type="button" className="btn btn-primary btn-sm" disabled={busy} onClick={getLink}>{err ? "Get a new link" : "Open Google"}</button>
                </div>
              </div>
            )
          ) : null}
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
