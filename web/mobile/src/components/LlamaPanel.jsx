import { useEffect, useRef, useState } from "react";
import * as Dialog from "./MobileSheet.jsx";
import { Command } from "cmdk";
import PageFrame from "./PageFrame.jsx";
import { llamaLoginSchema, parseForm } from "@picode/shared/contracts/schemas.js";
import "./llama.css";
import { api } from "@picode/shared/client/api.js";
import { toastError } from "../lib/toast.js";
import { askConfirm } from "../lib/confirm.js";
import { IconDocs } from "./Icons.jsx";
import { DOCS_BASE } from "../lib/commandDocs.js";

function bytes(n) {
  if (!n) return "";
  if (n < 1024) return n + " B";
  if (n < 1024 * 1024) return (n / 1024).toFixed(1) + " KiB";
  if (n < 1024 * 1024 * 1024) return (n / (1024 * 1024)).toFixed(1) + " MiB";
  return (n / (1024 * 1024 * 1024)).toFixed(2) + " GiB";
}

export default function LlamaPanel({ onRefresh }) {
  const [url, setUrl] = useState("http://127.0.0.1:8080");
  const [ok, setOk] = useState(false);
  const [key, setKey] = useState("");
  const [checking, setChecking] = useState(true);
  const [saving, setSaving] = useState(false);
  const [connection, setConnection] = useState({ code: "checking", message: "Checking connection…" });
  const [formError, setFormError] = useState("");
  const [searched, setSearched] = useState(false);
  const [searching, setSearching] = useState(false);
  const [section, setSection] = useState(() => location.hash.endsWith("/server") ? "server" : "models");
  const initialized = useRef(false);
  const operation = useRef(false);
  useEffect(() => {
    const update = () => setSection(location.hash.endsWith("/server") ? "server" : "models");
    window.addEventListener("hashchange", update);
    if (!/^#\/llama\/(models|server)$/.test(location.hash)) location.replace("#/llama/models");
    return () => window.removeEventListener("hashchange", update);
  }, []);
  const [models, setModels] = useState([]);
  const [busy, setBusy] = useState("");
  const [notice, setNotice] = useState("");
  const [dl, setDl] = useState(false);
  const [q, setQ] = useState("");
  const [hits, setHits] = useState([]);
  const [info, setInfo] = useState(null);

  async function refresh() {
    setChecking(true);
    try {
      const res = await api("/api/llama");
      if (!initialized.current) { setUrl(res.url || "http://127.0.0.1:8080"); initialized.current = true; }
      setOk(!!res.ok);
      setConnection(res.connection || { code: res.ok ? "ready" : "unreachable", message: res.ok ? "Connected" : "Cannot reach the server from PiCode." });
      setModels(res.models || []);
    } catch {
      setOk(false);
      setConnection({ code: "unreachable", message: "Cannot check the connection. Try again." });
    } finally { setChecking(false); }
  }

  useEffect(() => { refresh(); }, []);

  async function saveUrl(e) {
    e.preventDefault();
    const parsed = parseForm(llamaLoginSchema, { url, key });
    if (!parsed.ok) { setFormError(parsed.error); return; }
    setFormError(""); setSaving(true);
    try {
      await api("/api/providers/llama.cpp", {
        method: "PUT", headers: { "Content-Type": "application/json" },
        body: JSON.stringify(parsed.value),
      });
      setKey(""); initialized.current = false;
      await refresh();
      if (onRefresh) await onRefresh();
    } catch (ex) { setFormError(ex.message || "Could not save the connection."); }
    finally { setSaving(false); }
  }

  async function runOp(fn) {
    // The op endpoints block until done (load 5 min, unload 2 min,
    // download 10 min) and the busy row is the motion; one refresh on
    // completion is the authority. No 1 s poll while the op runs —
    // ADR-0048 follow-up.
    await fn();
    await refresh();
    if (onRefresh) await onRefresh();
  }

  async function load(id) {
    if (operation.current) return;
    operation.current = true;
    const loaded = models.filter((m) => m.id !== id && (m.status === "loaded" || m.status === "sleeping"));
    let unloadOthers = false;
    if (loaded.length) {
      const choice = await askConfirm({
        title: "Load " + id,
        message: "Other models are loaded. Choose whether to free their memory first.",
        confirmLabel: "Load model",
        choices: [{ id: "unload", label: "Unload other models first", checked: false }],
      });
      if (!choice) { operation.current = false; return; }
      unloadOthers = !!choice.unload;
    }
    setNotice(""); setBusy(id);
    try {
      await runOp(() => api("/api/llama/load", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id, unloadOthers }),
      }));
      setNotice("Loaded " + id + ".");
    } catch (ex) { toastError(ex); }
    finally { setBusy(""); operation.current = false; }
  }

  async function unload(id) {
    if (operation.current) return;
    operation.current = true;
    const yes = await askConfirm({
      title: "Unload " + id,
      message: "Does not delete the file.",
      confirmLabel: "Unload",
    });
    if (!yes) { operation.current = false; return; }
    setNotice(""); setBusy(id);
    try {
      await runOp(() => api("/api/llama/unload", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id }),
      }));
      setNotice("Unloaded " + id + ".");
    } catch (ex) { toastError(ex); }
    finally { setBusy(""); operation.current = false; }
  }

  async function search(e) {
    e.preventDefault();
    setInfo(null); setSearching(true); setSearched(false);
    try {
      const res = await api("/api/llama/hf?q=" + encodeURIComponent(q));
      setHits(res.hits || []); setSearched(true);
    } catch (ex) { toastError(ex); }
    finally { setSearching(false); }
  }

  async function pickRepo(id) {
    try {
      const inf = await api("/api/llama/hf/info?id=" + encodeURIComponent(id));
      if (inf.gated) {
        const yes = await askConfirm({
          title: inf.id,
          message: "Gated repo. llama-server needs HF_TOKEN. Continue?",
          confirmLabel: "Continue",
        });
        if (!yes) return;
      }
      setInfo(inf);
    } catch (ex) { toastError(ex); }
  }

  async function startDownload(quant) {
    if (operation.current) return;
    operation.current = true;
    const id = quant ? info.id + ":" + quant : info.id;
    setDl(false);
    setInfo(null);
    setHits([]);
    setNotice(""); setBusy(id);
    try {
      await runOp(() => api("/api/llama/download", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id }),
      }));
      setNotice("Downloaded " + id + ".");
    } catch (ex) { toastError(ex); }
    finally { setBusy(""); operation.current = false; }
  }

  return (
    <PageFrame id="llama-manager" title="llama.cpp" wide>
      <nav className="llama-nav" aria-label="llama.cpp sections">
        <a className={section === "models" ? "active" : ""} href="#/llama/models" aria-current={section === "models" ? "page" : undefined}>Models</a>
        <a className={section === "server" ? "active" : ""} href="#/llama/server" aria-current={section === "server" ? "page" : undefined}>Server</a>
      </nav>
      <div className="llama-status" role="status">
        <div className="llama-state"><span className={"llama-dot " + (checking ? "checking" : ok ? "ready" : "")}></span>
        <span>{checking ? "Checking connection…" : connection.message}</span></div>
        <button type="button" className="btn btn-ghost btn-sm" disabled={checking || saving || !!busy} onClick={refresh}>Test connection</button>
      </div>
      {notice && !busy ? <p role="status" className="settings-desc">{notice}</p> : null}
      {busy ? <p className="llama-working" role="status">Working on {busy}…</p> : null}
      {section === "server" ? (
        <form className="llama-form" noValidate onSubmit={saveUrl}>
          <label>Server URL<input value={url} onChange={(e) => setUrl(e.target.value)} placeholder="http://127.0.0.1:8080" autoComplete="off" /></label>
          <p className="settings-desc">Connection is made from the PiCode server.</p>
          <label><span>API key <span className="settings-desc">(optional)</span></span><input type="password" value={key} onChange={(e) => setKey(e.target.value)} placeholder="Blank keeps saved key" autoComplete="new-password" /></label>
          {formError ? <p role="alert">{formError}</p> : null}
          <div className="llama-actions" data-align-row>
            <button type="submit" className="btn btn-primary btn-sm" disabled={saving || checking || !!busy}>{saving ? "Saving…" : "Save connection"}</button>
            <a className="btn btn-ghost btn-sm" href={DOCS_BASE + "/guide/llama"} target="_blank" rel="noreferrer"><IconDocs /> Setup guide</a>
          </div>
        </form>
      ) : checking && !initialized.current ? (
        <div className="llama-skeleton" aria-label="Loading models"><div /><div /><div /></div>
      ) : !ok ? (
        <div className="llama-empty"><p>Connect a llama.cpp server to manage models.</p><a className="btn btn-primary btn-sm" href="#/llama/server">Configure server</a></div>
      ) : (
        <>
          <div className="llama-toolbar"><h3>Your models</h3><button type="button" className="btn btn-primary btn-sm" disabled={!!busy} onClick={() => { setDl(true); setHits([]); setInfo(null); setSearched(false); }}>Download model</button></div>
          {models.length === 0 ? <p className="side-empty">No models yet. Download your first model to get started.</p> : (
            <ul className="prov-list llama-models">
              {models.map((m) => {
                const on = m.status === "loaded" || m.status === "sleeping";
                return <li key={m.id} className="prov-row">
                  <div className="llama-model-name"><span>{m.id}</span><small>{busy === m.id ? "Working…" : ({ loaded: "Ready", sleeping: "Sleeping", unloaded: "Downloaded", failed: "Failed", loading: "Loading…", downloading: "Downloading…" }[m.status] || m.status)}</small></div>
                  {on ? <button type="button" className="btn btn-ghost btn-sm" disabled={!!busy} onClick={() => unload(m.id)}>Unload</button> : m.status === "downloading" || m.status === "loading" ? null : <button type="button" className="btn btn-ghost btn-sm" disabled={!!busy} onClick={() => load(m.id)}>Load</button>}
                </li>;
              })}
            </ul>
          )}
        </>
      )}

      <Dialog.Root open={dl} onOpenChange={(o) => { if (!o) setDl(false); }}>
        <Dialog.Portal>
          <Dialog.Overlay className="dlg-overlay" />
          <Dialog.Content className="dlg dlg-create" onCloseAutoFocus={(e) => e.preventDefault()}>
            <Dialog.Title className="dlg-title">Download model</Dialog.Title>
            <Dialog.Description className="dlg-body">Search Hugging Face GGUF. The router downloads the file.</Dialog.Description>
            {!info ? (
              <form className="form-new" noValidate onSubmit={search}>
                <input value={q} onChange={(e) => setQ(e.target.value)} placeholder="owner/repo or name" />
                <div className="dlg-actions">
                  <button type="button" className="btn btn-ghost btn-sm" onClick={() => setDl(false)}>Close</button>
                  <button type="submit" className="btn btn-primary btn-sm" disabled={searching}>{searching ? "Searching…" : "Search"}</button>
                </div>
              </form>
            ) : (
              <div>
                <p className="settings-desc">{info.id}</p>
                {(info.quantizations || []).length === 0 ? (
                  <div className="dlg-actions">
                    <button type="button" className="btn btn-ghost btn-sm" onClick={() => setInfo(null)}>Back</button>
                    <button type="button" className="btn btn-primary btn-sm" onClick={() => startDownload("")}>Download</button>
                  </div>
                ) : (
                  <ul className="prov-list">
                    {info.quantizations.map((z) => (
                      <li key={z.name} className="prov-row">
                        <span className="prov-id">{z.name}{z.name === "Q4_K_M" ? " · recommended" : ""}</span>
                        <span className="prov-auth">{bytes(z.size)}</span>
                        <button type="button" className="btn btn-ghost btn-sm" onClick={() => startDownload(z.name)}>Download</button>
                      </li>
                    ))}
                  </ul>
                )}
                {(info.quantizations || []).length ? (
                  <div className="dlg-actions">
                    <button type="button" className="btn btn-ghost btn-sm" onClick={() => setInfo(null)}>Back</button>
                  </div>
                ) : null}
              </div>
            )}
            {!info && searched && !hits.length ? <p className="side-empty">No matching models. Try another name.</p> : null}
            {!info && hits.length ? (
              <Command loop className="prov-pick" shouldFilter={false}>
                <Command.List className="prov-pick-list">
                  {hits.map((h) => (
                    <Command.Item key={h.id} value={h.id} className="cockpit-opt" onSelect={() => pickRepo(h.id)}>
                      <span>{h.id}</span>
                    </Command.Item>
                  ))}
                </Command.List>
              </Command>
            ) : null}
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>
    </PageFrame>
  );
}
