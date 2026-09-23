import { useEffect, useRef, useState } from "react";
import * as Dialog from "./MobileSheet.jsx";
import { api } from "@picode/shared/client/api.js";
import { toast } from "../lib/toast.js";
import { apiKeySchema, parseForm } from "@picode/shared/contracts/schemas.js";

// Claude Code's own sign-in, in PiCode's GUI (ADR-0187). It mirrors Claude
// Code's /login — a Claude subscription or an Anthropic Console key — so a
// person never has to open a terminal to sign in; the terminal stays one
// click away for whoever prefers it. The subscription runs through PiCode's
// browser flow with Claude Code's own OAuth client and lands in Claude Code's
// file; the Console key becomes the login Claude Code uses in new terminals
// (a key outranks the subscription there, so choosing one is explicit).
const JSON_POST = (body) => ({
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify(body),
});

export default function ClaudeCodeLoginDialog({ open, onClose, onSaved, onTerminalSignin }) {
  const attempt = useRef(0);
  useEffect(() => () => { attempt.current++; }, []);
  const [step, setStep] = useState("method"); // method | subscription | console
  const [key, setKey] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [waiting, setWaiting] = useState(false);

  function reset() {
    setStep("method");
    setKey("");
    setErr("");
    setBusy(false);
    setWaiting(false);
  }

  useEffect(() => {
    attempt.current++;
    reset();
  }, [open]);

  function close() {
    attempt.current++;
    reset();
    if (onClose) onClose();
  }

  async function done(message) {
    reset();
    if (onClose) onClose();
    toast.ok(message);
    if (onSaved) await onSaved();
  }

  function back() {
    attempt.current++;
    setErr("");
    setBusy(false);
    setWaiting(false);
    setStep("method");
  }

  async function subscription() {
    const mine = ++attempt.current;
    const current = () => mine === attempt.current;
    setBusy(true);
    setErr("");
    try {
      const res = await api("/api/credentials/signin", JSON_POST({ cli: "claude-code", provider: "anthropic" }));
      if (!current()) return;
      if (!res || !res.oauth) {
        // The server fell back to Claude Code's own /login: the pane's
        // sign-in strip owns that terminal.
        close();
        if (onTerminalSignin) onTerminalSignin(res);
        return;
      }
      if (res.url) window.open(res.url, "_blank", "noopener");
      setWaiting(true);
      const t0 = Date.now();
      while (Date.now() - t0 < 5 * 60 * 1000) {
        await new Promise((r) => setTimeout(r, 1000));
        if (!current()) return;
        const st = await api("/api/oauth/status");
        if (!current()) return;
        if (st && st.done) {
          setWaiting(false);
          if (st.error) { setErr(st.error); setBusy(false); return; }
          await done("Signed in to your Claude account.");
          return;
        }
        if (st && !st.pending && !st.done) break;
      }
      setWaiting(false);
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
      const row = await api("/api/credentials", JSON_POST({ provider: "anthropic", label: "", key: parsed.value.key }));
      await api("/api/credentials/anthropic/" + encodeURIComponent(row.id) + "/activate", JSON_POST({ cli: "claude-code" }));
      await done("Claude Code will use this key in new terminals.");
    } catch (ex) {
      setErr(ex.message);
      setBusy(false);
    }
  }

  async function terminal() {
    setBusy(true);
    try {
      const res = await api("/api/credentials/signin", JSON_POST({ cli: "claude-code" }));
      close();
      if (onTerminalSignin) onTerminalSignin(res);
    } catch (ex) {
      setErr(ex.message);
      setBusy(false);
    }
  }

  const title = step === "subscription" ? "Claude account" : step === "console" ? "Anthropic Console key" : "Sign in to Claude Code";
  const description = step === "subscription"
    ? (waiting ? "Finish sign-in in the browser tab." : "Pro, Max, Team or Enterprise. The sign-in opens in a browser tab.")
    : step === "console"
      ? "Billed by API usage. Claude Code uses this key in new terminals instead of your subscription."
      : "Use your Claude subscription or an Anthropic Console key.";

  return (
    <Dialog.Root open={open} onOpenChange={(o) => { if (!o) close(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg dlg-create" onCloseAutoFocus={(e) => e.preventDefault()}>
          <Dialog.Title className="dlg-title">{title}</Dialog.Title>
          <Dialog.Description className="dlg-body">{description}</Dialog.Description>

          {step === "method" ? (
            <div className="prov-methods">
              <button type="button" className="btn btn-primary" onClick={() => setStep("subscription")}>Claude account with subscription</button>
              <button type="button" className="btn btn-ghost" onClick={() => setStep("console")}>Anthropic Console account</button>
              <p className="form-error" hidden={!err}>{err}</p>
              <div className="dlg-actions" data-align-row data-align-wrap>
                <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={terminal}>Sign in from a terminal</button>
                <button type="button" className="btn btn-ghost btn-sm" onClick={close}>Cancel</button>
              </div>
            </div>
          ) : null}

          {step === "subscription" ? (
            <div>
              <p className="form-error" hidden={!err}>{err}</p>
              <div className="dlg-actions" data-align-row data-align-wrap>
                <button type="button" className="btn btn-ghost btn-sm" onClick={back}>Back</button>
                <button type="button" className="btn btn-primary btn-sm" disabled={busy || waiting} onClick={subscription}>{waiting ? "Waiting…" : "Continue in browser"}</button>
              </div>
            </div>
          ) : null}

          {step === "console" ? (
            <form className="form-new" noValidate onSubmit={saveKey}>
              <label className="cred-field">
                <span>API key</span>
                <input className="cred-input" type="password" autoComplete="off" spellCheck={false} placeholder="sk-ant-api03-…" aria-invalid={err ? true : undefined} value={key} onChange={(e) => setKey(e.target.value)} />
              </label>
              <a className="cred-help" href="https://console.anthropic.com/settings/keys" target="_blank" rel="noopener noreferrer">Create a key in the Anthropic Console</a>
              <p className="form-error" hidden={!err}>{err}</p>
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
