import { useEffect, useRef, useState } from "react";
import * as Dialog from "./MobileSheet.jsx";
import { api } from "@picode/shared/client/api.js";
import { toast } from "../lib/toast.js";
import { apiKeySchema, claudePlatformSchema, parseForm } from "@picode/shared/contracts/schemas.js";

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

// The third-party platforms Claude Code's /login offers, with the sign-in
// methods its own wizard lists (2.1.280) and the default the form starts on.
const PLATFORMS = {
  bedrock: {
    name: "Amazon Bedrock",
    auth: [
      ["bearer", "Bedrock API key"],
      ["profile", "AWS profile (SSO or named profile)"],
      ["accessKey", "Access key + secret"],
      ["environment", "Credentials already in my environment"],
    ],
    region: "us-east-1",
  },
  foundry: {
    name: "Microsoft Foundry",
    auth: [
      ["apiKey", "Foundry API key"],
      ["environment", "Microsoft Entra ID or my environment"],
    ],
  },
  vertex: {
    name: "Google Vertex AI",
    auth: [
      ["adc", "Application Default Credentials (gcloud auth)"],
      ["serviceAccount", "Service account key file"],
      ["environment", "Credentials already in my environment"],
    ],
    region: "global",
  },
};

function blankPlatform(kind, from) {
  const def = PLATFORMS[kind];
  const same = from && from.kind === kind ? from : null;
  return {
    kind,
    auth: (same && same.auth) || def.auth[0][0],
    region: (same && same.region) || def.region || "",
    profile: (same && same.profile) || "",
    project: (same && same.project) || "",
    resource: (same && same.resource) || "",
    keyFile: (same && same.keyFile) || "",
    bearerToken: "", accessKeyId: "", secretAccessKey: "", sessionToken: "", apiKey: "",
  };
}

