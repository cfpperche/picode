import { useEffect, useMemo, useRef, useState } from "react";
import * as Dialog from "./ResponsiveDialog.jsx";
import { Command, defaultFilter } from "cmdk";
import { api } from "@picode/shared/client/api.js";
import { toast } from "../lib/toast.js";
import { apiKeySchema, parseForm } from "@picode/shared/contracts/schemas.js";
import { ProviderFace } from "./ProviderFaces.jsx";

// OpenCode's own sign-in, in PiCode's GUI (ADR-0201). It follows the flow
// OpenCode's `auth login` walks: pick a provider from OpenCode's catalog, pick
// one of the methods OpenCode offers for it, answer its prompts, then paste a
// key or finish the sign-in on the provider's page (pasting a code back when
// the method asks for one). PiCode runs OpenCode's own server for all of it,
// so OpenCode writes its own credentials.
const JSON_BODY = (method, body) => ({
  method,
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify(body),
});

function pickValue(p) {
  return (p.id + " " + (p.name || "")).toLowerCase();
}

function rank(rows, q) {
  if (!q) return rows;
  return rows
    .map((p) => ({ p, score: defaultFilter(pickValue(p), q) }))
    .filter((r) => r.score > 0)
    .sort((a, b) => b.score - a.score)
    .map((r) => r.p);
}

// A prompt shows when its `when` rule holds for the answers so far.
function promptShown(prompt, inputs) {
  const w = prompt.when;
  if (!w) return true;
  const v = inputs[w.key] || "";
  return w.op === "neq" ? v !== w.value : v === w.value;
}

