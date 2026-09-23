import { useState } from "react";
import { api } from "@picode/shared/client/api.js";
import { askConfirm } from "../lib/confirm.js";

const labels = { queued: "Waiting", running: "In progress", unknown: "Result unknown", succeeded: "Completed", failed: "Failed", canceled: "Canceled", interrupted: "Interrupted", abandoned: "Abandoned" };
const verbs = { load: "Load", unload: "Unload", download: "Download" };
function bytes(value) { return value < 1048576 ? (value / 1024).toFixed(0) + " KiB" : value < 1073741824 ? (value / 1048576).toFixed(1) + " MiB" : (value / 1073741824).toFixed(2) + " GiB"; }

export default function LlamaActivity({ jobs, loading, error, onRefresh }) {
  const [pending, setPending] = useState("");
  const [actionError, setActionError] = useState("");
  // Abandon (ADR-0083 amendment): stop following a job PiCode cannot
  // resolve, releasing its model; nothing is sent to the server.
  async function abandon(job) {
    const yes = await askConfirm({
      title: "Stop following this operation?",
      message: "PiCode will stop waiting for its result and free the model for other operations. Nothing is sent to the server, so if it is still working there, it keeps going.",
      confirmLabel: "Abandon",
      danger: true,
    });
    if (yes) await action(job, "abandon");
  }
  async function action(job, name) {
    setPending(job.id); setActionError("");
    try { await api(`/api/llama/jobs/${encodeURIComponent(job.id)}/${name}`, { method: "POST" }); await onRefresh(); }
    catch (err) { setActionError(err.message || "Could not update this operation."); }
    finally { setPending(""); }
  }
  return <section aria-label="Model activity">
    <div className="llama-toolbar" data-align-row><h3>Activity</h3><button type="button" className="btn btn-ghost btn-sm" onClick={onRefresh}>Refresh activity</button></div>
    {error || actionError ? <p role="alert">{actionError || error}</p> : null}
    {loading && !jobs.length ? <div className="llama-skeleton" aria-label="Loading activity"><div /><div /><div /></div> : !jobs.length ?
      <div className="llama-empty"><p>No model operations yet.</p><a href="#/llama/models" className="btn btn-primary btn-sm">Manage models</a></div> :
      <ol className="llama-jobs">{jobs.map(job => <li key={job.id} className="llama-job" data-state={job.state}>
        <div className="llama-job-heading"><strong>{verbs[job.operation] || "Model operation"} · {job.model}</strong><span>{labels[job.state] || "Result unknown"}</span></div>
        <p>{job.message}</p>
        {(job.progress || []).map((file, index) => <div className="llama-file" key={`${file.file}-${index}`}>
          <div><span>{file.file}</span><small>{bytes(file.done)}{file.total > 0 ? " / " + bytes(file.total) : " · total unknown"}</small></div>
          <progress aria-label={file.file + " download progress"} max={file.total > 0 ? file.total : undefined} value={file.total > 0 ? file.done : undefined} />
        </div>)}
        {!job.progress?.length && ["queued", "running"].includes(job.state) ? <progress aria-label="Waiting for model operation" /> : null}
        <small className="llama-job-meta">{job.endpoint} · {new Date(job.createdAt).toLocaleString()}</small>
        {job.state === "unknown" ? <div className="llama-job-actions" data-align-row>
          <button type="button" className="btn btn-ghost btn-sm" disabled={pending === job.id} onClick={() => action(job, "reconcile")}>{pending === job.id ? "Checking…" : "Check result"}</button>
          <button type="button" className="btn btn-ghost btn-sm" disabled={pending === job.id} onClick={() => abandon(job)}>Abandon</button>
        </div> : null}
        {job.state === "running" && job.operation === "download" && job.observed === "downloading" ? job.cancelSupported ?
          <button type="button" className="btn btn-ghost btn-sm" disabled={job.cancelRequested || pending === job.id} onClick={() => action(job, "cancel")}>{job.cancelRequested || pending === job.id ? "Cancel requested…" : "Cancel download"}</button> :
          <p className="settings-desc">Cancellation is not verified for this server build.</p> : null}
      </li>)}</ol>}
  </section>;
}