export default function ClaudeCodeLoginDialog({ open, onClose, onSaved, onTerminalSignin, editPlatform }) {
  const attempt = useRef(0);
  useEffect(() => () => { attempt.current++; }, []);
  const [step, setStep] = useState("method"); // method | subscription | console | platforms | platform
  const [pf, setPf] = useState(null);
  const [key, setKey] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [waiting, setWaiting] = useState(false);

  function reset() {
    setStep(editPlatform ? "platform" : "method");
    setPf(editPlatform ? blankPlatform(editPlatform.kind, editPlatform) : null);
    setKey("");
    setErr("");
    setBusy(false);
    setWaiting(false);
  }

  useEffect(() => {
    attempt.current++;
    reset();
  }, [open, editPlatform ? editPlatform.kind : ""]);

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
    if (step === "platform") { setStep("platforms"); return; }
    setStep("method");
  }

  function pickPlatform(kind) {
    setErr("");
    setPf(blankPlatform(kind, editPlatform));
    setStep("platform");
  }

  // An error belongs to the form as it was: any edit clears it, so a "key
  // required" never lingers under a method that has no key field.
  const field = (name) => (e) => { setErr(""); setPf({ ...pf, [name]: e.target.value }); };

  async function savePlatform(e) {
    e.preventDefault();
    const parsed = parseForm(claudePlatformSchema, pf);
    if (!parsed.ok) { setErr(parsed.error); return; }
    setBusy(true);
    setErr("");
    try {
      await api("/api/claude-code/platform", { ...JSON_POST(parsed.value), method: "PUT" });
      await done("Claude Code will use " + PLATFORMS[pf.kind].name + " in new terminals.");
    } catch (ex) {
      setErr(ex.message);
      setBusy(false);
    }
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

  const title = step === "subscription" ? "Claude account"
    : step === "console" ? "Anthropic Console key"
      : step === "platforms" ? "3rd-party platform"
        : step === "platform" && pf ? PLATFORMS[pf.kind].name
          : "Sign in to Claude Code";
  const description = step === "subscription"
    ? (waiting ? "Finish sign-in in the browser tab." : "Pro, Max, Team or Enterprise. The sign-in opens in a browser tab.")
    : step === "console"
      ? "Billed by API usage. Claude Code uses this key in new terminals instead of your subscription."
      : step === "platforms"
        ? "Billed by your cloud account. Pick where Claude Code runs."
        : step === "platform"
          ? "Saved in Claude Code's own settings, the way its /login saves it. New terminals use it."
          : "Use your Claude subscription, an Anthropic Console key, or a cloud platform.";

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
              <button type="button" className="btn btn-ghost" onClick={() => setStep("platforms")}>3rd-party platform</button>
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

          {step === "platforms" ? (
            <div className="prov-methods">
              {Object.entries(PLATFORMS).map(([kind, def]) => (
                <button key={kind} type="button" className="btn btn-ghost" onClick={() => pickPlatform(kind)}>{def.name}</button>
              ))}
              <div className="dlg-actions" data-align-row data-align-wrap>
                <button type="button" className="btn btn-ghost btn-sm" onClick={back}>Back</button>
              </div>
            </div>
          ) : null}

          {step === "platform" && pf ? (
            <form className="cred-form" noValidate onSubmit={savePlatform}>
              <label className="cred-field">
                <span>Sign in with</span>
                <select className="cred-input" value={pf.auth} onChange={field("auth")}>
                  {PLATFORMS[pf.kind].auth.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
                </select>
              </label>
              {pf.kind === "foundry" ? (
                <label className="cred-field">
                  <span>Resource name</span>
                  <input className="cred-input" autoComplete="off" spellCheck={false} placeholder="my-foundry-resource" value={pf.resource} onChange={field("resource")} />
                </label>
              ) : null}
              {pf.kind === "vertex" ? (
                <label className="cred-field">
                  <span>Project ID</span>
                  <input className="cred-input" autoComplete="off" spellCheck={false} placeholder="my-gcp-project" value={pf.project} onChange={field("project")} />
                </label>
              ) : null}
              {pf.kind !== "foundry" ? (
                <label className="cred-field">
                  <span>Region</span>
                  <input className="cred-input" autoComplete="off" spellCheck={false} placeholder={PLATFORMS[pf.kind].region} value={pf.region} onChange={field("region")} />
                </label>
              ) : null}
              {pf.kind === "bedrock" && pf.auth === "bearer" ? (
                <label className="cred-field">
                  <span>Bedrock API key</span>
                  <input className="cred-input" type="password" autoComplete="off" spellCheck={false} value={pf.bearerToken} onChange={field("bearerToken")} />
                </label>
              ) : null}
              {pf.kind === "bedrock" && pf.auth === "profile" ? (
                <label className="cred-field">
                  <span>AWS profile</span>
                  <input className="cred-input" autoComplete="off" spellCheck={false} placeholder="default" value={pf.profile} onChange={field("profile")} />
                </label>
              ) : null}
              {pf.kind === "bedrock" && pf.auth === "accessKey" ? (
                <>
                  <label className="cred-field">
                    <span>Access key ID</span>
                    <input className="cred-input" autoComplete="off" spellCheck={false} placeholder="AKIA…" value={pf.accessKeyId} onChange={field("accessKeyId")} />
                  </label>
                  <label className="cred-field">
                    <span>Secret access key</span>
                    <input className="cred-input" type="password" autoComplete="off" spellCheck={false} value={pf.secretAccessKey} onChange={field("secretAccessKey")} />
                  </label>
                  <label className="cred-field">
                    <span>Session token (optional)</span>
                    <input className="cred-input" type="password" autoComplete="off" spellCheck={false} value={pf.sessionToken} onChange={field("sessionToken")} />
                  </label>
                </>
              ) : null}
              {pf.kind === "vertex" && pf.auth === "serviceAccount" ? (
                <label className="cred-field">
                  <span>Key file (full path)</span>
                  <input className="cred-input" autoComplete="off" spellCheck={false} placeholder="/home/me/key.json" value={pf.keyFile} onChange={field("keyFile")} />
                </label>
              ) : null}
              {pf.kind === "foundry" && pf.auth === "apiKey" ? (
                <label className="cred-field">
                  <span>Foundry API key</span>
                  <input className="cred-input" type="password" autoComplete="off" spellCheck={false} value={pf.apiKey} onChange={field("apiKey")} />
                </label>
              ) : null}
              {pf.auth === "adc" ? <em className="cred-help">Uses the Google Cloud login on this machine (gcloud auth application-default login).</em> : null}
              {pf.auth === "environment" ? <em className="cred-help">Claude Code uses the credentials its environment already has.</em> : null}
              {editPlatform && editPlatform.kind === pf.kind && ["bearer", "accessKey", "apiKey"].includes(pf.auth) ? (
                <em className="cred-help">The saved secret is not shown. Enter it again to keep this method.</em>
              ) : null}
              <p className="form-error" role="alert" hidden={!err}>{err}</p>
              <div className="dlg-actions" data-align-row data-align-wrap>
                <button type="button" className="btn btn-ghost btn-sm" onClick={editPlatform ? close : back}>{editPlatform ? "Cancel" : "Back"}</button>
                <button type="submit" className="btn btn-primary btn-sm" disabled={busy}>{busy ? "Saving…" : "Save and use"}</button>
              </div>
            </form>
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
