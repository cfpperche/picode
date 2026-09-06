import { useEffect, useRef, useState } from "react";
import * as Dialog from "./ResponsiveDialog.jsx";
import { api, humanizeError } from "@picode/shared/client/api.js";
import { feedConnected, subscribeFeed } from "@picode/shared/client/feed.js";
import { askConfirm } from "../lib/confirm.js";
import { mcpAddSchema, pairsToMap, parseForm } from "@picode/shared/contracts/schemas.js";
import { toast } from "../lib/toast.js";
import { paneContext } from "@picode/shared/domain/tree.js";
import PageFrame from "./PageFrame.jsx";
import PiSpinner from "./PiSpinner.jsx";
import { readConnectorDefinition, destinationLabel, connectorTabs } from "@picode/shared/domain/integrations.js";

export default function Mcps({ hidden, embedded, workspaceId, workspaceName, workspacePath, agentId, agentName, agentWorkPath, agentRunning, onReload }) {
  const [data, setData] = useState(null);
  const [loadError, setLoadError] = useState("");
  const [scope, setScope] = useState("user");
  const [job, setJob] = useState(null);
  const [form, setForm] = useState(emptyForm());
  const [formError, setFormError] = useState("");
  const [catalogTab, setCatalogTab] = useState("catalog");
  const [customOpen, setCustomOpen] = useState(false);
  const [hostPick, setHostPick] = useState(null);
  const [justSigned, setJustSigned] = useState({});
  const signCtl = useRef({ id: "", stop: false });
  const dataRef = useRef(null);
  dataRef.current = data;

  function listURL() {
    const q = [];
    if (workspaceId) q.push("workspace=" + encodeURIComponent(workspaceId));
    if (agentId) q.push("agent=" + encodeURIComponent(agentId));
    return "/api/mcp" + (q.length ? "?" + q.join("&") : "");
  }

  async function load() {
    try {
      setData(await api(listURL()));
      setLoadError("");
    } catch (err) {
      if (dataRef.current) return;
      setLoadError(humanizeError(err.message || String(err)));
    }
  }

  useEffect(() => {
    if (hidden) return;
    setData(null);
    setLoadError("");
    load();
  }, [hidden, workspaceId, agentId]);
  useEffect(() => {
    if (hidden) return;
    load();
  }, [agentRunning]);
  useEffect(() => {
    if (hidden) return;
    // Change feed (ADR-0048): pi's live MCP status streams as ephemeral
    // mcp.updated events — one fleet-wide watcher instead of one poll per
    // open panel. The interval below is only the feed-down fallback, and
    // the panel reconciles once when the feed (re)opens.
    const unsub = subscribeFeed((ev) => {
      if (ev.type === "feed.open" || ev.type === "feed.reset") load();
      else if (ev.type === "mcp.config") load();
      else if (ev.type === "mcp.updated" && ev.data && ev.data.agentId === agentId) load();
    });
    const t = agentRunning ? setInterval(() => { if (!feedConnected()) load(); }, 2500) : null;
    return () => { unsub(); clearInterval(t); };
  }, [hidden, agentRunning, workspaceId, agentId]);
  useEffect(() => {
    if (!workspaceId && scope === "project") setScope("user");
    if (!agentWorkPath && scope === "agent") setScope("user");
  }, [workspaceId, agentWorkPath, scope]);

  const loading = !hidden && data === null && !loadError;
  const installed = !!(data && data.adapter && data.adapter.installed);
  const servers = (data && data.servers) || [];
  const presets = (data && data.presets) || [];
  const tabs = connectorTabs(data && data.found);
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

  function whereLabel() {
    return scope === "user" ? "this machine" : scope === "project" ? (workspaceName || "this workspace") : (agentName || "this agent");
  }

  async function importDefinition(file) {
    if (!file || job) return;
    try {
      if (file.size > 65536) throw new Error("Choose a connector file smaller than 64 KB.");
      const entry = readConnectorDefinition(await file.text());
      const where = whereLabel();
      const message = entry.command
        ? "Runs " + [entry.command, ...(entry.args || [])].map(v => JSON.stringify(v)).join(" ") + " with your system permissions. Add to " + where + " only if you trust its source."
        : "Allows agents in " + where + " to use tools from " + destinationLabel(entry.url) + ". Only connect services you trust.";
      if (!await askConfirm({ title: "Add " + entry.name + "?", message, confirmLabel: "Add connector" })) return;
      setFormError("");
      await addServer(entry);
    } catch (err) { setFormError(err.message); }
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
          env: entry.env, headers: entry.headers, bearerToken: entry.bearerToken,
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
      setCustomOpen(false);
    }
    if (auth === "oauth") await signIn({ name, auth: "oauth", disabled: false }, { quiet: true });
  }

  async function toggle(s) {
    if (!installed) return;
    const turningOn = !!s.disabled;
    // Servers PiCode does not own (host imports and shared files) toggle a
    // user-layer override stub so OFF/ON is scope-proof: both directions
    // always land in the same file the adapter reads last.
    const toggleScope = s.owned ? writeScope(s, scope) : "user";
    const next = await runJob("toggle", s.name, () => api("/api/mcp", {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body({ name: s.name, scope: toggleScope, disabled: !s.disabled })),
    }));
    if (!next) return;
    if (turningOn) await signIn({ ...s, disabled: false }, { quiet: true });
  }

  async function applyPicks(picks) {
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
    return s.auth === "oauth" && !s.signedIn && !justSigned[s.name];
  }

  function canSignOut(s) {
    if (!s || s.disabled) return false;
    return !!(s.signedIn || justSigned[s.name]);
  }

  function markSigned(name) {
    setJustSigned((m) => ({ ...m, [name]: true }));
    void (async () => {
      for (let i = 0; i < 10; i++) {
        await new Promise((r) => setTimeout(r, 400));
        await load();
      }
    })();
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
        markSigned(s.name);
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
          if (!loopbackRedirect(st.url)) {
            throw new Error("This server cannot Sign in from PiCode. It sends you back to its own site.");
          }
          opened = true;
          window.open(st.url, "picode-mcp-auth");
        }
        if (st && st.ok) {
          setJob(null);
          toast.ok("Signed in to " + s.name + ".");
          markSigned(s.name);
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

  async function signOut(s) {
    if (!canSignOut(s)) return;
    const ok = await askConfirm({
      title: "Sign out " + s.name,
      message: "Forget this login on this machine. Every agent will need to Sign in again.",
      confirmLabel: "Sign out",
    });
    if (!ok) return;
    await runJob("signout", s.name, () => api("/api/mcp/auth/logout", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name: s.name, agentId: agentId || undefined, workspaceId: workspaceId || undefined }),
    }));
    setJustSigned((m) => {
      const next = { ...m };
      delete next[s.name];
      return next;
    });
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
    <PageFrame id={embedded ? "connectors-view" : "mcps-view"} title={embedded ? "Connectors" : "MCPs"} embedded={embedded} context={paneContext(agentName, workspaceName)} hidden={hidden}>
      {embedded && <p className="integrations-intro">Connect services your agents can use.</p>}
      {loading ? (
        <div className="mcp-skel" aria-hidden="true">
          <div className="skel-line w-70" />
          <div className="skel-line w-40" />
        </div>
      ) : loadError && !data ? (
        <div className="mcp-empty">
          <p>Couldn't load {embedded ? "connectors" : "MCPs"}.</p>
          <button type="button" className="btn btn-primary" onClick={() => { setLoadError(""); load(); }}>Retry</button>
        </div>
      ) : !installed ? (
        <div className="mcp-empty">
          <p>Install the MCP adapter to connect services.</p>
          <a className="btn btn-primary" href="#/packages">Open packages</a>
        </div>
      ) : (
        <>
          {embedded && !!data?.connectorPackages?.length && <section className="pkg-installed">
            <h3>Connector packages</h3>
            <ul className="mcp-list">{data.connectorPackages.map(p => <li key={p.scope + p.source} className="mcp-row integration-package"><div className="mcp-row-main"><strong>{p.name}</strong><span className="pkg-fine">Installed · {p.scope === "user" ? "This machine" : "This workspace"}</span></div><a className="btn btn-ghost" href="#/packages">Manage package</a></li>)}</ul>
            <p className="pkg-fine">Package tools load through the MCP adapter when an agent starts.</p>
          </section>}
          <section className="pkg-installed">
            <h3>{embedded ? "Configured services" : "Servers"}</h3>
            {servers.length ? (
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
                      {canSignOut(s) ? (
                        <button type="button" className="btn btn-ghost" disabled={!!job} onClick={() => signOut(s)}>Sign out</button>
                      ) : canSignIn(s) ? (
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
            ) : (
              <p className="pkg-fine">No {embedded ? "connectors" : "servers"} yet. Choose a service below to add one.</p>
            )}
          </section>

          <section className="mcp-add">
            <h3>{embedded ? "Add connector" : "Add"}</h3>
            {embedded && <label className="integration-import">Import a connector definition<input type="file" accept=".json,application/json" disabled={!!job} onChange={e => { const file = e.target.files?.[0]; e.target.value = ""; importDefinition(file); }} /></label>}
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
            <div className="catalog-tabs" role="tablist" aria-label="Connector catalogs">
              {tabs.map((t) => (
                <button
                  key={t.id}
                  type="button"
                  role="tab"
                  aria-selected={catalogTab === t.id}
                  className="pkg-scope-btn"
                  disabled={!!job}
                  onClick={() => setCatalogTab(t.id)}
                >{t.label}</button>
              ))}
            </div>
            {tabs.map((t) => {
              if (t.id !== catalogTab) return null;
              return (
                <div key={t.id} className={embedded ? "integration-catalog" : "mcp-presets"} data-align-row={embedded ? undefined : true} role="tabpanel" aria-label={t.fixed ? "Cataloged services" : t.label + " servers"}>
                  {t.fixed ? (
                    <>
                      <button
                        type="button"
                        className={(embedded ? "integration-preset" : "pkg-scope-btn") + " integration-custom"}
                        disabled={!!job}
                        onClick={() => { setForm(emptyForm()); setFormError(""); setCustomOpen(true); }}
                      ><strong>Custom</strong><span>Configure a server by URL or local command.</span></button>
                      {presets.map((p) => (
                        <button
                          key={p.id}
                          type="button"
                          className={embedded ? "integration-preset" : "pkg-scope-btn"}
                          disabled={!!job}
                          title={p.summary}
                          onClick={() => addServer({ name: p.id, ...p.entry })}
                        >{embedded ? <><strong>{p.name}</strong><span>{p.summary}</span></> : p.name}</button>
                      ))}
                    </>
                  ) : t.servers.map((s) => (
                    <button
                      key={s.name}
                      type="button"
                      className={(embedded ? "integration-preset" : "pkg-scope-btn") + (s.on && embedded ? " integration-added" : "")}
                      disabled={!!job || s.on}
                      title={s.on ? "Already added from " + t.label : "Import from " + t.label}
                      onClick={() => setHostPick({ kind: t.id, label: t.label, server: s.name })}
                    >{embedded ? <><strong>{s.name}</strong><span>{s.on ? "Added from " + t.label : "Found in " + t.label}</span></> : (s.on ? s.name + " ✓" : s.name)}</button>
                  ))}
                </div>
              );
            })}
          </section>
        </>
      )}

      {customOpen ? (
        <Dialog.Root open onOpenChange={(v) => { if (!v) setCustomOpen(false); }}>
          <Dialog.Portal>
            <Dialog.Overlay className="dlg-overlay" />
            <Dialog.Content className="dlg integration-dialog" onCloseAutoFocus={(e) => e.preventDefault()}>
              <Dialog.Title className="dlg-title">Add custom connector</Dialog.Title>
              <Dialog.Description className="dlg-body">Name the server and choose how PiCode reaches it.</Dialog.Description>
              <form className="mcp-form integration-form" noValidate onSubmit={(e) => { e.preventDefault(); addServer({}); }}>
                <label>Server name
                  <input className="dlg-input" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="my-connector" aria-label="Server name" disabled={!!job} autoComplete="off" />
                </label>
                <div className="mcp-field-group">
                  <span className="mcp-group-label">How PiCode reaches it</span>
                  <div className="mcp-form-row" data-align-row>
                    <div className="pkg-scope" role="radiogroup" aria-label="Transport">
                      <button type="button" role="radio" className="pkg-scope-btn" aria-checked={form.kind === "stdio"} onClick={() => setForm({ ...form, kind: "stdio", auth: "", token: "" })}>Command</button>
                      <button type="button" role="radio" className="pkg-scope-btn" aria-checked={form.kind === "url"} onClick={() => setForm({ ...form, kind: "url" })}>URL</button>
                    </div>
                    {form.kind === "stdio" ? (
                      <>
                        <input className="dlg-input" value={form.command} onChange={(e) => setForm({ ...form, command: e.target.value })} placeholder="Command" aria-label="Command" disabled={!!job} autoComplete="off" />
                        <input className="dlg-input" value={form.args} onChange={(e) => setForm({ ...form, args: e.target.value })} placeholder="Arguments" aria-label="Arguments" disabled={!!job} autoComplete="off" />
                      </>
                    ) : (
                      <input className="dlg-input" value={form.url} onChange={(e) => setForm({ ...form, url: e.target.value })} placeholder="https://" aria-label="Server URL" disabled={!!job} autoComplete="off" />
                    )}
                  </div>
                </div>
                {form.kind === "url" ? (
                  <>
                    <div className="mcp-field-group">
                      <span className="mcp-group-label">Sign-in</span>
                      <div className="pkg-scope" role="radiogroup" aria-label="Sign-in">
                        <button type="button" role="radio" className="pkg-scope-btn" aria-checked={form.auth === ""} onClick={() => setForm({ ...form, auth: "", token: "" })}>None</button>
                        <button type="button" role="radio" className="pkg-scope-btn" aria-checked={form.auth === "oauth"} onClick={() => setForm({ ...form, auth: "oauth", token: "" })}>Sign in</button>
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
                      {form.auth === "oauth" ? <p className="pkg-fine">The agent opens the server's sign-in page on first use.</p> : null}
                    </div>
                    <div className="mcp-field-group">
                      <span className="mcp-group-label">Headers</span>
                      <PairList kind="url" pairs={form.pairs} disabled={!!job} onChange={(pairs) => setForm({ ...form, pairs })} />
                    </div>
                  </>
                ) : (
                  <div className="mcp-field-group">
                    <span className="mcp-group-label">Environment variables</span>
                    <PairList kind="stdio" pairs={form.pairs} disabled={!!job} onChange={(pairs) => setForm({ ...form, pairs })} />
                  </div>
                )}
                {formError ? <p className="form-error" role="alert">{formError}</p> : null}
                <div className="dlg-actions" data-align-row>
                  <button type="button" className="btn btn-ghost" disabled={!!job} onClick={() => setCustomOpen(false)}>Cancel</button>
                  <button type="submit" className="btn btn-primary" disabled={!!job || !form.name.trim() || (scope === "project" && !workspaceId) || (scope === "agent" && !agentWorkPath)}>Add connector</button>
                </div>
              </form>
            </Dialog.Content>
          </Dialog.Portal>
        </Dialog.Root>
      ) : null}

      {hostPick ? (
        <Dialog.Root open onOpenChange={(v) => { if (!v) setHostPick(null); }}>
          <Dialog.Portal>
            <Dialog.Overlay className="dlg-overlay" />
            <Dialog.Content className="dlg integration-dialog" onCloseAutoFocus={(e) => e.preventDefault()}>
              <Dialog.Title className="dlg-title">Add {hostPick.server} from {hostPick.label}?</Dialog.Title>
              <Dialog.Description className="dlg-body">Pi reads those apps. The server lands in {whereLabel()}.</Dialog.Description>
              <div className="dlg-actions">
                <button type="button" className="btn btn-ghost" disabled={!!job} onClick={() => setHostPick(null)}>Cancel</button>
                <button
                  type="button"
                  className="btn btn-primary"
                  disabled={!!job}
                  onClick={() => { const hp = hostPick; setHostPick(null); applyPicks([{ kind: hp.kind, servers: [hp.server] }]); }}
                >Add connector</button>
              </div>
            </Dialog.Content>
          </Dialog.Portal>
        </Dialog.Root>
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

function emptyForm() {
  return { name: "", kind: "url", command: "", args: "", url: "", auth: "", token: "", pairs: [] };
}

function PairList({ kind, pairs, disabled, onChange }) {
  const rows = pairs;
  const keyLabel = kind === "stdio" ? "Variable" : "Header";
  function setRow(i, patch) {
    onChange(rows.map((row, idx) => (idx === i ? { ...row, ...patch } : row)));
  }
  function remove(i) {
    const next = rows.filter((_, idx) => idx !== i);
    onChange(next);
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

function loopbackRedirect(authUrl) {
  try {
    const redir = new URL(authUrl).searchParams.get("redirect_uri");
    if (!redir) return true;
    const h = new URL(redir).hostname.toLowerCase();
    return h === "localhost" || h === "127.0.0.1" || h === "[::1]" || h === "::1";
  } catch {
    return false;
  }
}

function rowLive(s) {
  if (s.live === "live" || s.live === "failed") return s.live;
  return s.live || "";
}

function liveLabel(v) {
  if (v === "live") return "Live";
  if (v === "failed") return "Failed";
  if (v === "signin") return "Sign in";
  return "Idle";
}

function liveTitle(v) {
  if (v === "live") return "Connected";
  if (v === "failed") return "Last connect failed";
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
    { id: "write", label: job.action === "signin" ? "Sign in to " + job.label : job.action === "signout" ? "Sign out " + job.label : job.action === "import" ? "Import servers" : (job.action === "remove" ? "Remove " : job.action === "toggle" ? "Update " : "Save ") + job.label },
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
