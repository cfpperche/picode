import { useEffect, useMemo, useState } from "react";
import { api, humanizeError } from "../lib/api.js";
import { askConfirm } from "../lib/confirm.js";
import PageFrame from "./PageFrame.jsx";

export default function Packages({ hidden, workspaceId, workspaceName, workspacePath, agentId, agentName }) {
  const [data, setData] = useState(null);
  const [source, setSource] = useState("");
  const [scope, setScope] = useState("user");
  const [q, setQ] = useState("");
  const [hits, setHits] = useState([]);
  const [searching, setSearching] = useState(true);
  const [job, setJob] = useState(null);

  function listURL() {
    const q = [];
    if (workspaceId) q.push("workspace=" + encodeURIComponent(workspaceId));
    if (agentId) q.push("agent=" + encodeURIComponent(agentId));
    return "/api/packages" + (q.length ? "?" + q.join("&") : "");
  }

  async function load() {
    try { setData(await api(listURL())); }
    catch { setData({ packages: [], capabilities: {}, gallery: "https://pi.dev/packages" }); }
  }

  useEffect(() => { if (!hidden) load(); }, [hidden, workspaceId, agentId]);
  useEffect(() => {
    if (!workspaceId && scope === "project") setScope("user");
    if (!agentId && scope === "agent") setScope("user");
  }, [workspaceId, agentId, scope]);

  useEffect(() => {
    if (hidden) return;
    const t = setTimeout(async () => {
      setSearching(true);
      try {
        const page = await api("/api/packages/gallery?q=" + encodeURIComponent(q.trim()));
        setHits(page.hits || []);
      } catch { setHits([]); }
      finally { setSearching(false); }
    }, q ? 280 : 0);
    return () => clearTimeout(t);
  }, [hidden, q]);

  const installed = useMemo(() => {
    const s = new Set();
    const want = scope === "project" ? "project" : scope === "agent" ? "agent" : "user";
    for (const p of (data && data.packages) || []) {
      if (p.scope === want) s.add(p.source);
    }
    return s;
  }, [data, scope]);

  async function installSource(src) {
    const nextSrc = (src || "").trim();
    if (!nextSrc || job) return;
    if (scope === "project" && !workspaceId) return;
    if (scope === "agent" && !agentId) return;
    setJob({ action: "install", source: nextSrc, scope, cwd: scope === "project" ? workspacePath : "", step: 0, error: "" });
    try {
      const next = await api("/api/packages", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ source: nextSrc, scope, workspaceId: workspaceId || undefined, agentId: agentId || undefined }),
      });
      setJob((j) => j && { ...j, step: 1 });
      setData(next);
      setSource("");
      setJob((j) => j && { ...j, step: 2 });
      setTimeout(() => setJob(null), 600);
    } catch (err) {
      setJob((j) => j && { ...j, error: humanizeError(err.message || String(err)) });
    }
  }

  async function remove(pkg) {
    const ok = await askConfirm({
      title: "Remove package",
      message: "Remove " + pkg.source + " from pi? This does not uninstall pi itself.",
      confirmLabel: "Remove",
      danger: true,
    });
    if (!ok || job) return;
    const sc = pkg.scope === "project" ? "project" : pkg.scope === "agent" ? "agent" : "user";
    setJob({ action: "remove", source: pkg.source, scope: sc, cwd: sc === "project" ? workspacePath : "", step: 0, error: "" });
    try {
      let url = "/api/packages?source=" + encodeURIComponent(pkg.source) + "&scope=" + sc;
      if (workspaceId) url += "&workspace=" + encodeURIComponent(workspaceId);
      if (agentId) url += "&agent=" + encodeURIComponent(agentId);
      const next = await api(url, { method: "DELETE" });
      setJob((j) => j && { ...j, step: 1 });
      setData(next);
      setJob((j) => j && { ...j, step: 2 });
      setTimeout(() => setJob(null), 600);
    } catch (err) {
      setJob((j) => j && { ...j, error: humanizeError(err.message || String(err)) });
    }
  }

  const list = data && data.packages ? data.packages : [];
  const gallery = (data && data.gallery) || "https://pi.dev/packages";

  return (
    <PageFrame id="packages-view" title="Packages" hidden={hidden} wide>
      <form className="pkg-by-source" noValidate onSubmit={(e) => { e.preventDefault(); installSource(source); }}>
        <input
          className="dlg-input"
          value={source}
          onChange={(e) => setSource(e.target.value)}
          placeholder="npm:pkg  ·  git:github.com/user/repo  ·  ./path"
          disabled={!!job}
          aria-label="Package source"
        />
        <button type="submit" className="btn btn-primary btn-sm" disabled={!!job || !source.trim() || (scope === "project" && !workspaceId) || (scope === "agent" && !agentId)}>Install</button>
      </form>
      <div className="pkg-scope" data-align-row role="radiogroup" aria-label="Install scope">
        <button type="button" role="radio" className="pkg-scope-btn" aria-checked={scope === "user"} onClick={() => setScope("user")}>This machine</button>
        <button
          type="button"
          role="radio"
          className="pkg-scope-btn"
          aria-checked={scope === "project"}
          disabled={!workspaceId}
          title={workspaceId ? "Writes .pi/settings.json in this workspace" : "Select an agent first"}
          onClick={() => workspaceId && setScope("project")}
        >This workspace</button>
        <button
          type="button"
          role="radio"
          className="pkg-scope-btn"
          aria-checked={scope === "agent"}
          disabled={!agentId}
          title={agentId ? "Only this agent, every session" : "Select an agent first"}
          onClick={() => agentId && setScope("agent")}
        >This agent</button>
      </div>
      {agentId ? (
        <label className="pkg-fine" style={{ display: "flex", gap: 8, alignItems: "center" }}>
          <input
            type="checkbox"
            checked={!!(data && data.isolated)}
            disabled={!!job}
            onChange={async (e) => {
              const on = e.target.checked;
              try {
                await api("/api/agents/" + agentId, {
                  method: "PATCH",
                  headers: { "Content-Type": "application/json" },
                  body: JSON.stringify({ packagesIsolated: on }),
                });
                await load();
              } catch (err) { /* keep checkbox from lying */ await load(); }
            }}
          />
          Only this agent's packages (skip machine and folder). Restart to apply.
        </label>
      ) : null}
      <p className="pkg-fine">Packages run with full access. Only install what you review.{scope === "project" && workspaceName ? " Target: " + workspaceName + "/.pi" : ""}{scope === "agent" && agentName ? " Target: " + agentName + " (next start)" : ""}</p>

      <section className="pkg-toolbar" data-align-row>
        <input
          className="pkg-search"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="Filter packages…"
          aria-label="Search gallery"
        />
        <span className="pkg-count">{searching && !hits.length ? "Loading…" : searching ? "Updating…" : hits.length ? hits.length + " shown" : "No matches"}</span>
        <a className="settings-link" href={gallery} target="_blank" rel="noopener noreferrer">pi.dev ↗</a>
      </section>

      {list.length > 0 ? (
        <section className="pkg-installed">
          <h3>Installed</h3>
          <ul className="pkg-chips">
            {list.map((p) => (
              <li key={p.scope + ":" + p.source} className="pkg-chip">
                <span className="pkg-scope-tag">{p.scope === "project" ? (workspaceName || "workspace") : p.scope === "agent" ? (agentName || "agent") : "machine"}</span>
                <span className="pkg-src">{p.source}</span>
                <button type="button" className="btn btn-ghost btn-sm" onClick={() => remove(p)} disabled={!!job}>Remove</button>
              </li>
            ))}
          </ul>
        </section>
      ) : null}

      <ul className="pkg-grid" aria-busy={searching && !hits.length}>
        {searching && !hits.length ? Array.from({ length: 6 }, (_, i) => (
          <li key={"skel-" + i} className="pkg-card pkg-skel" aria-hidden="true">
            <div className="pkg-preview">
              <div className="pkg-preview-frame"><span /><span /><span /></div>
            </div>
            <div className="pkg-card-body">
              <div className="skel-line w-50" />
              <div className="skel-line w-90" />
              <div className="skel-line w-70" />
              <div className="skel-line w-40" />
              <div className="skel-line w-80" />
            </div>
          </li>
        )) : null}
        {hits.map((h) => {
          const on = installed.has(h.source);
          return (
            <li key={h.source} className="pkg-card">
              <div className={"pkg-preview" + (h.image ? " has-media" : "")} aria-hidden="true">
                <div className="pkg-preview-frame">
                  {h.image ? <img src={h.image} alt="" loading="lazy" /> : <><span /><span /><span /></>}
                </div>
              </div>
              <div className="pkg-card-body">
              <div className="pkg-card-head">
                <span className="pkg-card-name">{h.name}</span>
                {h.kind ? <span className="pkg-type">{h.kind}</span> : null}
              </div>
              {h.description ? <p className="pkg-card-desc">{h.description}</p> : <p className="pkg-card-desc"> </p>}
              <div className="pkg-card-meta">
                {h.publisher ? <span>{h.publisher}</span> : null}
                {h.downloads ? <span>{fmtDown(h.downloads)}</span> : null}
                {h.updated ? <span>{fmtAge(h.updated)}</span> : null}
                {h.version ? <span>{h.version}</span> : null}
              </div>
              <div className="pkg-card-foot">
                <code className="pkg-cmd">pi install {h.source}</code>
                <button
                  type="button"
                  className="btn btn-primary btn-sm"
                  disabled={!!job || on}
                  onClick={() => installSource(h.source)}
                >
                  {on ? "Installed" : "Install"}
                </button>
              </div>
              </div>
            </li>
          );
        })}
      </ul>

      {job ? <JobOverlay job={job} onClose={() => setJob(null)} /> : null}
    </PageFrame>
  );
}

