import { useEffect, useRef, useState } from "react";
import * as Dialog from "@radix-ui/react-dialog";
import { api, humanizeError } from "../lib/api.js";
import { askConfirm } from "../lib/confirm.js";
import { mcpAddSchema, pairsToMap, parseForm } from "../lib/schemas.js";
import { toast } from "../lib/toast.js";
import { paneContext } from "../lib/tree.js";
import PageFrame from "./PageFrame.jsx";
import PiSpinner from "./PiSpinner.jsx";

export default function Mcps({ hidden, workspaceId, workspaceName, workspacePath, agentId, agentName, agentWorkPath, agentRunning, onReload }) {
  const [data, setData] = useState(null);
  const [scope, setScope] = useState("user");
  const [job, setJob] = useState(null);
  const [form, setForm] = useState(emptyForm());
  const [formError, setFormError] = useState("");
  const [pickOpen, setPickOpen] = useState(false);
  const signCtl = useRef({ id: "", stop: false });

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

  useEffect(() => { if (!hidden) load(); }, [hidden, workspaceId, agentId, agentRunning]);
  useEffect(() => {
    if (hidden || !agentRunning) return;
    const t = setInterval(() => { load(); }, 2500);
    return () => clearInterval(t);
  }, [hidden, agentRunning, workspaceId, agentId]);
  useEffect(() => {
    if (!workspaceId && scope === "project") setScope("user");
    if (!agentWorkPath && scope === "agent") setScope("user");
  }, [workspaceId, agentWorkPath, scope]);

  const loading = !hidden && data === null;
  const installed = !!(data && data.adapter && data.adapter.installed);
  const servers = (data && data.servers) || [];
  const presets = (data && data.presets) || [];
  const imported = (data && data.imports) || [];
  const mirrored = servers.filter((s) => s.scope === "import");
  const canProject = !!workspaceId;
  const canAgent = !!agentWorkPath;
  const showScopes = canProject || canAgent;

  async function runJob(action, label, fn) {
    if (job) return null;
    setJob({ action, label, step: 0, error: "", done: false });
    const tick = startJobTick(setJob, 2);
    try {
      const next = await fn();
      setJob((j) => j && { ...j, step: 1 });
      if (agentRunning && onReload) {
        await onReload();
      }
      setJob((j) => j && { ...j, step: 2, done: true });
      setData(next);
      setTimeout(() => setJob(null), 480);
      return next;
    } catch (err) {
      setJob((j) => j && { ...j, error: humanizeError(err.message || String(err)) });
      return null;
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
    if (!installed) return;
    let name = "";
    let auth = "";
    if (entry && entry.name) {
      name = entry.name;
      auth = entry.auth || "";
      const next = await runJob("add", entry.name, () => api("/api/mcp", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body({
          name: entry.name,
          command: entry.command || "",
          args: entry.args || [],
          url: entry.url || "",
          auth: entry.auth || "",
        })),
      }));
      if (!next) return;
    } else {
      const parsed = parseForm(mcpAddSchema, form);
      if (!parsed.ok) { setFormError(parsed.error); return; }
      const v = parsed.value;
      const pairs = pairsToMap(v.pairs);
      setFormError("");
      name = v.name;
      auth = v.kind === "url" ? v.auth : "";
      const next = await runJob("add", v.name, () => api("/api/mcp", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body({
          name: v.name,
          command: v.kind === "stdio" ? v.command : "",
          args: v.kind === "stdio" ? splitArgs(v.args) : [],
          url: v.kind === "url" ? v.url : "",
          env: v.kind === "stdio" ? pairs : undefined,
          headers: v.kind === "url" ? pairs : undefined,
          auth: v.kind === "url" ? v.auth : "",
          bearerToken: v.kind === "url" && v.auth === "bearer" ? v.token : "",
        })),
      }));
      if (!next) return;
      setForm(emptyForm());
    }
    if (auth === "oauth") await signIn({ name, auth: "oauth", disabled: false }, { quiet: true });
  }

  async function toggle(s) {
    if (!installed) return;
    const turningOn = !!s.disabled;
    const next = await runJob("toggle", s.name, () => api("/api/mcp", {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body({ name: s.name, scope: writeScope(s, scope), disabled: !s.disabled })),
    }));
    if (!next) return;
    if (turningOn) await signIn({ ...s, disabled: false }, { quiet: true });
  }

  function importHosts() {
    if (!installed) return;
    const found = (data && data.found) || [];
    if (!found.length) {
      toast.info("No other apps with servers.");
      return;
    }
    setPickOpen(true);
  }

  async function applyPicks(picks) {
    setPickOpen(false);
    await runJob("import", "apps", () => api("/api/mcp/import", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body({ picks })),
    }));
  }

  function canSignIn(s) {
    if (!s || s.disabled) return false;
    if (s.live === "live") return false;
    if (s.live === "signin") return true;
    return s.auth === "oauth" && !s.signedIn;
  }

  async function signIn(s, opts) {
    if (!canSignIn(s)) return;
    const quiet = !!(opts && opts.quiet);
    signCtl.current = { id: "", stop: false };
    setJob({ action: "signin", label: s.name, step: 0, error: "", done: false });
    try {
      const res = await api("/api/mcp/auth", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ agentId: agentId || undefined, name: s.name, workspaceId: workspaceId || undefined }),
      });
      if (res && res.ok && !res.id) {
        setJob(null);
        toast.ok("Signed in to " + s.name + ".");
        await load();
        return;
      }
      signCtl.current.id = res.id;
      setJob({ action: "signin", label: s.name, step: 1, error: "", done: false });
      let opened = false;
      const t0 = Date.now();
      while (Date.now() - t0 < 5 * 60 * 1000) {
        if (signCtl.current.stop) {
          await api("/api/mcp/auth/reply", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ id: res.id, cancelled: true }),
          }).catch(() => {});
          setJob(null);
          return;
        }
        const st = await api("/api/mcp/auth/status?id=" + encodeURIComponent(res.id));
        if (st && st.url && !opened) {
          opened = true;
          window.open(st.url, "picode-mcp-auth");
        }
        if (st && st.ok) {
          setJob(null);
          toast.ok("Signed in to " + s.name + ".");
          await load();
          return;
        }
        if (st && st.error) throw new Error(st.error);
        if (st && !st.pending && !st.ok) throw new Error("Sign-in expired.");
        await new Promise((r) => setTimeout(r, 250));
      }
      throw new Error("sign-in timed out");
    } catch (err) {
      const msg = humanizeError(err.message || String(err));
      if (quiet) {
        setJob(null);
        toast.error(msg);
        return;
      }
      setJob((j) => (j && j.action === "signin" ? { ...j, error: msg } : j));
      toast.error(msg);
    }
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
    <PageFrame id="mcps-view" title="MCPs" context={paneContext(agentName, workspaceName)} hidden={hidden}>
      {loading ? (
        <div className="mcp-skel" aria-hidden="true">
          <div className="skel-line w-70" />
          <div className="skel-line w-40" />
        </div>
      ) : !installed ? (
        <div className="mcp-empty">
          <p>Install the MCP adapter to add servers.</p>
          <a className="btn btn-primary" href="#/packages">Open packages</a>
        </div>
      ) : (
        <>
          <div className="mcp-toolbar" data-align-row>
            <button type="button" className="btn btn-ghost" disabled={!!job} onClick={importHosts}>Use from…</button>
          </div>
          {imported.length ? (
            <p className="pkg-fine">Using {imported.map(hostLabel).join(" · ")}</p>
          ) : null}
          {imported.length && !mirrored.length ? (
            <p className="pkg-fine">No servers in those apps.</p>
          ) : null}
          {showScopes ? (
            <div className="pkg-scope" data-align-row role="radiogroup" aria-label="Where to save">
              <button type="button" role="radio" className="pkg-scope-btn" aria-checked={scope === "user"} onClick={() => setScope("user")}>This machine</button>
              {canProject ? (
                <button
                  type="button"
                  role="radio"
                  className="pkg-scope-btn"
                  aria-checked={scope === "project"}
                  title={"Saves in " + (workspaceName || "this folder")}
                  onClick={() => setScope("project")}
                >{workspaceName || "This workspace"}</button>
              ) : null}
              {canAgent ? (
                <button
                  type="button"
                  role="radio"
                  className="pkg-scope-btn"
                  aria-checked={scope === "agent"}
                  title={"Saves with " + (agentName || "this agent")}
                  onClick={() => setScope("agent")}
                >This agent</button>
              ) : null}
            </div>
          ) : null}

          {servers.length > 0 ? (
            <section className="pkg-installed">
              <h3>Servers</h3>
              <ul className="mcp-list">
                {servers.map((s) => {
                  const word = rowLive(s);
                  return (
                  <li key={s.layer + ":" + s.name} className={"mcp-row" + (s.disabled ? " off" : "")}>
                    <div className="mcp-row-main">
                      <span className="pkg-scope-tag">{scopeLabel(s, workspaceName, agentName)}</span>
                      <strong className="mcp-name">{s.name}</strong>
                      {word && word !== "signin" ? (
                        <span className={"mcp-live " + word} title={liveTitle(word)}>{liveLabel(word)}</span>
                      ) : null}
                      <code className="mcp-target">{targetOf(s)}</code>
                    </div>
                    <div className="mcp-row-actions" data-align-row>
                      {canSignIn(s) ? (
                        <button type="button" className="btn btn-ghost" disabled={!!job} onClick={() => signIn(s)}>Sign in</button>
                      ) : null}
                      <button
                        type="button"
                        role="switch"
                        aria-checked={!s.disabled}
                        className="pkg-scope-btn"
                        disabled={!!job}
                        onClick={() => toggle(s)}
                      >{s.disabled ? "Off" : "On"}</button>
                      {s.owned ? (
                        <button type="button" className="btn btn-ghost btn-sm" disabled={!!job} onClick={() => remove(s)}>Remove</button>
                      ) : null}
                    </div>
                  </li>
                  );
                })}
              </ul>
            </section>
          ) : null}

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
              <div className="mcp-form-row" data-align-row>
                <input className="dlg-input" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="Name" aria-label="Server name" disabled={!!job} />
                <div className="pkg-scope" role="radiogroup" aria-label="How it connects">
                  <button type="button" role="radio" className="pkg-scope-btn" aria-checked={form.kind === "stdio"} onClick={() => setForm({ ...form, kind: "stdio", auth: "", token: "" })}>Command</button>
                  <button type="button" role="radio" className="pkg-scope-btn" aria-checked={form.kind === "url"} onClick={() => setForm({ ...form, kind: "url" })}>URL</button>
                </div>
              </div>
              <div className="mcp-form-row" data-align-row>
                {form.kind === "stdio" ? (
                  <>
                    <input className="dlg-input" value={form.command} onChange={(e) => setForm({ ...form, command: e.target.value })} placeholder="Command" aria-label="Command" disabled={!!job} />
                    <input className="dlg-input" value={form.args} onChange={(e) => setForm({ ...form, args: e.target.value })} placeholder="Arguments" aria-label="Arguments" disabled={!!job} />
                  </>
                ) : (
                  <input className="dlg-input" value={form.url} onChange={(e) => setForm({ ...form, url: e.target.value })} placeholder="https://" aria-label="Server URL" disabled={!!job} />
                )}
                <button type="submit" className="btn btn-primary" disabled={!!job || !form.name.trim() || (scope === "project" && !workspaceId) || (scope === "agent" && !agentWorkPath)}>Add</button>
              </div>
              <button
                type="button"
                className="btn btn-ghost mcp-more"
                aria-expanded={form.more}
                disabled={!!job}
                onClick={() => setForm({ ...form, more: !form.more, pairs: form.pairs.length ? form.pairs : [{ key: "", value: "" }] })}
              >{form.more ? "Less" : "More"}</button>
              {form.more ? (
                <div className="mcp-extra">
                  {form.kind === "url" ? (
                    <div className="mcp-form-row" data-align-row>
                      <div className="pkg-scope" role="radiogroup" aria-label="Sign-in">
                        <button type="button" role="radio" className="pkg-scope-btn" aria-checked={form.auth === ""} onClick={() => setForm({ ...form, auth: "", token: "" })}>None</button>
                        <button type="button" role="radio" className="pkg-scope-btn" aria-checked={form.auth === "oauth"} title="Opens the server's sign-in page" onClick={() => setForm({ ...form, auth: "oauth", token: "" })}>Sign in</button>
                        <button type="button" role="radio" className="pkg-scope-btn" aria-checked={form.auth === "bearer"} onClick={() => setForm({ ...form, auth: "bearer" })}>Token</button>
                      </div>
                      {form.auth === "bearer" ? (
                        <input
                          className="dlg-input"
                          type="password"
                          autoComplete="off"
                          value={form.token}
                          onChange={(e) => setForm({ ...form, token: e.target.value })}
                          placeholder="Token"
                          aria-label="Token"
                          disabled={!!job}
                        />
                      ) : null}
                    </div>
                  ) : null}
                  <PairList
                    kind={form.kind}
                    pairs={form.pairs}
                    disabled={!!job}
                    onChange={(pairs) => setForm({ ...form, pairs })}
                  />
                </div>
              ) : null}
              {formError ? <p className="form-error">{formError}</p> : null}
            </form>
          </section>
        </>
      )}

      {pickOpen ? (
        <UseFromDialog found={(data && data.found) || []} onClose={() => setPickOpen(false)} onUse={applyPicks} />
      ) : null}
      {job ? (
        <JobOverlay
          job={job}
          running={agentRunning}
          onClose={() => setJob(null)}
          onCancel={() => { signCtl.current.stop = true; }}
        />
      ) : null}
    </PageFrame>
  );
}

