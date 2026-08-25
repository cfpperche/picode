import { useEffect, useState } from "react";
import * as Dialog from "@radix-ui/react-dialog";
import { Command } from "cmdk";
import { api } from "../lib/api.js";
import { toast, toastError } from "../lib/toast.js";
import { askConfirm } from "../lib/confirm.js";

function bytes(n) {
  if (!n) return "";
  if (n < 1024) return n + " B";
  if (n < 1024 * 1024) return (n / 1024).toFixed(1) + " KiB";
  if (n < 1024 * 1024 * 1024) return (n / (1024 * 1024)).toFixed(1) + " MiB";
  return (n / (1024 * 1024 * 1024)).toFixed(2) + " GiB";
}

export default function LlamaPanel({ onRefresh, hideTitle }) {
  const [url, setUrl] = useState("http://127.0.0.1:8080");
  const [ok, setOk] = useState(false);
  const [models, setModels] = useState([]);
  const [busy, setBusy] = useState("");
  const [dl, setDl] = useState(false);
  const [q, setQ] = useState("");
  const [hits, setHits] = useState([]);
  const [info, setInfo] = useState(null);
  const [setup, setSetup] = useState(null);

  async function refresh() {
    try {
      const res = await api("/api/llama");
      setUrl(res.url || "http://127.0.0.1:8080");
      setOk(!!res.ok);
      setModels(res.models || []);
      setSetup(res.setup || null);
    } catch {
      setOk(false);
      setUrl((u) => u || "http://127.0.0.1:8080");
      setModels([]);
    }
  }

  useEffect(() => { refresh(); }, []);

  async function saveUrl(e) {
    if (e && e.preventDefault) e.preventDefault();
    try {
      await api("/api/providers/llama.cpp", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ url, key: "" }),
      });
      toast.ok("Saved router URL.");
      await refresh();
      if (onRefresh) await onRefresh();
    } catch (ex) { toastError(ex); }
  }

  async function startRouter() {
    try {
      await api("/api/llama/start", { method: "POST" });
      toast.ok("Starting llama-server…");
      await saveUrl();
      await new Promise((r) => setTimeout(r, 1200));
      await refresh();
    } catch (ex) { toastError(ex); }
  }

  async function runOp(fn) {
    const tick = setInterval(refresh, 1000);
    try {
      await fn();
      await refresh();
      if (onRefresh) await onRefresh();
    } finally {
      clearInterval(tick);
    }
  }

  async function load(id) {
    const loaded = models.filter((m) => m.status === "loaded" || m.status === "sleeping");
    let unloadOthers = false;
    if (loaded.length) {
      unloadOthers = await askConfirm({
        title: "Load " + id,
        message: "Unload the " + loaded.length + " already loaded first? Cancel keeps them.",
        confirmLabel: "Unload others",
      });
    }
    setBusy(id);
    try {
      await runOp(() => api("/api/llama/load", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id, unloadOthers }),
      }));
      toast.ok("Loaded " + id + ".");
    } catch (ex) { toastError(ex); }
    finally { setBusy(""); }
  }

  async function unload(id) {
    const yes = await askConfirm({
      title: "Unload " + id,
      message: "Does not delete the file.",
      confirmLabel: "Unload",
    });
    if (!yes) return;
    setBusy(id);
    try {
      await runOp(() => api("/api/llama/unload", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id }),
      }));
      toast.ok("Unloaded " + id + ".");
    } catch (ex) { toastError(ex); }
    finally { setBusy(""); }
  }

  async function search(e) {
    e.preventDefault();
    setInfo(null);
    try {
      const res = await api("/api/llama/hf?q=" + encodeURIComponent(q));
      setHits(res.hits || []);
    } catch (ex) { toastError(ex); }
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
    const id = quant ? info.id + ":" + quant : info.id;
    setDl(false);
    setInfo(null);
    setHits([]);
    setBusy(id);
    try {
      await runOp(() => api("/api/llama/download", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id }),
      }));
      toast.ok("Downloaded " + id + ".");
    } catch (ex) { toastError(ex); }
    finally { setBusy(""); }
  }

  return (
    <section id="llama-panel" className="settings-section">
      {hideTitle ? null : <h3>llama.cpp</h3>}
      <form className="form-new" noValidate onSubmit={saveUrl}>
        <div className="set-row" style={{ marginBottom: 0 }}>
          <input value={url} onChange={(e) => setUrl(e.target.value)} placeholder="http://127.0.0.1:8080" />
          <button type="submit" className="btn btn-primary btn-sm">Save</button>
          <button type="button" className="btn btn-ghost btn-sm" onClick={refresh}>Retry</button>
          {ok ? <button type="button" className="btn btn-ghost btn-sm" onClick={() => { setDl(true); setHits([]); setInfo(null); }}>Download</button> : null}
        </div>
      </form>
      <p className="settings-desc">{busy ? busy : ""}</p>
      {setup && !ok ? (
        <ol className="llama-steps">
          <li className={setup.binary ? "done" : ""}>
            <span>{setup.binary ? "Done" : "Next"}</span>
            <div>
              <p>Install llama-server and put it on PATH.</p>
              {!setup.binary ? (
                <p>
                  <a href={setup.releases} target="_blank" rel="noreferrer">Releases</a>
                  {" — Linux/WSL: llama-server. Windows: llama-server.exe."}
                </p>
              ) : <p className="settings-desc">{setup.binary}</p>}
            </div>
          </li>
          <li className={ok ? "done" : ""}>
            <span>{ok ? "Done" : "Next"}</span>
            <div>
              <p>Start the router (no --model flag).</p>
              {!ok ? (
                <div className="dlg-actions" style={{ marginTop: 8 }}>
                  {setup.binary ? <button type="button" className="btn btn-primary btn-sm" onClick={startRouter}>Start</button> : null}
                  <button type="button" className="btn btn-ghost btn-sm" onClick={() => navigator.clipboard.writeText(setup.startCmd).then(() => toast.ok("Copied start command."))}>Copy command</button>
                </div>
              ) : null}
            </div>
          </li>
          <li className={ok && setup.models > 0 ? "done" : ""}>
            <span>{ok && setup.models > 0 ? "Done" : "Next"}</span>
            <div>
              <p>Download or place a GGUF, then Load.</p>
              {ok && setup.models === 0 ? (
                <button type="button" className="btn btn-primary btn-sm" onClick={() => { setDl(true); setHits([]); setInfo(null); }}>Download</button>
              ) : null}
            </div>
          </li>
        </ol>
      ) : !ok ? (
        <p className="side-empty">Router unreachable. Retry after it is running.</p>
      ) : models.length === 0 ? (
        <p className="side-empty">No models. Download one.</p>
      ) : (
        <ul className="prov-list">
          {models.map((m) => {
            const on = m.status === "loaded" || m.status === "sleeping";
            return (
              <li key={m.id} className="prov-row">
                <span className="prov-id">{m.id}</span>
                <span className={"prov-auth" + (on ? " in" : "")}>{busy === m.id ? "working" : m.status}</span>
                {on ? (
                  <button type="button" className="btn btn-ghost btn-sm" disabled={!!busy} onClick={() => unload(m.id)}>Unload</button>
                ) : m.status === "downloading" ? (
                  <span className="prov-auth">downloading</span>
                ) : (
                  <button type="button" className="btn btn-ghost btn-sm" disabled={!!busy} onClick={() => load(m.id)}>Load</button>
                )}
              </li>
            );
          })}
        </ul>
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
                  <button type="submit" className="btn btn-primary btn-sm">Search</button>
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
    </section>
  );
}