function jobSteps(job) {
  const bin = job.action === "remove" ? "remove" : "install";
  if (job.scope === "agent") {
    return [
      { id: "run", label: (bin === "remove" ? "Drop from this agent: " : "Attach to this agent: ") + job.source },
      { id: "list", label: "Reload. Takes effect the next time this agent starts." },
    ];
  }
  const local = job.scope === "project";
  const cmd = local
    ? "pi " + bin + " -l " + job.source + " --no-approve"
    : "pi " + bin + " " + job.source + " --no-approve";
  const steps = [{ id: "run", label: cmd }];
  if (local && job.cwd) steps[0].label += "  (cwd " + job.cwd + ")";
  steps.push({ id: "list", label: "Reload installed packages" });
  return steps;
}

function JobOverlay({ job, onClose }) {
  const steps = jobSteps(job);
  const title = job.action === "remove" ? "Removing package" : "Installing package";
  return (
    <div className="pkg-job" role="alertdialog" aria-modal="true" aria-labelledby="pkg-job-title">
      <div className="pkg-job-card">
        <h3 id="pkg-job-title">{title}</h3>
        <p className="pkg-job-src">{job.source}</p>
        <ol className="pkg-job-steps">
          {steps.map((s, i) => {
            let st = "todo";
            if (job.error && i === job.step) st = "err";
            else if (i < job.step) st = "done";
            else if (i === job.step) st = "run";
            return (
              <li key={s.id} className={"pkg-job-step " + st}>
                <span className="pkg-job-mark" aria-hidden="true">{st === "done" ? "✓" : st === "run" ? "●" : st === "err" ? "!" : "○"}</span>
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
          <p className="pkg-fine">Stays here until pi finishes. npm can take a minute.</p>
        )}
      </div>
    </div>
  );
}

function fmtDown(n) {
  if (!n) return "";
  if (n >= 1e6) return trimNum(n / 1e6) + "M/mo";
  if (n >= 1000) return trimNum(n / 1000) + "k/mo";
  return n + "/mo";
}

function trimNum(n) {
  return n.toFixed(n >= 10 ? 0 : 1).replace(/\.0$/, "");
}

function fmtAge(iso) {
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return "";
  const d = Date.now() - t;
  const day = 86400000;
  if (d < day) return "today";
  if (d < 2 * day) return "1d ago";
  if (d < 30 * day) return Math.floor(d / day) + "d ago";
  if (d < 365 * day) return Math.floor(d / (30 * day)) + "mo ago";
  return Math.floor(d / (365 * day)) + "y ago";
}
