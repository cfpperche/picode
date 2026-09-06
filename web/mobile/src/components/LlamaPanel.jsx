import { useEffect, useRef, useState } from "react";
import * as Dialog from "./MobileSheet.jsx";
import { Command } from "cmdk";
import PageFrame from "./PageFrame.jsx";
import { llamaLoginSchema, parseForm } from "@picode/shared/contracts/schemas.js";
import "./llama.css";
import { mergeLlamaJob, mergeLlamaSnapshot } from "@picode/shared/domain/llamaJobs.js";
import LlamaActivity from "./LlamaActivity.jsx";
import { subscribeFeed } from "@picode/shared/client/feed.js";
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
  const [section, setSection] = useState(() => location.hash.endsWith("/activity") ? "activity" : location.hash.endsWith("/server") ? "server" : "models");
  const initialized = useRef(false);
  const operation = useRef(false);
  useEffect(() => {
    const update = () => setSection(location.hash.endsWith("/activity") ? "activity" : location.hash.endsWith("/server") ? "server" : "models");
    window.addEventListener("hashchange", update);
    if (!/^#\/llama\/(models|server|activity)$/.test(location.hash)) location.replace("#/llama/models");
    return () => window.removeEventListener("hashchange", update);
  }, []);
  const [models, setModels] = useState([]);
  const [jobs, setJobs] = useState([]);
  const [jobsLoading, setJobsLoading] = useState(true);
  const [jobsError, setJobsError] = useState("");
  const [capabilities, setCapabilities] = useState({});
  const [retryRequest, setRetryRequest] = useState(null);
  const refreshing = useRef(false);
  const jobsRefreshing = useRef(false);
  const active = jobs.filter(j => ["queued", "running", "unknown"].includes(j.state));
  const modelBusy = id => active.some(j => j.endpoint === url && (j.model === id || j.replaceOthers));

  async function refreshJobs() {
    if (jobsRefreshing.current) return;
    jobsRefreshing.current = true;
    const startedAt = Date.now();
    try { const res = await api("/api/llama/jobs"); setJobs(current => mergeLlamaSnapshot(current, res.jobs || [], startedAt)); setJobsError(""); }
    catch { setJobsError("Could not update activity."); }
    finally { setJobsLoading(false); jobsRefreshing.current = false; }
  }
  const [busy, setBusy] = useState("");
  const [notice, setNotice] = useState("");
  const [dl, setDl] = useState(false);
  const [q, setQ] = useState("");
  const [hits, setHits] = useState([]);
  const [info, setInfo] = useState(null);

  async function refresh() {
    if (refreshing.current) return;
    refreshing.current = true;
    setChecking(true);
    try {
      const res = await api("/api/llama");
      if (!initialized.current) { setUrl(res.url || "http://127.0.0.1:8080"); initialized.current = true; }
      setOk(!!res.ok);
      setConnection(res.connection || { code: res.ok ? "ready" : "unreachable", message: res.ok ? "Connected" : "Cannot reach the server from PiCode." });
      setModels(res.models || []);
      setCapabilities(res.capabilities || {});
    } catch {
      setOk(false);
      setConnection({ code: "unreachable", message: "Cannot check the connection. Try again." });
    } finally { setChecking(false); refreshing.current = false; }
  }

  useEffect(() => {
    refresh(); refreshJobs();
    const unsubscribe = subscribeFeed(event => {
      if (event.type === "llama.job") {
        const j = event.data;
        if (j?.id) {
          setJobs(current => mergeLlamaJob(current, j));
          setModels(current => current.map(m => m.id === j.model && j.observed ? { ...m, status: j.observed } : m));
          if (!["queued", "running", "unknown"].includes(j.state)) { refresh(); onRefresh?.(); }
        } else refreshJobs();
      }
      if (["feed.open", "feed.reset"].includes(event.type)) { refreshJobs(); refresh(); }
      if (event.type === "feed.down") setJobsError("Live updates paused. Refresh to check activity.");
    });
    const visible = () => { if (!document.hidden) { refreshJobs(); refresh(); } };
    document.addEventListener("visibilitychange", visible);
    return () => { unsubscribe(); document.removeEventListener("visibilitychange", visible); };
  }, []);

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

  async function submit(operationName, payload, requestKey = crypto.randomUUID()) {
    const request = { operationName, payload, requestKey };
    setRetryRequest(request);
    try {
      const res = await api("/api/llama/" + operationName, {
        method: "POST", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ ...payload, requestKey }),
      });
      if (!res.job?.id) throw new Error("The operation was not confirmed. Check activity.");
      setJobs(current => mergeLlamaJob(current, res.job));
      setRetryRequest(null);
      await refreshJobs();
    } catch (error) { setNotice(error.message + " Retry uses the same request."); throw error; }
  }

  async function retrySubmission() {
    if (!retryRequest || operation.current) return;
    operation.current = true; setBusy(retryRequest.payload.id);
    try { await submit(retryRequest.operationName, retryRequest.payload, retryRequest.requestKey); setNotice("Operation accepted. Follow its progress in Activity."); }
    catch { /* the inline request result remains visible */ }
    finally { operation.current = false; setBusy(""); }
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
      await submit("load", { id, unloadOthers });
      setNotice("Operation accepted. Follow its progress in Activity.");
    } catch { /* submit keeps an actionable inline error */ }
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
      await submit("unload", { id });
      setNotice("Operation accepted. Follow its progress in Activity.");
    } catch { /* submit keeps an actionable inline error */ }
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
      await submit("download", { id });
      setNotice("Operation accepted. Follow its progress in Activity.");
    } catch { /* submit keeps an actionable inline error */ }
    finally { setBusy(""); operation.current = false; }
  }

  return (
    <PageFrame id="llama-manager" title="llama.cpp" wide>
      <nav className="llama-nav" aria-label="llama.cpp sections">
        <a className={section === "models" ? "active" : ""} href="#/llama/models" aria-current={section === "models" ? "page" : undefined}>Models</a>
        <a className={section === "server" ? "active" : ""} href="#/llama/server" aria-current={section === "server" ? "page" : undefined}>Server</a>
        <a className={section === "activity" ? "active" : ""} href="#/llama/activity" aria-current={section === "activity" ? "page" : undefined}>Activity</a>
      </nav>
      <div className="llama-status" role="status">
        <div className="llama-state"><span className={"llama-dot " + (checking ? "checking" : ok ? "ready" : "")}></span>
        <span>{checking ? "Checking connection…" : connection.message}</span></div>
        <button type="button" className="btn btn-ghost btn-sm" disabled={checking || saving || !!busy} onClick={refresh}>Test connection</button>
      </div>
      {notice && !busy ? <p role="status" className="settings-desc">{notice}</p> : null}
      {busy ? <p className="llama-working" role="status">Working on {busy}…</p> : null}
      {retryRequest ? <button type="button" className="btn btn-ghost btn-sm" disabled={!!busy} onClick={retrySubmission}>Retry request</button> : null}
      {section !== "activity" && active.length ? <p className="llama-activity-link"><a href="#/llama/activity">{active.some(j => j.state === "unknown") ? "An operation needs attention" : "Model operations in progress"} · View activity</a></p> : null}
      {section === "activity" ? (
        <LlamaActivity jobs={jobs} loading={jobsLoading} error={jobsError} onRefresh={refreshJobs} />
      ) : section === "server" ? (
        <form className="llama-form" noValidate onSubmit={saveUrl}>
          {capabilities.build ? <p className="settings-desc">Server build: {capabilities.build}. {capabilities.events ? "Live model events available." : "Operations use periodic status checks."}</p> : null}
          {ok && !capabilities.cancelDownload ? <p className="settings-desc">Download cancellation is not verified for this server build.</p> : null}
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
                  {on ? <button type="button" className="btn btn-ghost btn-sm" disabled={!!busy || modelBusy(m.id)} onClick={() => unload(m.id)}>Unload</button> : m.status === "downloading" || m.status === "loading" ? null : <button type="button" className="btn btn-ghost btn-sm" disabled={!!busy || modelBusy(m.id)} onClick={() => load(m.id)}>Load</button>}
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
                        <span className="prov-id">{z.name}</span>
                        <span className="prov-auth">{bytes(z.size)}{z.estimatedMemory ? " · ~" + bytes(z.estimatedMemory) + " memory" : ""}</span>
                        <button type="button" className="btn btn-ghost btn-sm" onClick={() => startDownload(z.name)}>Download</button>
                        {z.guidance ? <small className="settings-desc llama-guidance">{z.guidance}</small> : null}
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