export default function OpenCodeLoginDialog({ open, onClose, onSaved, onTerminalSignin }) {
  const attempt = useRef(0);
  useEffect(() => () => { attempt.current++; }, []);
  const [catalog, setCatalog] = useState(null);
  const [loadErr, setLoadErr] = useState("");
  const [step, setStep] = useState("pick"); // pick | method | form | page
  const [pick, setPick] = useState(null);
  const [method, setMethod] = useState(null);
  const [inputs, setInputs] = useState({});
  const [key, setKey] = useState("");
  const [page, setPage] = useState(null); // { url, mode, instructions }
  const [code, setCode] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [waiting, setWaiting] = useState(false);
  const [query, setQuery] = useState("");
  const [selected, setSelected] = useState("");
  const listRef = useRef(null);

  function reset() {
    setStep("pick");
    setPick(null);
    setMethod(null);
    setInputs({});
    setKey("");
    setPage(null);
    setCode("");
    setErr("");
    setBusy(false);
    setWaiting(false);
    setQuery("");
  }

  // The catalog comes from OpenCode's own server, which PiCode may have to
  // start: it gets a bounded wait, and a failure says so with a retry.
  const loadSeq = useRef(0);
  function loadCatalog() {
    const mine = ++loadSeq.current;
    setLoadErr("");
    setCatalog(null);
    const slow = setTimeout(() => {
      if (mine === loadSeq.current) setLoadErr("OpenCode is taking too long to answer.");
    }, 40000);
    api("/api/opencode/catalog")
      .then((c) => { if (mine === loadSeq.current) { clearTimeout(slow); setLoadErr(""); setCatalog(c); } })
      .catch((ex) => { if (mine === loadSeq.current) { clearTimeout(slow); setLoadErr(ex.message); } });
  }

  useEffect(() => {
    attempt.current++;
    reset();
    if (!open) { loadSeq.current++; return; }
    loadCatalog();
    return () => { loadSeq.current++; };
  }, [open]);

  const rows = useMemo(() => {
    const list = catalog && Array.isArray(catalog.providers) ? [...catalog.providers] : [];
    return list.sort((a, b) => (a.name || a.id).localeCompare(b.name || b.id, undefined, { sensitivity: "base" }));
  }, [catalog]);
  const shown = useMemo(() => rank(rows, query.trim()), [rows, query]);

  function search(v) {
    const next = rank(rows, v.trim());
    setQuery(v);
    setSelected(next.length ? pickValue(next[0]) : "");
    requestAnimationFrame(() => { if (listRef.current) listRef.current.scrollTop = 0; });
  }
  const rowsKey = rows.map((p) => p.id).join("\n");
  useEffect(() => {
    if (step !== "pick") return;
    setSelected(shown.length ? pickValue(shown[0]) : "");
    requestAnimationFrame(() => { if (listRef.current) listRef.current.scrollTop = 0; });
  }, [step, rowsKey]);

  function cancelPending() {
    if (waiting) api("/api/opencode/credential", { method: "DELETE" }).catch(() => {});
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
    setCode("");
    if (step === "page" || step === "form") {
      if (pick && pick.methods.length > 1) { setStep("method"); return; }
    }
    setPick(null);
    setMethod(null);
    setStep("pick");
  }

  function choose(p) {
    setPick(p);
    setErr("");
    if (p.methods.length === 1) { chooseMethod(p.methods[0]); return; }
    setStep("method");
  }

  function chooseMethod(m) {
    setMethod(m);
    setInputs({});
    setKey("");
    setErr("");
    setStep("form");
  }

  const prompts = (method && method.prompts) || [];
  const visiblePrompts = prompts.filter((pr) => promptShown(pr, inputs));

  async function done(message) {
    reset();
    if (onClose) onClose();
    toast.ok(message);
    if (onSaved) await onSaved();
  }

  function answers() {
    const out = {};
    for (const pr of visiblePrompts) out[pr.key] = (inputs[pr.key] || "").trim();
    return out;
  }

  function missingAnswer() {
    for (const pr of visiblePrompts) {
      if (!(inputs[pr.key] || "").trim()) return pr.message || "Answer every question.";
    }
    return "";
  }

  async function submit(e) {
    e.preventDefault();
    const missing = missingAnswer();
    if (missing) { setErr(missing); return; }
    if (method.type === "api") {
      const parsed = parseForm(apiKeySchema, { key });
      if (!parsed.ok) { setErr(parsed.error); return; }
      setBusy(true);
      setErr("");
      try {
        await api("/api/opencode/credential", JSON_BODY("POST", { provider: pick.id, method: method.index, type: "api", key: parsed.value.key, inputs: answers() }));
        await done("Key saved for " + (pick.name || pick.id) + ".");
      } catch (ex) {
        setErr(ex.message);
        setBusy(false);
      }
      return;
    }
    const mine = ++attempt.current;
    const current = () => mine === attempt.current;
    setBusy(true);
    setErr("");
    try {
      const res = await api("/api/opencode/credential", JSON_BODY("POST", { provider: pick.id, method: method.index, type: "oauth", inputs: answers() }));
      if (!current()) return;
      setPage(res);
      setStep("page");
      setBusy(false);
      if (res.url) window.open(res.url, "_blank", "noopener");
      if (res.mode !== "auto") return;
      setWaiting(true);
      const t0 = Date.now();
      while (Date.now() - t0 < 10 * 60 * 1000) {
        await new Promise((r) => setTimeout(r, 1000));
        if (!current()) return;
        const st = await api("/api/opencode/credential");
        if (!current()) return;
        if (st && !st.pending && st.done) {
          setWaiting(false);
          if (st.error) { setErr(st.error); return; }
          await done("Signed in to " + (pick.name || pick.id) + ".");
          return;
        }
      }
      setWaiting(false);
      setErr("The sign-in expired. Start it again.");
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
    setBusy(true);
    setErr("");
    try {
      await api("/api/opencode/credential/code", JSON_BODY("POST", { code: value }));
      await done("Signed in to " + (pick.name || pick.id) + ".");
    } catch (ex) {
      setErr(ex.message);
      setBusy(false);
    }
  }

  async function terminal() {
    setBusy(true);
    try {
      const res = await api("/api/credentials/signin", JSON_BODY("POST", { cli: "opencode" }));
      close();
      if (onTerminalSignin) onTerminalSignin(res);
    } catch (ex) {
      setErr(ex.message);
      setBusy(false);
    }
  }

  const name = pick ? pick.name || pick.id : "";
  const title = step === "pick" ? "Add provider" : step === "method" ? name : step === "form" ? (method && method.type === "api" ? "API key · " + name : name) : name;
  const description = step === "pick"
    ? (catalog ? "Pick a provider. OpenCode keeps the credential in its own store." : loadErr ? "OpenCode's provider list could not be loaded." : "Loading OpenCode's providers…")
    : step === "method"
      ? "Choose how to sign in."
      : step === "form"
        ? (method && method.type === "api" ? "Paste the key. OpenCode keeps it in its own store." : "The sign-in opens on the provider's page.")
        : page && page.mode === "code"
          ? "Finish on the provider's page, then paste the code it shows."
          : (page && page.instructions) || "Finish sign-in in the browser tab.";

  return (
    <Dialog.Root open={open} onOpenChange={(o) => { if (!o) close(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg dlg-create" onCloseAutoFocus={(e) => e.preventDefault()}>
          <Dialog.Title className="dlg-title">{title}</Dialog.Title>
          <Dialog.Description className="dlg-body">{description}</Dialog.Description>

          {step === "pick" ? (
            <>
              {catalog ? (
                <Command loop className="prov-pick" shouldFilter={false} value={selected} onValueChange={setSelected}>
                  <Command.Input className="combo-input" placeholder="Search providers" value={query} onValueChange={search} />
                  <Command.List className="prov-pick-list" ref={listRef}>
                    <Command.Empty className="combo-empty">No matches</Command.Empty>
                    {shown.map((p) => (
                      <Command.Item key={p.id} value={pickValue(p)} className="cockpit-opt" onSelect={() => choose(p)}>
                        <ProviderFace id={p.id} name={p.name || p.id} />
                        <span>{p.name || p.id}</span>
                        <span className="combo-hint">{p.connected ? "connected" : p.methods.some((m) => m.type === "oauth") ? (p.methods.some((m) => m.type === "api") ? "account or api key" : "account") : "api key"}</span>
                      </Command.Item>
                    ))}
                  </Command.List>
                </Command>
              ) : loadErr ? (
                <div className="cli-notice is-error" role="alert">
                  <span>{loadErr}</span>
                  <button type="button" className="btn btn-ghost btn-sm" onClick={loadCatalog}>Try again</button>
                </div>
              ) : (
                <div className="cred-loading" role="status" aria-label="Loading OpenCode's providers">
                  <div className="cred-skel-row" />
                  <div className="cred-skel-row" />
                </div>
              )}
              <p className="form-error" role="alert" hidden={!err}>{err}</p>
              <div className="dlg-actions" data-align-row data-align-wrap>
                <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={terminal}>Sign in from a terminal</button>
                <button type="button" className="btn btn-ghost btn-sm" onClick={close}>Cancel</button>
              </div>
            </>
          ) : null}

          {step === "method" && pick ? (
            <div className="prov-methods">
              {pick.methods.map((m, i) => (
                <button key={m.index + ":" + i} type="button" className={i === 0 ? "btn btn-primary" : "btn btn-ghost"} onClick={() => chooseMethod(m)}>{m.label}</button>
              ))}
              <div className="dlg-actions" data-align-row data-align-wrap>
                <button type="button" className="btn btn-ghost btn-sm" onClick={back}>Back</button>
              </div>
            </div>
          ) : null}

          {step === "form" && method ? (
            <form className="cred-form" noValidate onSubmit={submit}>
              {visiblePrompts.map((pr) => (
                <label key={pr.key} className="cred-field">
                  <span>{pr.message}</span>
                  {pr.type === "select" ? (
                    <select className="cred-input" value={inputs[pr.key] || ""} onChange={(e) => { setErr(""); setInputs({ ...inputs, [pr.key]: e.target.value }); }}>
                      <option value="" disabled>Choose…</option>
                      {(pr.options || []).map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
                    </select>
                  ) : (
                    <input className="cred-input" autoComplete="off" spellCheck={false} placeholder={pr.placeholder || ""} value={inputs[pr.key] || ""} onChange={(e) => { setErr(""); setInputs({ ...inputs, [pr.key]: e.target.value }); }} />
                  )}
                </label>
              ))}
              {method.type === "api" ? (
                <label className="cred-field">
                  <span>API key</span>
                  <input className="cred-input" type="password" autoComplete="off" spellCheck={false} aria-invalid={err ? true : undefined} value={key} onChange={(e) => { setErr(""); setKey(e.target.value); }} />
                </label>
              ) : null}
              <p className="form-error" role="alert" hidden={!err}>{err}</p>
              <div className="dlg-actions" data-align-row data-align-wrap>
                <button type="button" className="btn btn-ghost btn-sm" onClick={back}>Back</button>
                <button type="submit" className="btn btn-primary btn-sm" disabled={busy}>{busy ? "Saving…" : method.type === "api" ? "Save" : "Continue"}</button>
              </div>
            </form>
          ) : null}

          {step === "page" && page ? (
            page.mode === "code" ? (
              <form className="cred-form" noValidate onSubmit={sendCode}>
                <a className="cred-help" href={page.url} target="_blank" rel="noopener noreferrer">Open the sign-in page again</a>
                <label className="cred-field">
                  <span>Code from the page</span>
                  <input className="cred-input" autoComplete="off" spellCheck={false} aria-invalid={err ? true : undefined} value={code} onChange={(e) => { setErr(""); setCode(e.target.value); }} />
                </label>
                <p className="form-error" role="alert" hidden={!err}>{err}</p>
                <div className="dlg-actions" data-align-row data-align-wrap>
                  <button type="button" className="btn btn-ghost btn-sm" onClick={back}>Back</button>
                  <button type="submit" className="btn btn-primary btn-sm" disabled={busy}>{busy ? <span className="cred-waiting">Signing in</span> : "Sign in"}</button>
                </div>
              </form>
            ) : (
              <div>
                <a className="cred-help cred-link-block" href={page.url} target="_blank" rel="noopener noreferrer">Open the sign-in page again</a>
                <p className="form-error" role="alert" hidden={!err}>{err}</p>
                <div className="dlg-actions" data-align-row data-align-wrap>
                  <button type="button" className="btn btn-ghost btn-sm" onClick={back}>Back</button>
                  <button type="button" className="btn btn-primary btn-sm" disabled>{waiting ? <span className="cred-waiting">Waiting</span> : "Waiting"}</button>
                </div>
              </div>
            )
          ) : null}
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
