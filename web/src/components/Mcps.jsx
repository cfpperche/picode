import { useEffect, useState } from "react";
import { api, humanizeError } from "../lib/api.js";
import { askConfirm } from "../lib/confirm.js";
import PageFrame from "./PageFrame.jsx";
import PiSpinner from "./PiSpinner.jsx";

export default function Mcps({ hidden, workspaceId, workspaceName, workspacePath, agentId, agentName, agentWorkPath, agentRunning, onReload }) {
  const [data, setData] = useState(null);
  const [scope, setScope] = useState("user");
  const [job, setJob] = useState(null);
  const [form, setForm] = useState(emptyForm());

  function listURL() {
    const q = [];
    if (workspaceId) q.push("workspace=" + encodeURIComponent(workspaceId));
    if (agentId) q.push("agent=" + encodeURIComponent(agentId));
    return "/api/mcp" + (q.length ? "?" + q.join("&") : "");
  }

  async function load() {
    try { setData(await api(listURL())); }
    catch { setData({ adapter: { installed: false }, servers: [], presets: [], layers: [] }); }
  }

  useEffect(() => { if (!hidden) load(); }, [hidden, workspaceId, agentId]);
  useEffect(() => {
    if (!workspaceId && scope === "project") setScope("user");
    if (!agentWorkPath && scope === "agent") setScope("user");
  }, [workspaceId, agentWorkPath, scope]);

  const installed = !!(data && data.adapter && data.adapter.installed);
  const servers = (data && data.servers) || [];
  const presets = (data && data.presets) || [];

  async function runJob(action, label, fn) {
    if (job) return;
    setJob({ action, label, step: 0, error: "", done: false });
    const tick = startJobTick(setJob, agentRunning ? 2 : 2);
    try {
      const next = await fn();
      setJob((j) => j && { ...j, step: 1 });
      if (agentRunning && onReload) {
        await onReload();
      }
      setJob((j) => j && { ...j, step: 2, done: true });
      setData(next);
      setTimeout(() => setJob(null), 480);
    } catch (err) {
      setJob((j) => j && { ...j, error: humanizeError(err.message || String(err)) });
    } finally {
      clearInterval(tick);
    }
  }

  function body(extra) {
    return {
      scope,
      workspaceId: workspaceId || undefined,
      agentId: agentId || undefined,
      ...extra,
    };
  }

  async function addServer(entry) {
    const name = (entry.name || form.name).trim();
    if (!name || !installed) return;
    await runJob("add", name, () => api("/api/mcp", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body({
        name,
        command: entry.command || (form.kind === "stdio" ? form.command : ""),
        args: entry.args || (form.kind === "stdio" ? splitArgs(form.args) : []),
        url: entry.url || (form.kind === "url" ? form.url : ""),
        auth: entry.auth || "",
      })),
    }));
    setForm(emptyForm());
  }

  async function toggle(s) {
    if (!installed) return;
    await runJob("toggle", s.name, () => api("/api/mcp", {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body({ name: s.name, scope: writeScope(s, scope), disabled: !s.disabled })),
    }));
  }

  async function remove(s) {
    if (!installed || !s.owned) return;
    const ok = await askConfirm({
      title: "Remove " + s.name,
      message: "Delete this server from " + (s.path || "the config") + ".",
      confirmLabel: "Remove",
      danger: true,
    });
    if (!ok) return;
    const q = new URLSearchParams({
      name: s.name,
      scope: writeScope(s, scope),
    });
    if (workspaceId) q.set("workspace", workspaceId);
    if (agentId) q.set("agent", agentId);
    await runJob("remove", s.name, () => api("/api/mcp?" + q.toString(), { method: "DELETE" }));
  }

  return (
    <PageFrame id="mcps-view" title="MCPs" hidden={hidden}>
      {!installed ? (
        <p className="pkg-fine">
          Pi has no native MCP. Install <code>npm:pi-mcp-adapter</code> first.
          {" "}<a className="settings-link" href="#/packages">Open packages</a>
        </p>
      ) : (
        <p className="pkg-fine">Writes the adapter&apos;s own files. Restart picks them up.</p>
      )}

      <div className="pkg-scope" data-align-row role="radiogroup" aria-label="MCP scope">
        <button type="button" role="radio" className="pkg-scope-btn" aria-checked={scope === "user"} onClick={() => setScope("user")}>This machine</button>
        <button
          type="button"
          role="radio"
          className="pkg-scope-btn"
          aria-checked={scope === "project"}
          disabled={!workspaceId}
          title={workspaceId ? "Writes .mcp.json in " + (workspaceName || "this folder") : "Select an agent first"}
          onClick={() => workspaceId && setScope("project")}
        >This workspace</button>
        <button
          type="button"
          role="radio"
          className="pkg-scope-btn"
          aria-checked={scope === "agent"}
          disabled={!agentWorkPath}
          title={agentWorkPath ? "Writes .pi/mcp.json in this agent folder" : "This agent shares its folder — use This workspace"}
          onClick={() => agentWorkPath && setScope("agent")}
        >This agent</button>
      </div>
      {scope === "project" && workspacePath ? <p className="pkg-fine">Target: {workspacePath}/.mcp.json</p> : null}
      {scope === "agent" && agentWorkPath ? <p className="pkg-fine">Target: {agentWorkPath}/.pi/mcp.json</p> : null}

      <section className="pkg-installed">
        <h3>Servers</h3>
        {servers.length === 0 ? (
          <p className="pkg-fine">No MCP servers.</p>
        ) : (
          <ul className="mcp-list">
            {servers.map((s) => (
              <li key={s.layer + ":" + s.name} className={"mcp-row" + (s.disabled ? " off" : "")}>
                <div className="mcp-row-main">
                  <span className="pkg-scope-tag">{scopeLabel(s, workspaceName, agentName)}</span>
                  <strong className="mcp-name">{s.name}</strong>
                  <code className="mcp-target">{targetOf(s)}</code>
                </div>
                <div className="mcp-row-actions">
                  <button
                    type="button"
                    role="switch"
                    aria-checked={!s.disabled}
                    className="pkg-scope-btn"
                    disabled={!installed || !!job}
                    onClick={() => toggle(s)}
                  >{s.disabled ? "Off" : "On"}</button>
                  {s.owned ? (
                    <button type="button" className="btn btn-ghost btn-sm" disabled={!installed || !!job} onClick={() => remove(s)}>Remove</button>
                  ) : null}
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>

      {installed ? (
        <section className="mcp-add">
          <h3>Add</h3>
          {presets.length ? (
            <div className="mcp-presets" data-align-row>
              {presets.map((p) => (
                <button
                  key={p.id}
                  type="button"
                  className="pkg-scope-btn"
                  disabled={!!job}
                  title={p.summary}
                  onClick={() => addServer({ name: p.id, ...p.entry })}
                >{p.name}</button>
              ))}
            </div>
          ) : null}
          <form className="mcp-form" noValidate onSubmit={(e) => { e.preventDefault(); addServer({}); }}>
            <input className="dlg-input" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="name" aria-label="Server name" disabled={!!job} />
            <div className="pkg-scope" role="radiogroup" aria-label="Transport">
              <button type="button" role="radio" className="pkg-scope-btn" aria-checked={form.kind === "stdio"} onClick={() => setForm({ ...form, kind: "stdio" })}>Command</button>
              <button type="button" role="radio" className="pkg-scope-btn" aria-checked={form.kind === "url"} onClick={() => setForm({ ...form, kind: "url" })}>URL</button>
            </div>
            {form.kind === "stdio" ? (
              <>
                <input className="dlg-input" value={form.command} onChange={(e) => setForm({ ...form, command: e.target.value })} placeholder="npx" aria-label="Command" disabled={!!job} />
                <input className="dlg-input" value={form.args} onChange={(e) => setForm({ ...form, args: e.target.value })} placeholder="-y package" aria-label="Args" disabled={!!job} />
              </>
            ) : (
              <input className="dlg-input" value={form.url} onChange={(e) => setForm({ ...form, url: e.target.value })} placeholder="https://…" aria-label="Server URL" disabled={!!job} />
            )}
            <button type="submit" className="btn btn-primary btn-sm" disabled={!!job || !form.name.trim() || (scope === "project" && !workspaceId) || (scope === "agent" && !agentWorkPath)}>Add</button>
          </form>
        </section>
      ) : null}

      {job ? <JobOverlay job={job} running={agentRunning} onClose={() => setJob(null)} /> : null}
    </PageFrame>
  );
}

function emptyForm() {
  return { name: "", kind: "url", command: "", args: "", url: "" };
}

function splitArgs(s) {
  return String(s || "").trim().split(/\s+/).filter(Boolean);
}

function writeScope(s, fallback) {
  if (s && s.scope && s.scope !== "import") return s.scope;
  return fallback || "user";
}

function scopeLabel(s, workspaceName, agentName) {
  if (s.scope === "project") return workspaceName || "workspace";
  if (s.scope === "agent") return agentName || "agent";
  if (s.scope === "import") return "shared";
  return "machine";
}

function targetOf(s) {
  if (s.url) return s.url;
  const cmd = [s.command].concat(s.args || []).filter(Boolean).join(" ");
  return cmd || s.transport;
}

function startJobTick(setJob, stepCount) {
  return setInterval(() => {
    setJob((j) => {
      if (!j || j.error || j.done) return j;
      if (j.step < stepCount - 1) return { ...j, step: j.step + 1 };
      return j;
    });
  }, 420);
}

function JobOverlay({ job, running, onClose }) {
  const steps = [
    { id: "write", label: (job.action === "remove" ? "Remove " : job.action === "toggle" ? "Update " : "Write ") + job.label },
    { id: "reload", label: running ? "Reload this agent" : "Next start picks this up" },
  ];
  return (
    <div className="pkg-job" role="alertdialog" aria-modal="true" aria-labelledby="mcp-job-title">
      <div className="pkg-job-card">
        <h3 id="mcp-job-title">MCP</h3>
        <ol className="pkg-job-steps">
          {steps.map((s, i) => {
            let st = "todo";
            if (job.error && i === job.step) st = "err";
            else if (i < job.step) st = "done";
            else if (i === job.step) st = "run";
            return (
              <li key={s.id} className={"pkg-job-step " + st}>
                <span className="pkg-job-mark" aria-hidden="true">
                  {st === "run" ? <PiSpinner title="Working" /> : st === "done" ? "✓" : st === "err" ? "!" : "○"}
                </span>
                <code>{s.label}</code>
              </li>
            );
          })}
        </ol>
        {job.error ? (
          <>
            <p className="pkg-job-err">{job.error}</p>
            <button type="button" className="btn btn-primary btn-sm" onClick={onClose}>Close</button>
          </>
        ) : (
          <p className="pkg-fine">Writing the adapter file.</p>
        )}
      </div>
    </div>
  );
}
