import { useEffect, useRef, useState } from "react";
import * as Dialog from "./MobileSheet.jsx";
import { api } from "@picode/shared/client/api.js";
import { toast } from "../lib/toast.js";
import { apiKeySchema, codexBedrockSchema, parseForm } from "@picode/shared/contracts/schemas.js";

// Codex's own sign-in, in PiCode's GUI (ADR-0191). It offers what Codex's
// /login offers — ChatGPT in the browser, a device code, an API key, Amazon
// Bedrock — and runs each one through Codex's own app-server, so Codex writes
// its own files. The terminal stays one click away for whoever prefers it.
const JSON_BODY = (method, body) => ({
  method,
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify(body),
});

const BEDROCK_AUTH = [
  ["amazonBedrock", "Bedrock API key"],
  ["amazonBedrockAccessKeys", "Access key + secret"],
];

export default function CodexLoginDialog({ open, onClose, onSaved, onTerminalSignin, editPlatform }) {
  const attempt = useRef(0);
  useEffect(() => () => { attempt.current++; }, []);
  const [step, setStep] = useState("method"); // method | chatgpt | device | key | bedrock
  const [key, setKey] = useState("");
  const [bedrock, setBedrock] = useState(null);
  const [device, setDevice] = useState(null);
  const [authUrl, setAuthUrl] = useState("");
  const [copied, setCopied] = useState(false);
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [waiting, setWaiting] = useState(false);

  function blankBedrock(from) {
    return {
      type: from && from.auth === "accessKey" ? "amazonBedrockAccessKeys" : "amazonBedrock",
      region: (from && from.region) || "us-east-1",
      apiKey: "", accessKeyId: "", secretAccessKey: "", sessionToken: "",
    };
  }

  function reset() {
    setStep(editPlatform ? "bedrock" : "method");
    setBedrock(editPlatform ? blankBedrock(editPlatform) : null);
    setKey("");
    setDevice(null);
    setAuthUrl("");
    setErr("");
    setBusy(false);
    setWaiting(false);
  }

  useEffect(() => {
    attempt.current++;
    reset();
  }, [open, editPlatform ? editPlatform.kind : ""]);

  function cancelPending() {
    if (waiting) api("/api/codex/login", { method: "DELETE" }).catch(() => {});
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
    setDevice(null);
    setStep("method");
  }

  // A ChatGPT or key login lands in Codex's auth.json; filing it in the vault
  // keeps the Providers table honest. A refusal there is not a failed sign-in.
  async function fileInVault() {
    try {
      await api("/api/credentials/import", JSON_BODY("POST", { cli: "codex" }));
    } catch (ex) {
      toast.warn("Codex is signed in, but saving it to the vault failed: " + ex.message);
    }
  }

  async function done(message, file) {
    reset();
    if (onClose) onClose();
    if (file) await fileInVault();
    toast.ok(message);
    if (onSaved) await onSaved();
  }

  async function waitForCodex(current) {
    const t0 = Date.now();
    while (Date.now() - t0 < 10 * 60 * 1000) {
      await new Promise((r) => setTimeout(r, 1000));
      if (!current()) return null;
      const st = await api("/api/codex/login");
      if (!current()) return null;
      if (st && !st.pending && st.done) return st;
    }
    return { error: "The sign-in expired. Start it again." };
  }

  async function browserOrDevice(type) {
    const mine = ++attempt.current;
    const current = () => mine === attempt.current;
    setBusy(true);
    setErr("");
    try {
      const res = await api("/api/codex/login", JSON_BODY("POST", { type }));
      if (!current()) return;
      if (type === "chatgpt" && res.authUrl) {
        setAuthUrl(res.authUrl);
        window.open(res.authUrl, "_blank", "noopener");
      }
      if (type === "chatgptDeviceCode") setDevice({ url: res.verificationUrl, code: res.userCode });
      setWaiting(true);
      const st = await waitForCodex(current);
      if (!st) return;
      setWaiting(false);
      if (st.error) { setErr(st.error); setBusy(false); return; }
      await done("Codex is signed in with ChatGPT.", true);
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
      await api("/api/codex/login", JSON_BODY("POST", { type: "apiKey", apiKey: parsed.value.key }));
      await done("Codex will use this API key.", true);
    } catch (ex) {
      setErr(ex.message);
      setBusy(false);
    }
  }

  async function saveBedrock(e) {
    e.preventDefault();
    const parsed = parseForm(codexBedrockSchema, bedrock);
    if (!parsed.ok) { setErr(parsed.error); return; }
    setBusy(true);
    setErr("");
    try {
      await api("/api/codex/login", JSON_BODY("POST", parsed.value));
      await done("Codex will use Amazon Bedrock.", false);
    } catch (ex) {
      setErr(ex.message);
      setBusy(false);
    }
  }

  async function terminal() {
    setBusy(true);
    try {
      const res = await api("/api/credentials/signin", JSON_BODY("POST", { cli: "codex" }));
      close();
      if (onTerminalSignin) onTerminalSignin(res);
    } catch (ex) {
      setErr(ex.message);
      setBusy(false);
    }
  }

  const field = (name) => (e) => { setErr(""); setBedrock({ ...bedrock, [name]: e.target.value }); };

  const title = step === "chatgpt" ? "ChatGPT"
    : step === "device" ? "Device code"
      : step === "key" ? "OpenAI API key"
        : step === "bedrock" ? "Amazon Bedrock"
          : "Sign in to Codex";
  const description = step === "chatgpt"
    ? (waiting ? "Finish sign-in in the browser tab." : "Plus, Pro, Business, Edu or Enterprise. The sign-in opens in a browser tab on this computer.")
    : step === "device"
      ? (device ? "Open the link on any device and enter the code." : "Sign in from another device — a phone, another computer — with a one-time code.")
      : step === "key"
        ? "Pay for what you use. Codex keeps the key in its own auth.json."
        : step === "bedrock"
          ? "Connect using your AWS credentials. Codex keeps them in its own files."
          : "Use your ChatGPT plan, an OpenAI API key, or Amazon Bedrock.";

  return (
    <Dialog.Root open={open} onOpenChange={(o) => { if (!o) close(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg dlg-create" onCloseAutoFocus={(e) => e.preventDefault()}>
          <Dialog.Title className="dlg-title">{title}</Dialog.Title>
          <Dialog.Description className="dlg-body">{description}</Dialog.Description>

          {step === "method" ? (
            <div className="prov-methods">
              <button type="button" className="btn btn-primary" onClick={() => setStep("chatgpt")}>Sign in with ChatGPT</button>
              <button type="button" className="btn btn-ghost" onClick={() => setStep("device")}>Sign in with a device code</button>
              <button type="button" className="btn btn-ghost" onClick={() => setStep("key")}>Provide your own API key</button>
              <button type="button" className="btn btn-ghost" onClick={() => { setBedrock(blankBedrock(null)); setStep("bedrock"); }}>Use Amazon Bedrock</button>
              <p className="form-error" hidden={!err}>{err}</p>
              <div className="dlg-actions" data-align-row data-align-wrap>
                <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={terminal}>Sign in from a terminal</button>
                <button type="button" className="btn btn-ghost btn-sm" onClick={close}>Cancel</button>
              </div>
            </div>
          ) : null}

          {step === "chatgpt" ? (
            <div>
              {waiting && authUrl ? (
                <p className="cred-help">
                  <a className="cred-help" href={authUrl} target="_blank" rel="noopener noreferrer">Open the sign-in page again</a>
                  {" · Not on this computer? Go back and use a device code."}
                </p>
              ) : null}
              <p className="form-error" role="alert" hidden={!err}>{err}</p>
              <div className="dlg-actions" data-align-row data-align-wrap>
                <button type="button" className="btn btn-ghost btn-sm" onClick={back}>Back</button>
                <button type="button" className="btn btn-primary btn-sm" disabled={busy || waiting} onClick={() => browserOrDevice("chatgpt")}>{waiting ? <span className="cred-waiting">Waiting</span> : "Continue in browser"}</button>
              </div>
            </div>
          ) : null}

          {step === "device" ? (
            <div>
              {device ? (
                <>
                  <p className="oauth-code">{device.code}</p>
                  <button type="button" className="btn btn-ghost btn-sm cred-copy" onClick={async () => {
                    try { await navigator.clipboard.writeText(device.code); setCopied(true); setTimeout(() => setCopied(false), 1200); } catch { toast.info("Copy failed — select the code instead."); }
                  }}>{copied ? "Copied" : "Copy code"}</button>
                  <a className="cred-help cred-link-block" href={device.url} target="_blank" rel="noopener noreferrer">{device.url}</a>
                </>
              ) : null}
              <p className="form-error" role="alert" hidden={!err}>{err}</p>
              <div className="dlg-actions" data-align-row data-align-wrap>
                <button type="button" className="btn btn-ghost btn-sm" onClick={back}>Back</button>
                {device ? (
                  <button type="button" className="btn btn-primary btn-sm" disabled><span className="cred-waiting">Waiting</span></button>
                ) : (
                  <button type="button" className="btn btn-primary btn-sm" disabled={busy} onClick={() => browserOrDevice("chatgptDeviceCode")}>Get a code</button>
                )}
              </div>
            </div>
          ) : null}

          {step === "key" ? (
            <form className="cred-form" noValidate onSubmit={saveKey}>
              <label className="cred-field">
                <span>API key</span>
                <input className="cred-input" type="password" autoComplete="off" spellCheck={false} placeholder="sk-…" aria-invalid={err ? true : undefined} value={key} onChange={(e) => { setErr(""); setKey(e.target.value); }} />
              </label>
              <a className="cred-help" href="https://platform.openai.com/api-keys" target="_blank" rel="noopener noreferrer">Create a key on the OpenAI platform</a>
              <p className="form-error" role="alert" hidden={!err}>{err}</p>
              <div className="dlg-actions" data-align-row data-align-wrap>
                <button type="button" className="btn btn-ghost btn-sm" onClick={back}>Back</button>
                <button type="submit" className="btn btn-primary btn-sm" disabled={busy}>{busy ? "Saving…" : "Save and use"}</button>
              </div>
            </form>
          ) : null}

          {step === "bedrock" && bedrock ? (
            <form className="cred-form" noValidate onSubmit={saveBedrock}>
              <label className="cred-field">
                <span>Sign in with</span>
                <select className="cred-input" value={bedrock.type} onChange={field("type")}>
                  {BEDROCK_AUTH.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
                </select>
              </label>
              <label className="cred-field">
                <span>Region</span>
                <input className="cred-input" autoComplete="off" spellCheck={false} placeholder="us-east-1" value={bedrock.region} onChange={field("region")} />
              </label>
              {bedrock.type === "amazonBedrock" ? (
                <label className="cred-field">
                  <span>Bedrock API key</span>
                  <input className="cred-input" type="password" autoComplete="off" spellCheck={false} value={bedrock.apiKey} onChange={field("apiKey")} />
                </label>
              ) : (
                <>
                  <label className="cred-field">
                    <span>Access key ID</span>
                    <input className="cred-input" autoComplete="off" spellCheck={false} placeholder="AKIA…" value={bedrock.accessKeyId} onChange={field("accessKeyId")} />
                  </label>
                  <label className="cred-field">
                    <span>Secret access key</span>
                    <input className="cred-input" type="password" autoComplete="off" spellCheck={false} value={bedrock.secretAccessKey} onChange={field("secretAccessKey")} />
                  </label>
                  <label className="cred-field">
                    <span>Session token (optional)</span>
                    <input className="cred-input" type="password" autoComplete="off" spellCheck={false} value={bedrock.sessionToken} onChange={field("sessionToken")} />
                  </label>
                </>
              )}
              {editPlatform ? <em className="cred-help">The saved secret is not shown. Enter it again to keep Bedrock.</em> : null}
              <p className="form-error" role="alert" hidden={!err}>{err}</p>
              <div className="dlg-actions" data-align-row data-align-wrap>
                <button type="button" className="btn btn-ghost btn-sm" onClick={editPlatform ? close : back}>{editPlatform ? "Cancel" : "Back"}</button>
                <button type="submit" className="btn btn-primary btn-sm" disabled={busy}>{busy ? "Saving…" : "Save and use"}</button>
              </div>
            </form>
          ) : null}
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
