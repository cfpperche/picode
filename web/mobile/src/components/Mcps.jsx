import { cliPackagesHash } from "@picode/shared/domain/cliPackages.js";
import { Fragment, useEffect, useRef, useState } from "react";
import * as Dialog from "./MobileSheet.jsx";
import * as Switch from "@radix-ui/react-switch";
import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import { api, humanizeError } from "@picode/shared/client/api.js";
import { feedConnected, subscribeFeed } from "@picode/shared/client/feed.js";
import { askConfirm } from "../lib/confirm.js";
import { mcpAddSchema, pairsToMap, parseForm } from "@picode/shared/contracts/schemas.js";
import { toast } from "../lib/toast.js";
import PageFrame from "./PageFrame.jsx";
import PiSpinner from "./PiSpinner.jsx";
import { connectorDriver, blockedLayers, connectorAddBody, connectorScopeNames, connectorDocsUrl } from "@picode/shared/domain/integrations.js";
import "../styles/integrations.css";

export default function Mcps({ hidden, cli = "pi", workspaceId, workspaceName, workspacePath, agentId, agentName, agentWorkPath, agentRunning, onReload, scope: scopeProp = "user", onScopeChange = () => {} }) {
  const [data, setData] = useState(null);
  const [loadError, setLoadError] = useState("");
  const scope = scopeProp;
  const setScope = onScopeChange;
  const [job, setJob] = useState(null);
  const [form, setForm] = useState(emptyForm());
  const [formError, setFormError] = useState("");
  const [customOpen, setCustomOpen] = useState(false);
  const [tab, setTab] = useState("installed"); // installed | marketplace
  const [q, setQ] = useState("");
  const [hits, setHits] = useState([]);
  const [searching, setSearching] = useState(true);
  const [catError, setCatError] = useState(false);
  const [justSigned, setJustSigned] = useState({});
  const signCtl = useRef({ id: "", stop: false });
  const dataRef = useRef(null);
  dataRef.current = data;
  const driver = connectorDriver(cli) || connectorDriver("pi");
  const cliScoped = cli !== "pi" && !!driver;

  function listURL() {
    const q = [];
    if (cliScoped) q.push("cli=" + encodeURIComponent(cli));
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
  }, [hidden, workspaceId, agentId, cli]);
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
    // CLI agents have no per-agent layer in phase 1; a stale agent-scope link
    // falls back to the machine instead of a 400 on the next action.
    if (cliScoped && scope === "agent") setScope("user");
  }, [workspaceId, agentWorkPath, scope, cliScoped]);

  const loading = !hidden && data === null && !loadError;
  const installed = !!(data && data.adapter && data.adapter.installed);
  const servers = (data && data.servers) || [];
  const blocked = blockedLayers(data);
  const canProject = !!workspaceId;
  const canAgent = !!agentWorkPath;
  // A catalog entry is "Added" when the selected layer already has it; another
  // layer's copy is not this one (docs/plans/connectors-ux.md).
  const configuredNames = connectorScopeNames(servers, scope);

  // Marketplace search (ADR-0157): one debounced gallery fetch per query,
  // seed hits first. A refetch keeps the last results up; skeletons cover
  // the first load, the error card covers a failed one.
  useEffect(() => {
    if (hidden || !installed || tab !== "marketplace") return;
    const t = setTimeout(pullGallery, q ? 280 : 0);
    return () => clearTimeout(t);
  }, [hidden, installed, tab, q]);

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
      ...(cliScoped ? { cli } : {}),
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
    if (auth === "oauth" && !cliScoped) await signIn({ name, auth: "oauth", disabled: false }, { quiet: true });
  }

  async function pullGallery() {
    setSearching(true);
    try {
      const page = await api("/api/connectors/gallery?q=" + encodeURIComponent(q.trim()));
      setHits(page.hits || []);
      setCatError(false);
    } catch {
      setHits([]);
      setCatError(true);
    } finally {
      setSearching(false);
    }
  }

  function openCustom() {
    setForm(emptyForm());
    setFormError("");
    setCustomOpen(true);
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

  function canSignIn(s) {
    if (cliScoped) return false;
    if (!s || s.disabled) return false;
    if (s.live === "live") return false;
    if (s.live === "signin") return true;
    return s.auth === "oauth" && !s.signedIn && !justSigned[s.name];
  }

  // CLI agents never sign in through PiCode: the pane shows the vendor's own
  // path instead of a failing call (ADR-0150) — a copyable command (a
  // terminal one by default, or the vendor TUI) beside the controls, or a
  // plain sentence on its own line under the row when the vendor signs in
  // outside any command line (a sentence inline would squeeze the target
  // and unbalance the action row).
  function needsVendorSignIn(s) {
    return cliScoped && !s.disabled && s.transport === "url" && driver.auth.includes("oauth");
  }

  function vendorSignIn(s) {
    const si = driver.signIn || {};
    if (si.text) return { text: si.text };
    const command = String(si.command || cli + " mcp login " + s.name).replaceAll("{name}", s.name);
    return { command, where: si.where || "a terminal" };
  }

  function copyVendorCmd(cmd) {
    navigator.clipboard.writeText(cmd).then(() => toast.ok("Command copied.")).catch(() => toast.error("Clipboard blocked — copy it by hand."));
  }

  function canSignOut(s) {
    if (cliScoped) return false;
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
      message: "Delete this server from " + (s.path || driver.name + "'s configuration") + ".",
      confirmLabel: "Remove",
      danger: true,
    });
    if (!ok) return;
    const q = new URLSearchParams({
      name: s.name,
      scope: writeScope(s, scope),
    });
    if (cliScoped) q.set("cli", cli);
    if (workspaceId) q.set("workspace", workspaceId);
    if (agentId) q.set("agent", agentId);
    await runJob("remove", s.name, () => api("/api/mcp?" + q.toString(), { method: "DELETE" }));
  }

  // A blocked config file is opened in the host file manager; the server
  // resolves the path itself, so the client never supplies one.
  async function revealBlocked(b) {
    try {
      await api("/api/mcp/reveal", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ cli, scope: b.scope }),
      });
    } catch (err) {
      toast.error(humanizeError(err.message || String(err)));
    }
  }

  return (
    <PageFrame id="connectors-view" title="Connectors" embedded hidden={hidden}>
      <div className="connectors-head">
        <p className="integrations-intro">Connect services your agents can use.</p>
      </div>
      {blocked.map((b) => (
        <div key={b.scope + ":" + b.path} className="mcp-blocked" role="status">
          <span className="mcp-blocked-text"><code title={b.path || undefined}>{b.file}</code> {b.error}.</span>
          <span className="mcp-row-actions" data-align-row>
            {cliScoped ? <button type="button" className="btn btn-ghost btn-sm" disabled={!!job} onClick={() => revealBlocked(b)}>Open</button> : null}
            <button type="button" className="btn btn-ghost btn-sm" onClick={() => load()}>Retry</button>
          </span>
        </div>
      ))}
      {loading ? (
        <div className="mcp-skel" aria-hidden="true">
          <div className="skel-line w-70" />
          <div className="skel-line w-40" />
        </div>
      ) : loadError && !data ? (
        <div className="mcp-empty">
          <p>Couldn't load connectors.</p>
          <button type="button" className="btn btn-primary" onClick={() => { setLoadError(""); load(); }}>Retry</button>
        </div>
      ) : !installed ? (
        cliScoped ? (
          <div className="mcp-empty">
            <p>{driver.name} is not installed on this machine.</p>
            <button type="button" className="btn btn-primary" onClick={() => load()}>Check again</button>
          </div>
        ) : (
          <div className="mcp-empty">
            <p>Install the MCP adapter to connect services.</p>
            <a className="btn btn-primary" href={cliPackagesHash("pi", { workspaceId, agentId })}>Open packages</a>
          </div>
        )
      ) : (
        <>
          <div className="pkg-tabs" role="tablist" aria-label="Connectors">
            <button type="button" role="tab" className="pkg-tab" aria-selected={tab === "installed"} onClick={() => setTab("installed")}>
              Installed{servers.length ? <span className="pkg-tab-count">{servers.length}</span> : null}
            </button>
            <button type="button" role="tab" className="pkg-tab" aria-selected={tab === "marketplace"} onClick={() => setTab("marketplace")}>Marketplace</button>
          </div>
          {tab === "installed" ? (
            <>
              {!!data?.connectorPackages?.length && <section className="pkg-installed">
                <h3>Connector packages</h3>
                <ul className="mcp-list">{data.connectorPackages.map(p => <li key={p.scope + p.source} className="mcp-row integration-package"><div className="mcp-row-main"><strong>{p.name}</strong><span className="pkg-fine">Installed · {p.scope === "user" ? "Global" : workspaceName || "This workspace"}</span></div><a className="btn btn-ghost" href={cliPackagesHash("pi", { workspaceId, agentId })}>Manage package</a></li>)}</ul>
                <p className="pkg-fine">Package tools load through the MCP adapter when an agent starts.</p>
              </section>}
              <section className="pkg-installed">
                <h3>Configured services{servers.length ? <span className="mcp-count">{servers.length}</span> : null}</h3>
                {!cliScoped && !agentRunning && servers.length ? <p className="pkg-fine">Live status comes from a running agent.</p> : null}
                {servers.length ? (
                  <ul className="mcp-list">
                    {servers.map((s) => {
                      // Pi rows stream live state from the adapter while an
                      // agent runs; live-capable CLI agents probe their vendor
                      // CLI on the server, agent-independent (ADR-0150 d4).
                      const word = agentRunning || (cliScoped && driver.status === "live") ? rowLive(s) : "";
                      const state = word === "live" || word === "failed" ? word : "";
                      const menu = canSignOut(s) || s.owned;
                      const vendor = needsVendorSignIn(s) ? vendorSignIn(s) : null;
                      const note = vendor && !vendor.command ? vendor.text : "";
                      return (
                      <li key={s.layer + ":" + s.name} className={"mcp-row" + (s.disabled ? " off" : "") + (note ? " has-note" : "")}>
                        <div className="mcp-row-main">
                          {state ? (
                            <span className={"mcp-live " + state} title={liveTitle(state)}>{liveLabel(state)}</span>
                          ) : null}
                          <strong className="mcp-name">{s.name}</strong>
                          <span className="pkg-scope-tag" title={s.path || undefined}>{scopeLabel(s, workspaceName, agentName)}</span>
                          <code className="mcp-target" title={targetOf(s)}>{targetOf(s)}</code>
                        </div>
                        <div className="mcp-row-actions" data-align-row>
                          {vendor && vendor.command ? (
                            <span className="mcp-login">
                              <code title={"Run this in " + vendor.where + " to sign in to " + s.name}>{vendor.command}</code>
                              <button type="button" className="btn btn-ghost btn-sm" disabled={!!job} aria-label={"Copy the sign-in command for " + s.name} onClick={() => copyVendorCmd(vendor.command)}>Copy</button>
                            </span>
                          ) : !vendor && canSignIn(s) ? (
                            <button type="button" className="btn btn-ghost" disabled={!!job} onClick={() => signIn(s)}>Sign in</button>
                          ) : null}
                          {driver.toggle !== "none" ? (
                            <span className="mcp-switch">
                              <Switch.Root
                                className="rx-switch"
                                checked={!s.disabled}
                                disabled={!!job}
                                onCheckedChange={() => toggle(s)}
                                aria-label={(s.disabled ? "Enable " : "Disable ") + s.name}
                              ><Switch.Thumb className="rx-switch-thumb" /></Switch.Root>
                            </span>
                          ) : null}
                          {menu ? (
                            <DropdownMenu.Root>
                              <DropdownMenu.Trigger asChild>
                                <button type="button" className="btn btn-ghost btn-sm mcp-more" aria-label={"More actions for " + s.name} disabled={!!job}>•••</button>
                              </DropdownMenu.Trigger>
                              <DropdownMenu.Portal>
                                <DropdownMenu.Content className="um-popover" align="end" sideOffset={5} collisionPadding={12}>
                                  {canSignOut(s) ? <DropdownMenu.Item className="um-item" onSelect={() => signOut(s)}>Sign out</DropdownMenu.Item> : null}
                                  {s.owned ? <>{canSignOut(s) ? <DropdownMenu.Separator className="um-divider" /> : null}<DropdownMenu.Item className="um-item cli-danger-item" onSelect={() => remove(s)}>Remove</DropdownMenu.Item></> : null}
                                </DropdownMenu.Content>
                              </DropdownMenu.Portal>
                            </DropdownMenu.Root>
                          ) : null}
                        </div>
                        {note ? <p className="mcp-login-note">{note}</p> : null}
                      </li>
                      );
                    })}
                  </ul>
                ) : (
                  <p className="pkg-fine">No connectors yet.</p>
                )}
              </section>
            </>
          ) : (
            // Marketplace (ADR-0157): the curated catalog is the add surface —
            // search the gallery, save to the selected layer, custom form stays
            // as the quiet escape hatch.
            <section className="connector-market" role="tabpanel" aria-label="Connector marketplace">
              <div className="connector-pick" data-align-row data-align-wrap>
                <input
                  className="pkg-search"
                  value={q}
                  onChange={(e) => setQ(e.target.value)}
                  placeholder="Search connectors…"
                  aria-label="Search connectors"
                  disabled={!!job}
                  autoComplete="off"
                />
                {canProject || (canAgent && !cliScoped) ? (
                  <div className="connector-scope">
                    <span className="mcp-group-label">Save to</span>
                    <div className="pkg-scope" role="radiogroup" aria-label="Save to">
                      <button type="button" role="radio" className="pkg-scope-btn" aria-checked={scope === "user"} onClick={() => setScope("user")}>Global</button>
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
                      {canAgent && !cliScoped ? (
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
                  </div>
                ) : null}
                <button type="button" className="btn btn-ghost" disabled={!!job} onClick={openCustom}>Custom server…</button>
              </div>
              {catError ? (
                <div className="mcp-empty">
                  <p>Couldn't load the catalog.</p>
                  <button type="button" className="btn btn-primary" disabled={!!job} onClick={() => pullGallery()}>Retry</button>
                </div>
              ) : searching && !hits.length ? (
                <ul className="pkg-grid" aria-busy="true">
                  {Array.from({ length: 6 }, (_, i) => (
                    <li key={"skel-" + i} className="pkg-card pkg-skel" aria-hidden="true">
                      <div className="pkg-preview"><span className="conn-tile" /></div>
                      <div className="pkg-card-body">
                        <div className="skel-line w-50" />
                        <div className="skel-line w-90" />
                        <div className="skel-line w-70" />
                        <div className="skel-line w-40" />
                        <div className="skel-line w-80" />
                      </div>
                    </li>
                  ))}
                </ul>
              ) : hits.length ? (
                <>
                  {[
                    // The seed rides unlabeled at the top; only the synced
                    // registry rows take a group label — the seed mixes
                    // PiCode tools and hand-picked presets, so "PiCode"
                    // mislabeled more than it explained (owner, 2026-09-19).
                    ["", hits.filter((h) => h.source !== "registry")],
                    ["Catalog", hits.filter((h) => h.source === "registry")],
                  ].map(([group, rows]) => rows.length ? (
                    <Fragment key={group || "seed"}>
                      {group ? <p className="mcp-group-label connector-group">{group}</p> : null}
                      <ul className="pkg-grid" aria-busy={searching}>
                        {rows.map((h) => {
                          const on = configuredNames.has(h.id);
                          const docs = connectorDocsUrl(h);
                          return (
                            <li key={h.id} className="pkg-card">
                              <div className="pkg-preview" aria-hidden="true">
                                <span className="conn-tile">{(h.name || h.id || "?").trim().charAt(0).toUpperCase()}</span>
                              </div>
                              <div className="pkg-card-body">
                                <div className="pkg-card-head">
                                  <span className="pkg-card-name" title={h.name}>{h.name}</span>
                                  <span className="pkg-type">{h.kind === "stdio" ? "Local" : "Remote"}</span>
                                  {h.auth === "oauth" ? <span className="pkg-type">Sign-in required</span> : null}
                                </div>
                                <p className="pkg-card-desc">{h.summary || " "}</p>
                                <div className="pkg-card-foot">
                                  {docs ? (
                                    <a className="btn btn-ghost btn-sm" href={docs} target="_blank" rel="noopener noreferrer" aria-label={"Docs for " + h.name}>Docs</a>
                                  ) : null}
                                  <span className="pkg-foot-spacer" />
                                  <button
                                    type="button"
                                    className="btn btn-primary btn-sm"
                                    disabled={!!job || on}
                                    onClick={() => addServer(connectorAddBody(h))}
                                  >{on ? "Added" : "Add"}</button>
                                </div>
                              </div>
                            </li>
                          );
                        })}
                      </ul>
                    </Fragment>
                  ) : null)}
                </>
              ) : (
                <div className="mcp-empty">
                  <p>No connectors match — try another word, or use Custom server… above.</p>
                </div>
              )}
            </section>
          )}
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
  return "global";
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
    { id: "write", label: job.action === "signin" ? "Sign in to " + job.label : job.action === "signout" ? "Sign out " + job.label : (job.action === "remove" ? "Remove " : job.action === "toggle" ? "Update " : "Save ") + job.label },
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
