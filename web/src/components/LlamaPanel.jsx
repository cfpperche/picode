import { useEffect, useState } from "react";
import { api } from "../lib/api.js";
import { toast, toastError } from "../lib/toast.js";
import { askConfirm } from "../lib/confirm.js";

export default function LlamaPanel() {
  const [url, setUrl] = useState("");
  const [ok, setOk] = useState(false);
  const [err, setErr] = useState("");
  const [models, setModels] = useState([]);
  const [busy, setBusy] = useState("");

  async function refresh() {
    setErr("");
    try {
      const res = await api("/api/llama");
      setUrl(res.url || "");
      setOk(!!res.ok);
      setErr(res.error || "");
      setModels(res.models || []);
    } catch (ex) {
      setOk(false);
      setErr(ex.message || "unreachable");
      setModels([]);
    }
  }

  useEffect(() => { refresh(); }, []);

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
      await api("/api/llama/load", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id, unloadOthers }),
      });
      toast.ok("Loading " + id + ".");
      await refresh();
    } catch (ex) { toastError(ex); }
    finally { setBusy(""); }
  }

  async function unload(id) {
    const ok = await askConfirm({
      title: "Unload " + id,
      message: "Does not delete the file.",
      confirmLabel: "Unload",
    });
    if (!ok) return;
    setBusy(id);
    try {
      await api("/api/llama/unload", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id }),
      });
      toast.ok("Unloaded " + id + ".");
      await refresh();
    } catch (ex) { toastError(ex); }
    finally { setBusy(""); }
  }

  return (
    <section id="llama-panel" className="settings-section">
      <div className="set-row">
        <h3>llama.cpp</h3>
        <button type="button" className="btn btn-ghost btn-sm" onClick={refresh}>Retry</button>
      </div>
      <p className="settings-desc">{url || "No router URL"}{ok ? "" : err ? " — " + err : " — unreachable"}</p>
      {!ok ? (
        <p className="side-empty">Start llama-server, then Retry.</p>
      ) : models.length === 0 ? (
        <p className="side-empty">No models in the router.</p>
      ) : (
        <ul className="prov-list">
          {models.map((m) => {
            const on = m.status === "loaded" || m.status === "sleeping";
            return (
              <li key={m.id} className="prov-row">
                <span className="prov-id">{m.id}</span>
                <span className={"prov-auth" + (on ? " in" : "")}>{m.status}</span>
                {on ? (
                  <button type="button" className="btn btn-ghost btn-sm" disabled={!!busy} onClick={() => unload(m.id)}>Unload</button>
                ) : (
                  <button type="button" className="btn btn-ghost btn-sm" disabled={!!busy} onClick={() => load(m.id)}>Load</button>
                )}
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}
