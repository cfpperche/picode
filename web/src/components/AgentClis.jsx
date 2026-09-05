import { useCallback, useEffect, useRef, useState } from "react";
import * as Switch from "@radix-ui/react-switch";
import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import { api } from "../lib/api.js";
import { subscribeFeed } from "../lib/feed.js";
import { askConfirm } from "../lib/confirm.js";
import { toast, toastError } from "../lib/toast.js";
import { cliLaunchSchema, cliTerminalSchema, parseForm } from "../lib/schemas.js";
import { cliLocation, launchDraft, launchConfig, launchOverrides, resolveLaunch, cliTerminals, terminalLaunchCLI } from "../lib/cliLaunch.js";
import { terminalCli, terminalStatusLabel, terminalStatus } from "../lib/terminalCli.js";
import { termHash } from "../lib/routes.js";
import PageFrame from "./PageFrame.jsx";
import TerminalCliBadge from "./TerminalCliBadge.jsx";
import { IconChevronRight } from "./Icons.jsx";
import "./agent-clis.css";

const json = (method, body) => ({ method, headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
const navigate = (path) => { location.hash = "#/clis" + path; };

function Notice({ children, action, onAction, danger = false }) {
  return <div className={"cli-notice" + (danger ? " is-error" : "")} role={danger ? "alert" : "status"}><span>{children}</span>{action ? <button type="button" className="btn btn-ghost btn-sm" onClick={onAction}>{action}</button> : null}</div>;
}

export default function AgentClis({ hidden = false }) {
  const [hash, setHash] = useState(location.hash);
  const route = cliLocation(hash);
  const [data, setData] = useState(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState("");
  const active = useRef(!hidden);
  const request = useRef(0);
  active.current = !hidden;
  const refresh = useCallback(async () => {
    const seq = ++request.current;
    try {
      const [catalog, terms, work] = await Promise.all([api("/api/clis"), api("/api/terminals"), api("/api/workspaces")]);
      if (seq !== request.current || !active.current) return;
      setData({ ...catalog, terminals: terms.terminals || [], workspaces: work.workspaces || [] }); setError("");
    } catch (e) { if (seq === request.current && active.current) setError(e.message); }
  }, []);
  useEffect(() => {
    const update = () => setHash(location.hash);
    window.addEventListener("hashchange", update);
    return () => window.removeEventListener("hashchange", update);
  }, []);
  useEffect(() => {
    if (!hidden && hash === "#/preferences/status") location.replace("#/clis");
  }, [hidden, hash]);
  useEffect(() => {
    if (hidden) return;
    refresh();
    let timer;
    const unsub = subscribeFeed((e) => {
      if (/^(cli\.|terminal\.|workspace\.|feed\.(open|reset))/.test(e.type)) {
        clearTimeout(timer); timer = setTimeout(refresh, 80);
      }
    });
    const focus = () => refresh(); window.addEventListener("focus", focus);
    return () => { unsub(); clearTimeout(timer); window.removeEventListener("focus", focus); };
  }, [hidden, refresh]);

  const run = async (key, fn) => {
    if (busy) return;
    setBusy(key);
    try { await fn(); await refresh(); } catch (e) { toastError(e); throw e; } finally { setBusy(""); }
  };
  const selected = data?.clis.find((c) => c.id === route.id) || data?.clis[0];
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

  return <PageFrame id="agent-clis-view" title="Agent CLIs" hidden={hidden} wide>
    <nav className="cli-tabs" aria-label="Agent CLIs">
      <a href="#/clis" aria-current={route.view !== "terminals" ? "page" : undefined}>CLIs</a>
      <a href="#/clis/terminals" aria-current={route.view === "terminals" ? "page" : undefined}>Terminals</a>
    </nav>
    {error ? <Notice danger action="Try again" onAction={refresh}>{error}</Notice> : null}
    {!data && !error ? <div className="cli-loading" aria-label="Loading Agent CLIs"><div /><div /><div /></div> : null}
    {data && !data.terminalAvailable ? <Notice action="Open System" onAction={() => { location.hash = "#/system"; }}>Terminal control is unavailable.</Notice> : null}
    {data && route.view === "clis" && selected ? <div className="cli-layout">
      <nav className="cli-catalog" aria-label="Compatible CLIs">{data.clis.map((c) => <a key={c.id} href={"#/clis/" + c.id} aria-current={c.id === selected.id ? "page" : undefined}>
        <TerminalCliBadge term={{ cli: c.id }} /><span><strong>{c.name}</strong><small>{c.installed ? "Installed" : "Not found"}</small></span><IconChevronRight size={14} />
      </a>)}</nav>
      <div className="cli-detail" key={selected.id}>
        <div className="cli-heading"><div><h3>{selected.name}</h3><p>{selected.diagnostic?.version || (selected.installed ? "Version not checked" : "Executable not found")}</p></div><div className="cli-actions" data-align-row>
          <button className="btn btn-ghost btn-sm" disabled={!!busy} onClick={() => { run("check:" + selected.id, async () => { const d = await api(`/api/clis/${selected.id}/check`, json("POST", {})); if (d.error) toastError(new Error(d.error)); }).catch(() => {}); }}>{busy === "check:" + selected.id ? "Checking…" : "Check setup"}</button>
          <button className="btn btn-primary btn-sm" disabled={!data.terminalAvailable} onClick={() => navigate("/new/" + selected.id)}>New terminal</button>
        </div></div>
        {selected.problem ? <Notice action="Edit launch" onAction={() => { const el = document.querySelector("#cli-launch-settings"); if (el) { el.open = true; el.scrollIntoView({ block: "center", behavior: "smooth" }); el.querySelector("input")?.focus({ preventScroll: true }); } }}>{selected.problem}</Notice> : null}
        {selected.diagnostic?.error ? <Notice danger action="Check again" onAction={() => { run("check:" + selected.id, () => api(`/api/clis/${selected.id}/check`, json("POST", {}))).catch(() => {}); }}>{selected.diagnostic.error}</Notice> : null}
        {selected.diagnostic && !selected.diagnostic.error ? <p className="cli-check-result">Executable responds · checked {new Date(selected.diagnostic.checkedAt).toLocaleTimeString()}</p> : null}
        <div className="cli-integration"><label htmlFor="cli-integration">Activity reporting <span>{selected.config.integration ? "On for new launches" : "Off for new launches"}</span></label>
          <Switch.Root id="cli-integration" className="rx-switch" checked={selected.config.integration} disabled={!!busy} onCheckedChange={(value) => { const old = selected.config.integration; setData((cur) => ({ ...cur, clis: cur.clis.map((c) => c.id === selected.id ? { ...c, config: { ...c.config, integration: value } } : c) })); run("integration", async () => { try { const v = await api(`/api/clis/${selected.id}`, json("PUT", { ...selected.config, integration: value })); if (v.problem) toastError(new Error(v.problem)); } catch (e) { setData((cur) => ({ ...cur, clis: cur.clis.map((c) => c.id === selected.id ? { ...c, config: { ...c.config, integration: old } } : c) })); throw e; } }).catch(() => {}); }}><Switch.Thumb className="rx-switch-thumb" /></Switch.Root>
        </div>
        <details className="cli-settings" id="cli-launch-settings"><summary>Launch settings</summary><LaunchForm config={selected.config} busy={!!busy} onSave={async (c) => { await run("settings", async () => { const v = await api(`/api/clis/${selected.id}`, json("PUT", c)); if (v.problem) toastError(new Error(v.problem)); else toast.ok("Launch settings saved."); }); }} /></details>
        <TerminalList terminals={cliTerminals(data.terminals, selected.id)} workspaces={data.workspaces} busy={busy} onAction={action} onNew={() => navigate("/new/" + selected.id)} />
        <a className="cli-docs" href={selected.docs} target="_blank" rel="noreferrer">{selected.name} documentation ↗</a>
      </div>
    </div> : null}
    {data && route.view === "terminals" ? <TerminalList terminals={cliTerminals(data.terminals)} workspaces={data.workspaces} busy={busy} onAction={action} onNew={() => navigate("")} all /> : null}
    {data && (route.view === "new" || route.view === "terminal") ? <TerminalEditor key={hash} route={route} data={data} run={run} busy={!!busy} /> : null}
  </PageFrame>;
}

function LaunchFields({ draft, setDraft, includeIntegration = false }) {
  const field = (key) => ({ value: draft[key], onChange: (e) => setDraft({ ...draft, [key]: e.target.value }) });
  return <div className="cli-fields">
    <label>Executable<input type="text" placeholder="Detect automatically" autoComplete="off" {...field("executable")} /></label>
    <label>Arguments <span>One argument per line</span><textarea rows={3} spellCheck={false} {...field("argsText")} /></label>
    <label>Extra PATH entries <span>One directory per line</span><textarea rows={2} placeholder="/opt/tools/bin" spellCheck={false} {...field("pathText")} /></label>
    <label>Environment <span>One NAME=value per line</span><textarea rows={3} autoComplete="off" spellCheck={false} {...field("envText")} /></label>
    {includeIntegration ? <label className="cli-checkbox"><input type="checkbox" checked={draft.integration} onChange={(e) => setDraft({ ...draft, integration: e.target.checked })} />Report activity</label> : null}
  </div>;
}

function LaunchForm({ config, onSave, busy }) {
  const [draft, setDraft] = useState(() => launchDraft(config));
  const [dirty, setDirty] = useState(false);
  const [error, setError] = useState("");
  useEffect(() => { if (!dirty) setDraft(launchDraft(config)); }, [config, dirty]);
  return <form noValidate onSubmit={async (e) => { e.preventDefault(); const parsed = parseForm(cliLaunchSchema, { ...draft, integration: config.integration }); if (!parsed.ok) { setError(parsed.error); return; } try { await onSave(launchConfig(parsed.value)); setDirty(false); setError(""); } catch (e) { setError(e.message); } }}>
    <LaunchFields draft={draft} setDraft={(v) => { setDraft(v); setDirty(true); }} />
    {error ? <p className="cli-field-error" role="alert">{error}</p> : null}
    <div className="cli-actions" data-align-row><button type="submit" className="btn btn-primary btn-sm" disabled={busy || !dirty}>{busy ? "Saving…" : "Save launch settings"}</button>{dirty ? <button type="button" className="btn btn-ghost btn-sm" onClick={() => { setDraft(launchDraft(config)); setDirty(false); }}>Discard</button> : null}</div>
  </form>;
}

function TerminalList({ terminals, workspaces, busy, onAction, onNew, all }) {
  const [query, setQuery] = useState("");
  const filtered = terminals.filter((t) => `${t.name} ${t.cwd} ${t.cli || ""} ${t.launchCli || ""}`.toLowerCase().includes(query.toLowerCase()));
  return <section className="cli-terminals"><div className="cli-list-heading"><h3>{all ? "CLI terminals" : "Terminals"}</h3>{terminals.length > 4 ? <input aria-label="Search terminals" placeholder="Find a terminal…" value={query} onChange={(e) => setQuery(e.target.value)} /> : null}</div>
    {!terminals.length ? <Notice action="New terminal" onAction={onNew}>No CLI terminals yet.</Notice> : !filtered.length ? <Notice action="Clear search" onAction={() => setQuery("")}>No matching terminals.</Notice> : null}
    <div className="cli-terminal-list">{filtered.map((t) => {
      const workspace = workspaces.find((w) => w.id === t.workspaceId);
      return <article className={"cli-terminal-row" + (busy.startsWith(t.id + ":") ? " is-busy" : "")} key={t.id}>
        <TerminalCliBadge term={t} />
        <div className="cli-terminal-info"><a href={termHash(t.id)}>{t.name}</a><p>{workspace?.name || "Free terminal"} · <span title={t.cwd}>{t.cwd}</span></p><div className="cli-terminal-meta"><span className={"cli-state is-" + terminalStatus(t)}>{busy.startsWith(t.id + ":") ? "Updating…" : terminalStatusLabel(t)}</span>{t.running && terminalCli(t) && !t.state ? <span>Activity not reported</span> : null}{t.launchPending ? <span>Launch changes pending</span> : null}</div></div>
        <div className="cli-actions" data-align-row><button className="btn btn-ghost btn-sm" disabled={!!busy} onClick={() => t.running ? (location.hash = termHash(t.id)) : onAction(t, "start")}>{t.running ? "Open" : "Start"}</button>
          <DropdownMenu.Root><DropdownMenu.Trigger asChild><button className="btn btn-ghost btn-sm cli-more" aria-label={"Actions for " + t.name} disabled={!!busy}>•••</button></DropdownMenu.Trigger><DropdownMenu.Portal><DropdownMenu.Content className="um-popover" align="end" sideOffset={5} collisionPadding={12}>
            <DropdownMenu.Item className="um-item" onSelect={() => navigate("/terminal/" + encodeURIComponent(t.id))}>Launch settings</DropdownMenu.Item>
            {t.running ? <><DropdownMenu.Item className="um-item" onSelect={() => onAction(t, "restart")}>Restart terminal</DropdownMenu.Item><DropdownMenu.Item className="um-item" onSelect={() => onAction(t, "stop")}>Stop terminal</DropdownMenu.Item></> : null}
            <DropdownMenu.Separator className="um-divider" /><DropdownMenu.Item className="um-item" onSelect={() => onAction(t, "remove")}>Remove terminal</DropdownMenu.Item>
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
  const [form, setForm] = useState({ name: existing?.name || cli.name, workspaceId: existing?.workspaceId || "", cwd: existing?.cwd || "" });
  const [custom, setCustom] = useState(false);
  const [draft, setDraft] = useState(() => launchDraft(cli.config));
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
      if (v) { const c = data.clis.find((x) => x.id === v.cli) || cli; setCliId(c.id); setDraft(launchDraft(resolveLaunch(c.config, v.overrides))); setCustom(Object.keys(v.overrides || {}).length > 0); }
    }).catch((e) => { if (!stopped) setLoadError(e.message); }).finally(() => { if (!stopped) setLoading(false); });
    return () => { stopped = true; };
  }, [retry]); // this editor is keyed by its route
  if (route.view === "terminal" && !existing) return <Notice action="Back to terminals" onAction={() => navigate("/terminals")}>That terminal is gone.</Notice>;
  if (loadError) return <Notice danger action="Try again" onAction={() => setRetry((v) => v + 1)}>{loadError}</Notice>;
  return <section className="cli-editor"><div className="cli-heading"><h3>{existing ? existing.name + " · Launch settings" : "New " + cli.name + " terminal"}</h3><button className="btn btn-ghost btn-sm" onClick={() => navigate(existing ? "/terminals" : "/" + cli.id)}>Back</button></div>
    {loading ? <div className="cli-loading" aria-label="Loading launch settings"><div /><div /></div> : <form noValidate onSubmit={async (e) => {
      e.preventDefault(); const parsed = parseForm(cliTerminalSchema, form); if (!parsed.ok) { setError(parsed.error); return; }
      const settings = parseForm(cliLaunchSchema, draft); if (custom && !settings.ok) { setError(settings.error); return; }
      const overrides = custom ? launchOverrides(cli.config, launchConfig(settings.value)) : {};
      try { await run("terminal-save", async () => {
        if (existing) { await api(`/api/terminals/${encodeURIComponent(existing.id)}/launch`, json("PUT", { cli: cli.id, overrides })); toast.ok("Settings saved for the next launch."); navigate("/terminals"); }
        else { const t = await api(`/api/clis/${cli.id}/terminals`, json("POST", { ...parsed.value, overrides })); if (t.launchError) { toastError(new Error(t.launchError)); navigate("/terminals"); } else location.hash = termHash(t.id); }
      }); } catch (e) { setError(e.message); }
    }}>
      <div className="cli-fields">
        <label>CLI<select value={cli.id} onChange={(e) => { const c = data.clis.find((x) => x.id === e.target.value); setCliId(c.id); setDraft(launchDraft(c.config)); setCustom(false); }}>{data.clis.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}</select></label>
        {!existing ? <><label>Name<input autoComplete="off" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} /></label><label>Workspace<select value={form.workspaceId} onChange={(e) => setForm({ ...form, workspaceId: e.target.value, cwd: "" })}><option value="">Free terminal</option>{data.workspaces.filter((w) => w.id !== "ws_free").map((w) => <option key={w.id} value={w.id}>{w.name}</option>)}</select></label><label>Folder<input placeholder={form.workspaceId ? "Use workspace folder" : "Use home folder"} value={form.cwd} onChange={(e) => setForm({ ...form, cwd: e.target.value })} /></label></> : null}
        <label className="cli-checkbox"><input type="checkbox" checked={custom} onChange={(e) => { setCustom(e.target.checked); if (!e.target.checked) setDraft(launchDraft(cli.config)); }} />Customize this terminal</label>
      </div>
      {custom ? <LaunchFields draft={draft} setDraft={setDraft} includeIntegration /> : <p className="cli-muted">Uses {cli.name} launch defaults.</p>}
      {existing?.launchApplied ? <AppliedLaunch applied={existing.launchApplied} clis={data.clis} /> : null}
      {error ? <p className="cli-field-error" role="alert">{error}</p> : null}
      <div className="cli-actions" data-align-row><button type="submit" className="btn btn-primary btn-sm" disabled={busy || (!existing && !data.terminalAvailable)}>{busy ? (existing ? "Saving…" : "Opening terminal…") : existing ? "Save launch settings" : "Open terminal"}</button><button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={() => navigate("/terminals")}>Cancel</button></div>
    </form>}
  </section>;
}

function AppliedLaunch({ applied, clis }) {
  return <details className="cli-effective"><summary>Last applied launch</summary><dl>
    <dt>CLI</dt><dd>{clis.find((c) => c.id === applied.cli)?.name || applied.cli}</dd>
    <dt>Executable</dt><dd>{applied.executable}</dd>
    <dt>Arguments</dt><dd>{applied.args?.join(" · ") || "None"}</dd>
    <dt>Extra PATH</dt><dd>{applied.path?.join(" · ") || "None"}</dd>
    <dt>Environment</dt><dd>{applied.envKeys?.map((k) => k + "=••••").join(" · ") || "Inherited"}</dd>
    <dt>Integration</dt><dd>{applied.integration ? "On" : "Off"}</dd>
    <dt>Started</dt><dd>{new Date(applied.startedAt).toLocaleString()}</dd>
  </dl></details>;
}
