import { useCallback, useEffect, useRef, useState } from "react";
import * as Switch from "@radix-ui/react-switch";
import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { askConfirm } from "../lib/confirm.js";
import { toast, toastError } from "../lib/toast.js";
import { cliLaunchSchema, cliTerminalSchema, parseForm } from "@picode/shared/contracts/schemas.js";
import { cliLocation, cliPaneHash, cliPaneSetupContext, launchDraft, launchConfig, editLaunchOverrides, resolveLaunch, cliTerminals, terminalLaunchCLI, profileOverrides, cliWorkspaceList } from "@picode/shared/domain/cliLaunch.js";
import { loadPiPackagesContext } from "@picode/shared/domain/cliPackages.js";
import { displayAgentName } from "@picode/shared/domain/tree.js";
import { terminalCli, terminalStatusLabel, terminalStatus } from "@picode/shared/domain/terminalCli.js";
import { termHash } from "../lib/routes.js";
import AgentClisFrame from "./AgentClisFrame.jsx";
import CliTabs from "./CliTabs.jsx";
import CliSettings from "./CliSettings.jsx";
import PeerMessages from "./PeerMessages.jsx";
import CliProviders from "./CliProviders.jsx";
import CliPackages from "./CliPackages.jsx";
import { supportsCliConnectors, cliConnectorsHash } from "@picode/shared/domain/integrations.js";
import Mcps from "./Mcps.jsx";
import SessionsView from "./SessionsView.jsx";
import CliPaneTabs, { cliSetupHref } from "./CliPaneTabs.jsx";
import CliCombo from "./CliCombo.jsx";
import TerminalCliBadge from "./TerminalCliBadge.jsx";
import { IconChevronRight } from "./Icons.jsx";
import { termRowMenu } from "../lib/termRowMenu.js";
import { CLIDefaults, LaunchFields, LaunchPreview, confirmDiscard, useLaunchGuard } from "./CliLaunchSettings.jsx";
import { CLIProfiles, CLIProfileEditor } from "./CliProfiles.jsx";