function UseFromDialog({ found, onClose, onUse }) {
  const [open, setOpen] = useState(() => {
    const o = {};
    (found || []).forEach((h) => { o[h.kind] = true; });
    return o;
  });
  const [on, setOn] = useState(() => {
    const o = {};
    (found || []).forEach((h) => {
      (h.servers || []).forEach((s) => { o[h.kind + ":" + s.name] = !!s.on; });
    });
    return o;
  });

  function key(kind, name) { return kind + ":" + name; }

  function appStats(h) {
    const names = (h.servers || []).map((s) => s.name);
    const n = names.filter((name) => on[key(h.kind, name)]).length;
    return { all: names.length > 0 && n === names.length, some: n > 0 && n < names.length, n };
  }

  function toggleApp(h, v) {
    setOn((cur) => {
      const next = { ...cur };
      (h.servers || []).forEach((s) => { next[key(h.kind, s.name)] = v; });
      return next;
    });
  }

  function submit() {
    const picks = (found || []).map((h) => ({
      kind: h.kind,
      servers: (h.servers || []).map((s) => s.name).filter((name) => on[key(h.kind, name)]),
    })).filter((p) => p.servers.length);
    onUse(picks);
  }

  return (
    <Dialog.Root open onOpenChange={(v) => { if (!v) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg dlg-use-from" onCloseAutoFocus={(e) => e.preventDefault()}>
          <Dialog.Title className="dlg-title">Use from other apps</Dialog.Title>
          <Dialog.Description className="dlg-body">Check the servers to use. Pi reads those apps.</Dialog.Description>
          <ul className="mcp-tree">
            {(found || []).map((h) => {
              const st = appStats(h);
              return (
                <li key={h.kind}>
                  <div className="mcp-tree-row">
                    <button
                      type="button"
                      className="mcp-tree-chev"
                      aria-expanded={!!open[h.kind]}
                      aria-label={open[h.kind] ? "Collapse" : "Expand"}
                      onClick={() => setOpen((o) => ({ ...o, [h.kind]: !o[h.kind] }))}
                    >{open[h.kind] ? "▾" : "▸"}</button>
                    <TreeCheck checked={st.all} some={st.some} onChange={(v) => toggleApp(h, v)} label={h.label} />
                    <span className="mcp-tree-label">{h.label}</span>
                  </div>
                  {open[h.kind] ? (
                    <ul className="mcp-tree-kids">
                      {(h.servers || []).length === 0 ? (
                        <li className="mcp-tree-row mcp-tree-empty">No servers</li>
                      ) : (h.servers || []).map((s) => (
                        <li key={s.name} className="mcp-tree-row">
                          <span className="mcp-tree-spc" aria-hidden="true" />
                          <label className="mcp-tree-item">
                            <input
                              type="checkbox"
                              checked={!!on[key(h.kind, s.name)]}
                              onChange={(e) => setOn((cur) => ({ ...cur, [key(h.kind, s.name)]: e.target.checked }))}
                            />
                            <span>{s.name}</span>
                          </label>
                        </li>
                      ))}
                    </ul>
                  ) : null}
                </li>
              );
            })}
          </ul>
          <div className="dlg-actions">
            <button type="button" className="btn btn-ghost" onClick={onClose}>Cancel</button>
            <button type="button" className="btn btn-primary" onClick={submit}>Use</button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

function TreeCheck({ checked, some, onChange, label }) {
  const ref = useRef(null);
  useEffect(() => {
    if (ref.current) ref.current.indeterminate = !!some && !checked;
  }, [some, checked]);
  return (
    <input
      ref={ref}
      type="checkbox"
      checked={!!checked}
      aria-label={label}
      onChange={(e) => onChange(e.target.checked)}
    />
  );
}

function hostLabel(id) {
  const names = {
    cursor: "Cursor",
    "claude-code": "Claude Code",
    "claude-desktop": "Claude Desktop",
    codex: "Codex",
    opencode: "OpenCode",
    windsurf: "Windsurf",
    vscode: "VS Code",
    grok: "Grok",
  };
  return names[id] || id;
}

function emptyForm() {
  return { name: "", kind: "url", command: "", args: "", url: "", auth: "", token: "", pairs: [], more: false };
}

function PairList({ kind, pairs, disabled, onChange }) {
  const rows = pairs.length ? pairs : [{ key: "", value: "" }];
  const keyLabel = kind === "stdio" ? "Variable" : "Header";
  function setRow(i, patch) {
    onChange(rows.map((row, idx) => (idx === i ? { ...row, ...patch } : row)));
  }
  function remove(i) {
    const next = rows.filter((_, idx) => idx !== i);
    onChange(next.length ? next : [{ key: "", value: "" }]);
  }
  return (
    <div className="mcp-pairs">
      {rows.map((row, i) => (
        <div key={i} className="mcp-form-row" data-align-row>
          <input className="dlg-input" value={row.key} onChange={(e) => setRow(i, { key: e.target.value })} placeholder={keyLabel} aria-label={keyLabel} disabled={disabled} />
          <input className="dlg-input" value={row.value} onChange={(e) => setRow(i, { value: e.target.value })} placeholder="Value" aria-label={keyLabel + " value"} disabled={disabled} />
          <button type="button" className="btn btn-ghost" disabled={disabled} aria-label={"Remove " + keyLabel.toLowerCase()} onClick={() => remove(i)}>×</button>
        </div>
      ))}
      <button type="button" className="btn btn-ghost" disabled={disabled} onClick={() => onChange(rows.concat({ key: "", value: "" }))}>Add {keyLabel.toLowerCase()}</button>
    </div>
  );
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

function rowLive(s) {
  if (s.live === "live" || s.live === "failed") return s.live;
  if (s.signedIn) return "signed";
  return s.live || "";
}

function liveLabel(v) {
  if (v === "live") return "Live";
  if (v === "failed") return "Failed";
  if (v === "signed") return "Signed in";
  if (v === "signin") return "Sign in";
  return "Idle";
}

function liveTitle(v) {
  if (v === "live") return "Connected";
  if (v === "failed") return "Last connect failed";
  if (v === "signed") return "This machine has a login";
  if (v === "signin") return "Needs sign-in";
  return "Not used yet";
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

function JobOverlay({ job, running, onClose, onCancel }) {
  const steps = [
    { id: "write", label: job.action === "signin" ? "Sign in to " + job.label : job.action === "import" ? "Use from other apps" : (job.action === "remove" ? "Remove " : job.action === "toggle" ? "Update " : "Save ") + job.label },
    { id: "reload", label: job.action === "signin" ? "Finish in the browser tab" : (running ? "Reload this agent" : "Applies on next start") },
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
            <button type="button" className="btn btn-primary" onClick={onClose}>Close</button>
          </>
        ) : job.action === "signin" ? (
          <div className="pkg-job-actions" data-align-row>
            <button type="button" className="btn btn-ghost" onClick={onCancel}>Cancel</button>
          </div>
        ) : (
          <p className="pkg-fine">Saving…</p>
        )}
      </div>
    </div>
  );
}