const json = (method, body) => ({ method, headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
const navigate = (path) => { location.hash = "#/clis" + path; };

function Notice({ children, action, onAction, danger = false }) {
  return <div className={"cli-notice" + (danger ? " is-error" : "")} role={danger ? "alert" : "status"}><span>{children}</span>{action ? <button type="button" className="btn btn-ghost btn-sm" onClick={onAction}>{action}</button> : null}</div>;
}

export default function AgentClis({ hidden = false, catalog, onCatalogChange, legacyAgentId = "", legacyPackageContext = {}, legacyContextReady = true, packageUpdates = [], onPackageUpdates, onAgentConfig, onOpenAgent = () => {}, onCompactAgent = () => {}, onRenameTerm, onReloadAgent }) {
  const [hash, setHash] = useState(location.hash);
  const route = cliLocation(hash, { packageContext: legacyPackageContext, agentId: legacyAgentId });
  const setupCtx = cliPaneSetupContext(route, { workspaceId: legacyPackageContext.workspaceId || "", agentId: legacyPackageContext.agentId || legacyAgentId || "" });
  const [data, setData] = useState(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState("");
  const [editRequested, setEditRequested] = useState({});
  const [launchEditing, setLaunchEditing] = useState(false);
  const active = useRef(!hidden);
  const request = useRef(0);
  active.current = !hidden;
  const refresh = useCallback(async () => {
    const seq = ++request.current;
    try {
      const [catalog, terms, work, presets, jobs] = await Promise.all([api("/api/clis"), api("/api/terminals"), api("/api/workspaces"), api("/api/clis/profiles"), api("/api/cli-jobs").catch(() => ({ jobs: [] }))]);
      if (seq !== request.current || !active.current) return;
      setData({ ...catalog, terminals: terms.terminals || [], workspaces: cliWorkspaceList(work), profiles: presets.profiles || [], jobs: jobs.jobs || [] }); setError("");
    } catch (e) { if (seq === request.current && active.current) setError(e.message); }
  }, []);
  useEffect(() => {
    const update = () => setHash(location.hash);
    window.addEventListener("hashchange", update);
    return () => window.removeEventListener("hashchange", update);
  }, []);
  useEffect(() => {
    if (hidden || route.view === "messages") return;
    if (hash === "#/preferences/status") location.replace("#/clis");
    if (route.adoptPane && !legacyContextReady) return;
    if (route.redirect && route.redirect !== hash) location.replace(route.redirect);
    if (!legacyContextReady || route.invalid) return;
    const paneName = route.pane || "launch";
    if (!["packages", "connectors", "settings"].includes(paneName)) return;
    if (route.workspaceId || route.agentId) return;
    if (!setupCtx.workspaceId && !setupCtx.agentId) return;
    const next = cliSetupHref(route.id || "pi", paneName, setupCtx, route.workspace || "");
    if (next && next !== hash) location.replace(next);
  }, [hidden, hash, route.view, route.redirect, route.adoptPane, route.invalid, route.pane, route.id, route.workspace, route.workspaceId, route.agentId, setupCtx.workspaceId, setupCtx.agentId, setupCtx.scope, setupCtx.focus, legacyContextReady]);
  useEffect(() => {
    if (hidden || route.view === "messages") return;
    refresh();
    // ADR-0087: refresh stale update checks once per visit, server-side
    // cached — never a polling timer.
    const stale = (c) => !c.installed || !c.diagnostic || !c.diagnostic.updateCheckedAt || Date.now() - new Date(c.diagnostic.updateCheckedAt).getTime() > 6 * 3600 * 1000;
    api("/api/clis").then((catalog) => {
      for (const c of catalog.clis || []) {
        if (stale(c)) api(`/api/clis/${c.id}/update-check`, json("POST", {})).catch(() => {});
      }
    }).catch(() => {});
    let timer;
    const unsub = subscribeFeed((e) => {
      if (/^(cli\.|terminal\.|workspace\.|feed\.(open|reset))/.test(e.type)) {
        clearTimeout(timer); timer = setTimeout(refresh, 80);
      }
    });
    const focus = () => refresh(); window.addEventListener("focus", focus);
    return () => { unsub(); clearTimeout(timer); window.removeEventListener("focus", focus); };
  }, [hidden, route.view, refresh]);

  const run = async (key, fn) => {
    if (busy) return;
    setBusy(key);
    try { await fn(); await refresh(); } catch (e) { toastError(e); throw e; } finally { setBusy(""); }
  };
  const selected = data?.clis.find((c) => c.id === route.id) || data?.clis[0];
  const pane = route.pane || "launch";
  useEffect(() => { setLaunchEditing(false); }, [selected?.id]);
  useEffect(() => { if (pane !== "launch") setLaunchEditing(false); }, [pane]);
  const action = async (t, op) => {
    const destructive = op === "remove" || (t.running && op !== "start");
    if (destructive && !(await askConfirm({ title: `${op === "remove" ? "Remove" : op === "stop" ? "Stop" : "Restart"} ${t.name}?`, message: t.running ? "This ends the processes running in this terminal." : "Remove this saved terminal and its launch settings?", confirmLabel: op === "remove" ? "Remove terminal" : op === "stop" ? "Stop terminal" : "Restart terminal", danger: true }))) return;
    try {
      await run(t.id + ":" + op, async () => {
        await api(`/api/terminals/${encodeURIComponent(t.id)}/launch/${op}`, json("POST", { confirm: destructive }));
        if (op === "start") location.hash = termHash(t.id);
        else toast.ok(op === "stop" ? "Terminal stopped." : op === "restart" ? "Terminal restarted." : "Terminal removed.");
      });
    } catch { /* toast from run */ }
  };

  const sessionsWs = pane === "sessions" && route.workspace && data ? data.workspaces.find((w) => w.id === route.workspace) : undefined;

  const selectedJob = data?.jobs?.find((j) => j.cli === selected?.id);
  const lifecycleBusy = !!busy || (selectedJob && (selectedJob.state === "queued" || selectedJob.state === "running"));
  const updateLine = selected ? updateCheckLine(selected) : null;
  const checkUpdates = () => run("uc:" + selected.id, () => api(`/api/clis/${selected.id}/update-check`, json("POST", {}))).catch(() => {});
  const startLifecycle = async (action) => {
    const cli = selected;
    if (action === "uninstall") {
      const ok = await askConfirm({
        title: `Uninstall ${cli.name}?`,
        message: "This removes the CLI from this machine. Your settings and conversations stay.",
        confirmLabel: "Uninstall",
        danger: true,
        choices: [{ id: "confirm", label: `I understand — type "${cli.name}" to confirm`, typed: { expected: cli.name } }],
      });
      if (!ok) return;
    }
    await run(action + ":" + cli.id, async () => {
      try {
        await api(`/api/clis/${cli.id}/lifecycle`, json("POST", { action, requestKey: action + "-" + Date.now() }));
      } catch (e) {
        if (!/terminal/.test(e.message || "")) throw e;
        const ok = await askConfirm({ title: `Terminals are running`, message: e.message, confirmLabel: "Run anyway", danger: true });
        if (!ok) return;
        await api(`/api/clis/${cli.id}/lifecycle`, json("POST", { action, requestKey: action + "-" + Date.now(), confirmTerminals: true }));
      }
    });
  };

  if (route.view === "messages") return <PeerMessages hidden={hidden} ownerKey={route.id} />;

  return <AgentClisFrame hidden={hidden}>
    <CliTabs view={route.view} />
    {error ? <Notice danger action="Try again" onAction={refresh}>{error}</Notice> : null}
    {!data && !error ? <div className="cli-loading" aria-label="Loading Agent CLIs"><div /><div /><div /></div> : null}
    {data && !data.terminalAvailable ? <Notice action="Open System" onAction={() => { location.hash = "#/system"; }}>Terminal control is unavailable.</Notice> : null}
    {data && route.view === "clis" && selected ? <div className="cli-layout">
      <nav className="cli-catalog" aria-label="Compatible CLIs">{data.clis.map((c) => <a key={c.id} href={c.id === selected.id ? cliSetupHref(c.id, pane, setupCtx, route.workspace || "") : cliPaneHash(c.id, pane, pane === "sessions" ? (route.workspace || "") : "")} aria-current={c.id === selected.id ? "page" : undefined}>
        <TerminalCliBadge term={{ cli: c.id }} /><span><strong>{c.name}</strong><small>{c.installed ? (c.diagnostic?.updateAvailable ? "Update available" : "Installed") : "Not found"}</small></span>{c.diagnostic?.updateAvailable ? <span className="cli-update-pill">Update</span> : null}<IconChevronRight size={14} />
      </a>)}</nav>
      <div className="cli-detail" key={selected.id}>
        <div className="cli-heading"><div><h3>{selected.name}</h3><p>{selected.diagnostic?.version || (selected.installed ? "Version not checked" : "Not installed")}{selected.diagnostic?.stale ? " · check out of date" : ""}{selected.diagnostic?.updateAvailable ? ` · update available${selected.diagnostic.latest ? " to " + selected.diagnostic.latest : ""}` : ""}</p>{updateLine ? <p>{updateLine}</p> : selected.diagnostic ? <p>Checked {new Date(selected.diagnostic.checkedAt).toLocaleString()}</p> : null}</div><div className="cli-actions" data-align-row data-align-wrap>
          <button className="btn btn-ghost btn-sm" disabled={!!busy} onClick={() => { run("check:" + selected.id, async () => { const d = await api(`/api/clis/${selected.id}/check`, json("POST", {})); if (d.error) toastError(new Error(d.error)); }).catch(() => {}); }}>{busy === "check:" + selected.id ? "Checking…" : "Check setup"}</button>
          {!selected.installed && selected.lifecycle?.canInstall ? <button className="btn btn-primary btn-sm" disabled={!!lifecycleBusy} onClick={() => startLifecycle("install")}>{lifecycleBusy ? "Working…" : "Install"}</button> : null}
          {selected.installed && selected.lifecycle?.canUpdate && selected.diagnostic?.updateAvailable ? <button className="btn btn-primary btn-sm" disabled={!!lifecycleBusy} onClick={() => startLifecycle("update")}>{lifecycleBusy ? "Working…" : "Update"}</button> : null}
          <button className="btn btn-primary btn-sm" disabled={!data.terminalAvailable} onClick={() => navigate("/new/" + selected.id)}>New terminal</button>
          {selected.installed && (selected.lifecycle?.canUpdate || selected.lifecycle?.canReinstall || selected.lifecycle?.uninstall) ? <DropdownMenu.Root><DropdownMenu.Trigger asChild><button className="btn btn-ghost btn-sm cli-more" aria-label={"Lifecycle actions for " + selected.name} disabled={!!lifecycleBusy}>•••</button></DropdownMenu.Trigger><DropdownMenu.Portal><DropdownMenu.Content className="um-popover" align="end" sideOffset={5} collisionPadding={12}>
            <DropdownMenu.Item className="um-item" onSelect={checkUpdates}>Check for updates</DropdownMenu.Item>
            {selected.lifecycle?.canReinstall ? <DropdownMenu.Item className="um-item" onSelect={() => startLifecycle("reinstall")}>Reinstall</DropdownMenu.Item> : null}
            {selected.lifecycle?.uninstall === "vendor" || selected.lifecycle?.uninstall === "npm" ? <><DropdownMenu.Separator className="um-divider" /><DropdownMenu.Item className="um-item cli-danger-item" onSelect={() => startLifecycle("uninstall")}>Uninstall {selected.name}</DropdownMenu.Item></> : null}
          </DropdownMenu.Content></DropdownMenu.Portal></DropdownMenu.Root> : null}
        </div></div>
        <CLILifecycleCard cli={selected} job={selectedJob} onCheck={checkUpdates} />
        {selected.problem ? <Notice action={selected.installed && selected.config.integration && !selected.integrationApplied ? "Repair integration" : "Customize"} onAction={() => {
          if (selected.installed && selected.config.integration && !selected.integrationApplied) run("repair", () => api(`/api/clis/${selected.id}/repair`, json("POST", {}))).catch(() => {});
          else {
            setEditRequested({ id: selected.id, at: Date.now() });
            setLaunchEditing(true);
            if (pane !== "launch") location.hash = cliPaneHash(selected.id, "launch");
          }
        }}>{selected.problem}</Notice> : null}
        {selected.diagnostic?.error ? <Notice danger action="Check again" onAction={() => { run("check:" + selected.id, () => api(`/api/clis/${selected.id}/check`, json("POST", {}))).catch(() => {}); }}>{selected.diagnostic.error}</Notice> : null}
        <CliPaneTabs
          cli={selected.id}
          pane={pane}
          workspace={route.workspace || ""}
          workspaceId={setupCtx.workspaceId}
          agentId={setupCtx.agentId}
          scope={setupCtx.scope}
          focus={setupCtx.focus}
          hasPackageUpdates={packageUpdates.length > 0}
        />
        {pane === "launch" ? <>
          <div className="cli-integration"><label htmlFor="cli-integration">Activity reporting <span>{selected.config.integration ? "On for new launches" : "Off for new launches"}</span></label>
            <Switch.Root id="cli-integration" className="rx-switch" checked={selected.config.integration} disabled={!!busy} onCheckedChange={(value) => { const old = selected.config.integration; setData((cur) => ({ ...cur, clis: cur.clis.map((c) => c.id === selected.id ? { ...c, config: { ...c.config, integration: value } } : c) })); run("integration", async () => { try { const v = await api(`/api/clis/${selected.id}`, json("PUT", { ...selected.config, integration: value })); if (v.problem) toastError(new Error(v.problem)); } catch (e) { setData((cur) => ({ ...cur, clis: cur.clis.map((c) => c.id === selected.id ? { ...c, config: { ...c.config, integration: old } } : c) })); throw e; } }).catch(() => {}); }}><Switch.Thumb className="rx-switch-thumb" /></Switch.Root>
          </div>
          <CLIDefaults cli={selected} editing={launchEditing} onEditingChange={setLaunchEditing} editRequested={editRequested.id === selected.id ? editRequested.at : 0} busy={!!busy} onSave={async (c) => { await run("settings", async () => { const v = await api(`/api/clis/${selected.id}`, json("PUT", c)); if (v.problem) toastError(new Error(v.problem)); else toast.ok("Launch settings saved."); }); }} />
          <CLIDiagnostics cli={selected} terminals={data.terminals} busy={!!busy} run={run} />
          <CLIProfiles cli={selected} profiles={data.profiles} run={run} busy={!!busy} />
          <a className="cli-docs" href={selected.docs} target="_blank" rel="noreferrer">{selected.name} documentation ↗</a>
        </> : null}
        {pane === "terminals" ? <TerminalList hideTitle cliName={selected.name} terminals={cliTerminals(data.terminals, selected.id)} workspaces={data.workspaces} busy={busy} onAction={action} onNew={() => navigate("/new/" + selected.id)} onRenameTerm={onRenameTerm} /> : null}
        {pane === "sessions" ? <SessionsView
          embedded
          wsId={route.workspace || ""}
          workspace={sessionsWs}
          agents={(sessionsWs && sessionsWs.agents) || []}
          workspaces={data.workspaces}
          onOpenAgent={onOpenAgent}
          onCompactAgent={onCompactAgent}
          cli={selected.id}
          wsReady={!!data}
          cliNames={Object.fromEntries(data.clis.map((c) => [c.id, c.name]))}
          clis={data.clis}
          onNewTerminal={() => navigate("/new/" + selected.id)}
        /> : null}
        {pane === "providers" ? <CliProviders
          cli={route.id}
          add={!!route.add}
          invalid={!!route.invalid || !route.id}
          scoped={!!route.scoped}
          onCatalogChange={onCatalogChange}
        /> : null}
        {pane === "settings" ? <CliSettings hidden={false} route={route} catalog={catalog} onAgentConfig={onAgentConfig} /> : null}
        {pane === "packages" ? <CliPackages hidden={false} route={route} catalog={catalog} onPackageUpdates={onPackageUpdates} describe={new URLSearchParams((hash || "").split("?")[1] || "").get("describe") === "1"} /> : null}
        {pane === "connectors" ? <ConnectorsPane route={route} onReload={onReloadAgent} /> : null}
      </div>
    </div> : null}
    {data && (route.view === "new" || route.view === "terminal") ? <TerminalEditor key={hash} route={route} data={data} run={run} busy={!!busy} /> : null}
    {data && route.view === "profile" ? <CLIProfileEditor key={hash} route={route} data={data} run={run} busy={!!busy} /> : null}
  </AgentClisFrame>;
}

function ConnectorsPane({ route, onReload }) {
  const supported = supportsCliConnectors(route.id);
  const [ctx, setCtx] = useState(null);
  const [error, setError] = useState(null);
  useEffect(() => {
    if (route.invalid || !supported) return undefined;
    let live = true;
    loadPiPackagesContext(route, api).then((next) => { if (live) { setCtx(next); setError(null); } }).catch((err) => { if (live) setError(err); });
    return () => { live = false; };
  }, [route.workspaceId, route.agentId, route.scope, route.invalid, supported]);
  if (route.invalid || !supported) {
    return <section id="cli-connectors-view"><div className="cli-notice" role="status"><span>{route.invalid ? "This connector link is invalid." : "Connectors are not available for this CLI."}</span><a className="btn btn-ghost btn-sm" href={cliConnectorsHash("pi")}>Open Pi</a></div></section>;
  }
  if (error) {
    return <section id="cli-connectors-view"><div className="cli-notice is-error" role="alert"><span>{error.message}</span><a className="btn btn-ghost btn-sm" href={cliConnectorsHash("pi")}>Open Pi</a></div></section>;
  }
  if (!ctx) return <section id="cli-connectors-view"><div className="cli-loading" aria-label="Loading connectors"><div /><div /><div /></div></section>;
  const workspace = ctx.workspace, agent = ctx.agent;
  return <Mcps
    embedded
    hidden={false}
    workspaceId={workspace?.id || ""}
    workspaceName={workspace?.name || ""}
    workspacePath={workspace?.path || ""}
    agentId={agent?.id || ""}
    agentName={agent ? displayAgentName(agent, workspace) : ""}
    agentWorkPath={agent?.workPath || ""}
    agentRunning={!!(agent && agent.mode && agent.mode !== "stopped")}
    onReload={agent?.id && onReload ? () => onReload(agent.id) : undefined}
  />;
}

function TerminalList({ terminals, workspaces, busy, onAction, onNew, cliName, onRenameTerm, hideTitle = false }) {
  const [query, setQuery] = useState("");
  const filtered = terminals.filter((t) => `${t.name} ${t.cwd} ${t.cli || ""} ${t.launchCli || ""}`.toLowerCase().includes(query.toLowerCase()));
  const search = terminals.length > 4 ? <input aria-label="Search terminals" placeholder="Find a terminal…" value={query} onChange={(e) => setQuery(e.target.value)} /> : null;
  return <section className="cli-terminals">{!hideTitle || search ? <div className="cli-list-heading">{hideTitle ? null : <h3>Terminals</h3>}{search}</div> : null}
    {!terminals.length ? <Notice action={"Open " + cliName} onAction={onNew}>No {cliName} terminals yet.</Notice> : !filtered.length ? <Notice action="Clear search" onAction={() => setQuery("")}>No matching terminals.</Notice> : null}
    <div className="cli-terminal-list">{filtered.map((t) => {
      const workspace = workspaces.find((w) => w.id === t.workspaceId);
      return <article className={"cli-terminal-row" + (busy.startsWith(t.id + ":") ? " is-busy" : "")} key={t.id}>
        <TerminalCliBadge term={t} />
        <div className="cli-terminal-info"><a href={termHash(t.id)}>{t.name}</a><p>{workspace?.name || "Free terminal"} · <span title={t.cwd}>{t.cwd}</span></p><div className="cli-terminal-meta"><span className={"cli-state is-" + terminalStatus(t)}>{busy.startsWith(t.id + ":") ? "Updating…" : terminalStatusLabel(t)}</span>{t.running && terminalCli(t) && !t.state ? <span>Activity not reported</span> : null}{t.launchPending ? <span>Launch changes pending</span> : null}{t.launchAttempt?.error ? <a className="cli-field-error" href={"#/clis/terminal/" + t.id}>Last launch failed · review settings</a> : null}</div></div>
        <div className="cli-actions" data-align-row data-align-wrap><button className="btn btn-ghost btn-sm" disabled={!!busy} onClick={() => t.running ? (location.hash = termHash(t.id)) : onAction(t, "start")}>{t.running ? "Open" : "Start"}</button>
          <DropdownMenu.Root><DropdownMenu.Trigger asChild><button className="btn btn-ghost btn-sm cli-more" aria-label={"Actions for " + t.name} disabled={!!busy}>•••</button></DropdownMenu.Trigger><DropdownMenu.Portal><DropdownMenu.Content className="um-popover" align="end" sideOffset={5} collisionPadding={12}>
            {/* Same menu as the sidebar's terminal rows (termRowMenu.js) — one contract, two surfaces. */}
            {termRowMenu(t).map((r) => r.sep
              ? <DropdownMenu.Separator key={"sep"} className="um-divider" />
              : <DropdownMenu.Item
                  key={r.id}
                  className={"um-item" + (r.danger ? " cli-danger-item" : "")}
                  title={r.title}
                  onSelect={() => {
                    if (r.id === "launch") { navigate("/terminal/" + encodeURIComponent(t.id)); return; }
                    if (r.id === "settings") { location.hash = "#/termset/" + encodeURIComponent(t.id); return; }
                    if (r.id === "rename") return onRenameTerm && onRenameTerm(t);
                    return onAction(t, r.id); // start | restart | stop | remove
                  }}
                >{r.label}</DropdownMenu.Item>
            )}
          </DropdownMenu.Content></DropdownMenu.Portal></DropdownMenu.Root>
        </div>
      </article>;
    })}</div>
  </section>;
}

function TerminalEditor({ route, data, run, busy }) {
  const existing = route.view === "terminal" ? data.terminals.find((t) => t.id === route.id) : null;
  const [cliId, setCliId] = useState(() => terminalLaunchCLI(existing, route.id));
  const cli = data.clis.find((c) => c.id === cliId) || data.clis[0];
  const [initialProfile] = useState(() => data.profiles.find((p) => p.id === route.profile && p.cli === cli.id));
  const [profileId, setProfileId] = useState(initialProfile?.id || "");
  const [form, setForm] = useState({ name: existing?.name || initialProfile?.name || cli.name, workspaceId: existing?.workspaceId || route.workspace || "", cwd: existing?.cwd || "" });
  const [custom, setCustom] = useState(!!initialProfile);
  const [draft, setDraft] = useState(() => launchDraft(initialProfile?.config || cli.config));
  const [originalOverrides, setOriginalOverrides] = useState({});
  const [dirty, setDirty] = useState(false);
  const allowNavigation = useLaunchGuard(dirty);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(!!existing);
  const [loadError, setLoadError] = useState("");
  const [retry, setRetry] = useState(0);
  useEffect(() => {
    if (!existing) return;
    setLoading(true); setLoadError("");
    let stopped = false;
    api(`/api/terminals/${encodeURIComponent(existing.id)}/launch`).then((v) => {
      if (stopped) return;
      if (v) { const c = data.clis.find((x) => x.id === v.cli) || cli; setCliId(c.id); setDraft(launchDraft(resolveLaunch(c.config, v.overrides))); setOriginalOverrides(v.overrides || {}); setCustom(Object.keys(v.overrides || {}).length > 0); }
    }).catch((e) => { if (!stopped) setLoadError(e.message); }).finally(() => { if (!stopped) setLoading(false); });
    return () => { stopped = true; };
  }, [retry]); // this editor is keyed by its route
  if (route.view === "terminal" && !existing) return <Notice action="Choose a CLI" onAction={() => navigate("")}>That terminal is gone.</Notice>;
  if (route.profile && !initialProfile) return <Notice action="Choose a profile" onAction={() => { location.hash = cliPaneHash(cli.id, "launch"); }}>That launch profile is no longer available.</Notice>;
  if (loadError) return <Notice danger action="Try again" onAction={() => setRetry((v) => v + 1)}>{loadError}</Notice>;
  const settingsPreview = parseForm(cliLaunchSchema, draft);
  const overrides = custom && settingsPreview.ok ? (profileId ? profileOverrides(cli.config, launchConfig(settingsPreview.value)) : editLaunchOverrides(cli.config, originalOverrides, launchConfig(settingsPreview.value))) : {};
  const back = async () => { if (!dirty || await confirmDiscard()) { allowNavigation(); location.hash = cliPaneHash(cli.id, existing ? "terminals" : "launch"); } };
  return <section className="cli-editor"><div className="cli-heading"><h3>{existing ? existing.name + " · Launch settings" : "New " + cli.name + " terminal"}</h3><button className="btn btn-ghost btn-sm" onClick={back}>Back</button></div>
    {loading ? <div className="cli-loading" aria-label="Loading launch settings"><div /><div /></div> : <form noValidate onChange={() => setDirty(true)} onSubmit={async (e) => {
      e.preventDefault(); const parsed = parseForm(cliTerminalSchema, form); if (!parsed.ok) { setError(parsed.error); return; }
      const settings = parseForm(cliLaunchSchema, draft); if (custom && !settings.ok) { setError(settings.error); return; }
      try { await run("terminal-save", async () => {
        if (existing) { await api(`/api/terminals/${encodeURIComponent(existing.id)}/launch`, json("PUT", { cli: cli.id, overrides })); allowNavigation(); toast.ok("Settings saved for the next launch."); location.hash = cliPaneHash(cli.id, "terminals"); }
        else { const t = await api(`/api/clis/${cli.id}/terminals`, json("POST", { ...parsed.value, overrides })); allowNavigation(); if (t.launchError) { toastError(new Error(t.launchError)); location.hash = cliPaneHash(cli.id, "terminals"); } else location.hash = termHash(t.id); }
      }); } catch (e) { setError(e.message); }
    }}>
      <div className="cli-fields">
        <label>CLI<CliCombo ariaLabel="CLI" value={cli.id} options={data.clis} onChange={async (id) => { const c = data.clis.find((x) => x.id === id); if (!c || (custom && dirty && !(await confirmDiscard()))) return; setCliId(c.id); setDraft(launchDraft(c.config)); setCustom(false); setProfileId(""); setOriginalOverrides({}); }} /></label>
        {!existing && data.profiles.some((p) => p.cli === cli.id) ? <label>Launch profile<select value={profileId} onChange={async (e) => { const id = e.target.value; if (custom && dirty && !(await confirmDiscard())) return; const p = data.profiles.find((p) => p.id === id); setProfileId(id); setDraft(launchDraft(p?.config || cli.config)); setCustom(!!p); setOriginalOverrides({}); }}>{<option value="">CLI defaults</option>}{data.profiles.filter((p) => p.cli === cli.id).map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}</select></label> : null}
        {!existing ? <><label>Name<input autoComplete="off" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} /></label><label>Workspace<select value={form.workspaceId} onChange={(e) => setForm({ ...form, workspaceId: e.target.value, cwd: "" })}><option value="">Free terminal</option>{data.workspaces.filter((w) => w.id !== "ws_free").map((w) => <option key={w.id} value={w.id}>{w.name}</option>)}</select></label><label>Folder<input placeholder={form.workspaceId ? "Use workspace folder" : "Use home folder"} value={form.cwd} onChange={(e) => setForm({ ...form, cwd: e.target.value })} /></label></> : null}
        <label className="cli-checkbox"><input type="checkbox" checked={custom} onChange={async (e) => { const checked = e.target.checked; if (!checked && dirty && !(await confirmDiscard())) return; setCustom(checked); if (!checked) { setDraft(launchDraft(cli.config)); setProfileId(""); setOriginalOverrides({}); } }} />Customize this terminal</label>
      </div>
      {custom ? <LaunchFields draft={draft} setDraft={setDraft} includeIntegration /> : <p className="cli-muted">Uses {cli.name} launch defaults.</p>}
      {existing?.launchAttempt?.error ? <Notice danger action="Back to terminals" onAction={back}>{existing.launchAttempt.error}</Notice> : null}
      {error ? <p className="cli-field-error" role="alert">{error}</p> : null}
      <div className="cli-actions" data-align-row data-align-wrap><button type="submit" className="btn btn-primary btn-sm" disabled={busy || (!existing && !data.terminalAvailable)}>{busy ? (existing ? "Saving…" : "Opening terminal…") : existing ? "Save launch settings" : "Open terminal"}</button><button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={back}>Cancel</button></div>
      {!custom || settingsPreview.ok ? <LaunchPreview cli={cli.id} overrides={overrides} terminalId={existing?.id} applied={existing?.launchApplied} /> : null}
    </form>}
  </section>;
}

function CLIDiagnostics({ cli, terminals, busy, run }) {
  const observed = terminals.filter((t) => t.stateAt && t.state && (t.tui?.cli || t.cli) === cli.id).sort((a, b) => new Date(b.stateAt) - new Date(a.stateAt))[0];
  return <details className="cli-diagnostics"><summary>Setup and activity</summary><dl className="cli-summary-grid">
    <dt>Executable</dt><dd>{cli.installed ? "Found" : "Not found"}</dd>
    <dt>Version check</dt><dd>{cli.diagnostic?.stale ? "Out of date · check again" : cli.diagnostic?.version ? cli.diagnostic.version : "Not verified"}</dd>
    <dt>Integration files</dt><dd>{!cli.config.integration ? "Off for new launches" : cli.integrationApplied ? "Prepared" : "Repair needed"}</dd>
    <dt>Reporter tools</dt><dd>{cli.diagnostic?.prerequisites && !cli.diagnostic.stale ? "Checked" : "Not verified"}</dd>
    <dt>Last activity signal</dt><dd>{observed ? <>{terminalStatusLabel(observed)} · <a href={termHash(observed.id)}>{observed.name}</a><small>{new Date(observed.stateAt).toLocaleString()}</small></> : "No current signal observed"}</dd>
  </dl><button className="btn btn-ghost btn-sm" disabled={busy} onClick={() => { run("repair", () => api(`/api/clis/${cli.id}/repair`, json("POST", {}))).catch(() => {}); }}>{busy ? "Updating…" : "Repair PiCode files"}</button></details>;
}

const jobStateLabel = { queued: "Waiting to start", running: "Running", succeeded: "Done", failed: "Failed", interrupted: "Interrupted" };
const jobActionLabel = { install: "Install", uninstall: "Uninstall", reinstall: "Reinstall", update: "Update" };

// updateCheckLine is the detail header's update-check sentence. A failed
// check on an install PiCode does not manage is not a failure — guidance
// lives in the lifecycle card, so the line stays out.
function updateCheckLine(cli) {
  const d = cli.diagnostic;
  if (!d?.updateCheckedAt) return null;
  if (d.updateError && !cli.lifecycle?.canUpdate) return null;
  if (d.updateError) return "Update check failed · " + d.updateError;
  return "Update check saved " + new Date(d.updateCheckedAt).toLocaleString();
}

// CLILifecycleCard surfaces the latest lifecycle job for one CLI plus the
// guided-uninstall facts for install methods without an uninstall command
// (ADR-0087). One line + one action per state, never a blank well.
function CLILifecycleCard({ cli, job, onCheck }) {
  const active = job && (job.state === "queued" || job.state === "running");
  const guidedInstall = !cli.installed && !cli.lifecycle?.canInstall && !!cli.lifecycle?.docs;
  if (!active && !job && cli.lifecycle?.uninstall !== "guided" && !guidedInstall) return null;
  return <div className="cli-job-card" role="status" data-state={job ? job.state : "guided"}>
    {job ? <>
      <div className="cli-job-head"><span className={"cli-job-state is-" + job.state}>{jobStateLabel[job.state] || job.state}</span><span>{jobActionLabel[job.action] || job.action} · {cli.name}</span></div>
      <p>{job.message}</p>
      {job.output ? <details className="cli-job-log"><summary>Output</summary><pre>{job.output}</pre></details> : null}
      {job.state === "interrupted" ? <button className="btn btn-ghost btn-sm" onClick={onCheck}>Check result</button> : null}
      {job.state === "failed" ? <button className="btn btn-ghost btn-sm" onClick={onCheck}>Check setup</button> : null}
    </> : null}
    {!active && cli.lifecycle?.uninstall === "guided" ? <p className="cli-job-guided">Uninstalling {cli.name} follows its own guide for this install type. <a href={cli.lifecycle.docs || cli.docs} target="_blank" rel="noreferrer">Open the uninstall guide ↗</a></p> : null}
    {guidedInstall ? <p className="cli-job-guided">Installing {cli.name} follows its own guide. <a href={cli.lifecycle.docs || cli.docs} target="_blank" rel="noreferrer">Open the install guide ↗</a></p> : null}
  </div>;
}
