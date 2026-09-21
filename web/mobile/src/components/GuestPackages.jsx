import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import * as Dialog from "./MobileSheet.jsx";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { guestPackagesApi, guestPackagesNotes, catalogRowAction, refusalCommand, rowUpdateState } from "@picode/shared/domain/cliPackages.js";
import { terminalCliLabel } from "@picode/shared/domain/terminalCli.js";
import { askConfirm } from "../lib/confirm.js";
import { toast } from "../lib/toast.js";
import PageFrame from "./PageFrame.jsx";

// The plugins a guest CLI really has (ADR-0167). One pane for the eight CLIs
// whose vendor owns the plugin store: every control comes from the answer's
// `caps`, every blank cell from its `notes`, and the vendor binary stays the
// authority — PiCode runs the vendor's own command, never passes its
// auto-consent flag, and shows a refusal verbatim. Pi is not here: its pane is
// Packages.jsx, which this file never touches.
//
// Both apps carry this file (ADR-0072); the only difference is the modal
// primitive on line 2.

const JOB_STATE = { queued: "Waiting to start", running: "Running", succeeded: "Done", failed: "Failed", interrupted: "Interrupted" };
const JOB_ACTION = {
  "pkg-install": "Install",
  "pkg-remove": "Remove",
  "pkg-update": "Update",
  "pkg-marketplace-add": "Add marketplace source",
  "pkg-marketplace-update": "Update marketplace source",
};
const JOB_ENDED = ["succeeded", "failed", "interrupted"];
const isPackageJob = job => !!job && String(job.action || "").startsWith("pkg-");
// A 202 answers with the job itself (store.CLIJob, the lane's shape) — never a
// wrapper: the pane takes `state`/`message`/`output` and follows `cli.job`
// events from there.
const acceptedJob = res => (res && !Array.isArray(res.rows) && res.cli && res.state ? res : null);
// A vendor row's key: its own id, or the name/source it prints when the vendor
// has none. Row errors and pending verbs follow the row, never a shared "".
const keyOf = row => String((row && (row.id || row.name || row.source)) || "");
const newKey = () => (typeof crypto !== "undefined" && crypto.randomUUID ? crypto.randomUUID() : Date.now() + "-" + Math.random());
const json = (method, body) => ({ method, headers: { "Content-Type": "application/json" }, body: body == null ? undefined : JSON.stringify(body) });

// Problem is one failed action: the vendor's own words, and — when the fix is a
// command only a person in a terminal can run — that exact line with a copy
// button. The command is rendered server-side by the same builder that executed
// it, so it is never a line PiCode would not run.
function Problem({ entry }) {
  if (!entry || !entry.message) return entry && entry.command ? <CommandLine command={entry.command} /> : null;
  return (
    <div className="gpkg-problem" role="alert">
      <span className="gpkg-row-note is-error">{entry.message}</span>
      {entry.command ? <CommandLine command={entry.command} /> : null}
    </div>
  );
}

function CommandLine({ command }) {
  return (
    <span className="gpkg-cmd">
      <button
        type="button"
        className="btn btn-ghost btn-sm"
        title={command}
        onClick={() => navigator.clipboard.writeText(command)
          .then(() => toast.ok("Command copied."))
          .catch(() => toast.error("Clipboard blocked — copy it by hand."))}
      >Copy command</button>
      <span className="pkg-fine">Run it in a terminal.</span>
    </span>
  );
}

export default function GuestPackages({ hidden, route, onScopeChange = () => {} }) {
  const cli = route.id;
  const scope = route.scope || "user";
  const workspaceId = route.workspaceId || "";
  const cliName = terminalCliLabel(cli);
  const paths = useMemo(() => guestPackagesApi(cli, { workspaceId, scope }), [cli, workspaceId, scope]);

  const [report, setReport] = useState(null);
  const [problem, setProblem] = useState(null);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState("installed");
  // checkedAt is the last availability check's stamp: an empty value means the
  // catalog was never compared, which is when the pane offers the check itself
  // instead of a blind Update per row (ADR-0167).
  const [checkedAt, setCheckedAt] = useState("");
  const [checking, setChecking] = useState(false);
  const [busy, setBusy] = useState(null);
  const [job, setJob] = useState(null);
  const [rowError, setRowError] = useState({});
  const [sourceProblem, setSourceProblem] = useState("");
  const [installedFilter, setInstalledFilter] = useState("");
  const [source, setSource] = useState("");
  const [market, setMarket] = useState(null);
  const [marketProblem, setMarketProblem] = useState(null);
  const [marketLoading, setMarketLoading] = useState(false);
  const [marketFilter, setMarketFilter] = useState("");
  const [addSource, setAddSource] = useState({ name: "", source: "" });
  const [marketplaces, setMarketplaces] = useState(null);
  const [inspect, setInspect] = useState(null);
  const live = useRef(true);
  const sequence = useRef(0);
  const sourceField = useRef(null);
  const tabNow = useRef(tab);
  tabNow.current = tab;

  // A project layer with no workspace is refused before anything is sent: the
  // pane never falls back to the machine layer on the user's behalf (§6).
  const needsWorkspace = scope === "project" && !workspaceId;
  const caps = (report && report.caps) || {};
  const jobRunning = !!job && !JOB_ENDED.includes(job.state);
  const guard = !!busy || jobRunning;

  const load = useCallback(async (opts = {}) => {
    if (needsWorkspace) return;
    const request = ++sequence.current;
    setLoading(true);
    try {
      const next = await api(paths.roster(opts).path);
      if (!live.current || request !== sequence.current) return;
      setReport(next);
      setProblem(null);
    } catch (ex) {
      // The vendor's own words, verbatim: an unparsable or refused roster is
      // never an empty list, and an offline vendor keeps the last one on screen.
      if (live.current && request === sequence.current) setProblem({ message: ex.message, status: ex.status });
    } finally {
      if (live.current && request === sequence.current) setLoading(false);
    }
  }, [paths, needsWorkspace]);

  const loadMarket = useCallback(async () => {
    if (needsWorkspace) return;
    setMarketLoading(true);
    try {
      const next = await api(paths.available().path);
      if (!live.current) return;
      setMarket({ rows: Array.isArray(next.rows) ? next.rows : [], note: next.note || "" });
      setMarketProblem(null);
    } catch (ex) {
      if (live.current) setMarketProblem({ message: ex.message, status: ex.status });
    } finally {
      if (live.current) setMarketLoading(false);
    }
  }, [paths, needsWorkspace]);

  // The CLI's own configured marketplace sources. A 400 is the contract's
  // "this CLI manages no sources" (Hermes), not a failure; anything else keeps
  // the catalog-derived chips rather than putting a second error over a
  // catalog that loaded.
  const loadSources = useCallback(async () => {
    if (needsWorkspace || !caps.marketplace) return;
    try {
      const next = await api(paths.marketplaces().path);
      if (live.current) setMarketplaces(Array.isArray(next.marketplaces) ? next.marketplaces : []);
    } catch (ex) {
      if (live.current) setMarketplaces(ex.status === 400 ? [] : null);
    }
  }, [paths, needsWorkspace, caps.marketplace]);

  useEffect(() => {
    live.current = true;
    return () => { live.current = false; sequence.current++; };
  }, []);

  useEffect(() => {
    if (needsWorkspace) { setLoading(false); return; }
    load();
  }, [load, needsWorkspace]);

  // The job lane (ADR-0087/0093) keeps one active job per CLI, so a reload
  // mid-install finds it there instead of pretending nothing is running. A
  // terminal state is what re-reads the vendor's roster: success is only
  // success after that read (§6).
  useEffect(() => {
    if (needsWorkspace) return undefined;
    let timer;
    const settle = () => { clearTimeout(timer); timer = setTimeout(() => { load(); if (tabNow.current === "marketplace") { loadMarket(); loadSources(); } }, 80); };
    const off = subscribeFeed(event => {
      if (event.type === "cli.job") {
        const next = event.data;
        if (!isPackageJob(next) || next.cli !== cli) return;
        setJob(next);
        if (JOB_ENDED.includes(next.state)) settle();
        return;
      }
      if (event.type === "cli.packages" && event.data && event.data.cli === cli) settle();
      if (event.type === "feed.open" || event.type === "feed.reset") settle();
    });
    api("/api/cli-jobs").then(page => {
      const found = (page.jobs || []).filter(isPackageJob).find(row => row.cli === cli);
      if (live.current && found) setJob(found);
    }).catch(() => {});
    return () => { off(); clearTimeout(timer); };
  }, [cli, needsWorkspace, load, loadMarket, loadSources]);

  // The marketplace catalog is one vendor call: read it when the tab is opened
  // (and again when a mutation invalidates it), never on a timer.
  useEffect(() => {
    if (hidden || needsWorkspace || tab !== "marketplace") return;
    if (caps.available) loadMarket();
    if (caps.marketplace) loadSources();
  }, [hidden, needsWorkspace, tab, caps.available, caps.marketplace, loadMarket, loadSources]);

  // One request key per user action: the retry after the terminals guard is the
  // same request, which is exactly what the lane's idempotency is for.
  async function send(build, fields) {
    const first = build(fields);
    try {
      return await api(first.path, json(first.method, first.body));
    } catch (ex) {
      // Only the lane's own guard, never a vendor sentence that happens to
      // contain the word: clijob refuses with "... terminal(s) are running."
      if (!/terminals? are running/i.test(ex.message || "")) throw ex;
      const ok = await askConfirm({ title: "Terminals are running", message: ex.message, confirmLabel: "Run anyway", danger: true });
      if (!ok) return null;
      const retry = build({ ...fields, confirmTerminals: true });
      return await api(retry.path, json(retry.method, retry.body));
    }
  }

  function fail(key, ex) {
    // A refusal may carry the command that fixes it (Grok's --trust, a
    // marketplace-declared command): keep it beside the vendor's words so the
    // row offers the line to run instead of a dead end.
    const entry = refusalCommand(ex);
    if (key) setRowError(prev => ({ ...prev, [key]: entry }));
    else setSourceProblem(entry);
  }

  async function install(value, row) {
    const target = String(value || "").trim();
    if (!target || guard || needsWorkspace) return;
    const id = row ? keyOf(row) : "";
    setBusy({ verb: "install", id, label: "Installing…" });
    if (id) setRowError(prev => ({ ...prev, [id]: null })); else setSourceProblem(null);
    try {
      const res = await send(build => paths.install({ ...build, source: target }), { requestKey: newKey() });
      const started = acceptedJob(res);
      if (started) setJob(started);
      if (!row) setSource("");
    } catch (ex) {
      fail(id, ex);
    } finally {
      setBusy(null);
    }
  }

  // checkUpdates asks the server to compare the CLI's catalog with its roster.
  // The answer is the same view with the behind rows marked, so it replaces the
  // report rather than being merged into it.
  const autoChecked = useRef(false);
  const checkUpdates = useCallback(async (refresh = false) => {
    if (guard || needsWorkspace) return;
    setChecking(true);
    try {
      const next = await api(paths.updates(refresh ? { refresh: true } : undefined).path);
      if (next && Array.isArray(next.rows)) {
        setReport(next);
        setCheckedAt(next.checkedAt || "");
      }
    } catch (ex) {
      fail("", ex);
    } finally {
      setChecking(false);
    }
  }, [guard, needsWorkspace, paths]);

  useEffect(() => {
    // One check per mount: the pane shows badges without being asked, and the
    // button is there to check again (the server caches the catalog read).
    if (hidden || autoChecked.current || !caps.update || needsWorkspace) return;
    autoChecked.current = true;
    checkUpdates(false);
  }, [hidden, caps.update, needsWorkspace, checkUpdates]);

  async function toggle(row, on) {
    if (guard || needsWorkspace) return;
    const name = row.name || row.id;
    setBusy({ verb: "toggle", id: keyOf(row), label: on ? "Enabling…" : "Disabling…" });
    setRowError(prev => ({ ...prev, [keyOf(row)]: "" }));
    // The vendor answers a toggle with the new roster, so the row moves now and
    // the answer either confirms it or is rolled back with the reason shown.
    setReport(cur => (cur ? { ...cur, rows: cur.rows.map(r => (keyOf(r) === keyOf(row) ? { ...r, enabled: on } : r)) } : cur));
    try {
      const res = await send(build => paths.toggle({ ...build, name, source: row.source || "" }, on), {});
      // A toggle answers with the whole view: the vendor's own roster, scopes
      // and capabilities, so the pane takes it as the new state.
      if (res && Array.isArray(res.rows)) setReport(res);
    } catch (ex) {
      setReport(cur => (cur ? { ...cur, rows: cur.rows.map(r => (keyOf(r) === keyOf(row) ? { ...r, enabled: row.enabled } : r)) } : cur));
      fail(keyOf(row), ex);
    } finally {
      setBusy(null);
    }
  }

  async function runJob(verb, label, build, fields, id) {
    if (guard || needsWorkspace) return;
    setBusy({ verb, id, label });
    setRowError(prev => ({ ...prev, [id]: "" }));
    try {
      const res = await send(build, fields);
      const started = acceptedJob(res);
      if (started) setJob(started);
    } catch (ex) {
      fail(id, ex);
    } finally {
      setBusy(null);
    }
  }

  async function remove(row) {
    if (guard || needsWorkspace) return;
    const name = row.name || row.id;
    // The confirm names the plugin, and — for the integration PiCode installed
    // itself — the consequence the vendor note spells out.
    const ok = await askConfirm({
      title: "Remove " + name + "?",
      message: cliName + " stops loading this plugin." + (row.managedByPiCode && row.note ? " " + row.note : ""),
      confirmLabel: "Remove",
      danger: true,
    });
    if (!ok) return;
    await runJob("remove", "Removing…", build => paths.remove({ ...build, name, source: row.source || "" }), { requestKey: newKey() }, keyOf(row));
  }

  const update = row => runJob("update", "Updating…", build => paths.update({ ...build, name: row.name || row.id }), { requestKey: newKey() }, keyOf(row));

  async function openInspect(row) {
    if (busy || needsWorkspace) return;
    const name = row.name || row.id;
    setBusy({ verb: "inspect", id: keyOf(row), label: "Reading…" });
    setInspect({ name, output: "", problem: "" });
    try {
      const res = await send(build => paths.inspect({ ...build, name }), {});
      setInspect({ name, output: (res && res.output) || "", problem: "" });
    } catch (ex) {
      setInspect({ name, output: "", problem: ex.message });
    } finally {
      setBusy(null);
    }
  }

  async function addMarketplace(event) {
    event.preventDefault();
    if (guard || needsWorkspace) return;
    const name = addSource.name.trim();
    const value = addSource.source.trim();
    if (!value) return;
    setBusy({ verb: "source", id: "", label: "Adding…" });
    setSourceProblem(null);
    try {
      const res = await send(build => paths.marketplace("add", { ...build, name, source: value }), { requestKey: newKey() });
      const started = acceptedJob(res);
      if (started) setJob(started);
      setAddSource({ name: "", source: "" });
    } catch (ex) {
      fail("", ex);
    } finally {
      setBusy(null);
    }
  }

  async function marketSource(action, name) {
    if (guard || needsWorkspace || !name) return;
    if (action === "remove") {
      const ok = await askConfirm({
        title: "Remove " + name + "?",
        message: "Plugins it installed stay installed. " + cliName + " stops listing this source.",
        confirmLabel: "Remove source",
        danger: true,
      });
      if (!ok) return;
    }
    setBusy({ verb: "source:" + action, id: name, label: action === "remove" ? "Removing…" : "Updating…" });
    setSourceProblem(null);
    try {
      const res = await send(build => paths.marketplace(action, { ...build, name }), action === "remove" ? {} : { requestKey: newKey() });
      const started = acceptedJob(res);
      if (started) setJob(started);
      // A source removal is a fast call and answers with the CLI's own
      // marketplace list, so the chips move without a second read.
      if (res && Array.isArray(res.marketplaces)) setMarketplaces(res.marketplaces);
    } catch (ex) {
      fail("", ex);
    } finally {
      setBusy(null);
    }
  }

  const rows = (report && Array.isArray(report.rows) && report.rows) || [];
  const behind = rows.filter(row => row.updateAvailable).length;
  const scopes = (report && Array.isArray(report.scopes) && report.scopes) || [];
  const currentScope = scopes.find(entry => entry.id === scope) || null;
  const absence = guestPackagesNotes(caps, (report && report.notes) || {});
  const needle = installedFilter.trim().toLowerCase();
  const shown = needle
    ? rows.filter(row => [row.name, row.id, row.source, row.version].some(value => String(value || "").toLowerCase().includes(needle)))
    : rows;
  const marketRows = (market && market.rows) || [];
  const mneedle = marketFilter.trim().toLowerCase();
  const shownMarket = mneedle
    ? marketRows.filter(row => [row.name, row.id, row.source, row.description].some(value => String(value || "").toLowerCase().includes(mneedle)))
    : marketRows;
  // The CLI's own source list when the route (or a marketplace call) answered;
  // an empty one is a real answer ("this CLI lists no sources"), so only a
  // missing answer falls back to the sources this catalog's rows name.
  const sources = marketplaces
    ? marketplaces.map(row => row.name || row.id).filter(Boolean)
    : [...new Set(marketRows.map(row => row.marketplace).filter(Boolean))];
  const verbFor = row => (busy && busy.id && busy.id === keyOf(row) ? busy.verb : "");
  const marketplace = caps.available && tab === "marketplace";

  return (
    <PageFrame id="cli-packages-view" title="Packages" hidden={hidden} embedded>
      {problem ? (
        <div className="cli-notice is-error" role="alert">
          <span>{problem.message}</span>
          <button type="button" className="btn btn-ghost btn-sm" disabled={loading} onClick={() => load({ refresh: true })}>{loading ? "Retrying…" : "Try again"}</button>
          {scope !== "user" ? <button type="button" className="btn btn-ghost btn-sm" onClick={() => onScopeChange("user")}>Use this machine</button> : null}
        </div>
      ) : null}

      {needsWorkspace ? (
        <div className="cli-notice" role="status">
          <span>{"Changing plugins in a project layer needs a workspace."}</span>
          <button type="button" className="btn btn-ghost btn-sm" onClick={() => onScopeChange("user")}>Use this machine</button>
        </div>
      ) : null}

      {!report && loading && !problem && !needsWorkspace ? (
        <div className="gpkg-grid gpkg-skel" role="status" aria-label="Loading plugins">
          {Array.from({ length: 4 }, (_, index) => (
            <div key={"skel-" + index} className="gpkg-card" aria-hidden="true">
              <div className="gpkg-card-head"><div className="skel-line w-50" /></div>
              <div className="skel-line w-90" />
              <div className="skel-line w-70" />
              <div className="gpkg-card-foot"><div className="skel-line w-40" /></div>
            </div>
          ))}
        </div>
      ) : null}

      {report ? (
        <fieldset className="cli-packages-fields" disabled={guard}>
          {scopes.length ? (
            <div className="gpkg-scope" data-align-row data-align-wrap>
              <span className="pkg-scope-label">Plugins go to</span>
              <div className="pkg-scope" role="radiogroup" aria-label="Plugin scope">
                {scopes.map(entry => (
                  <button
                    key={entry.id}
                    type="button"
                    role="radio"
                    className="pkg-scope-btn"
                    aria-checked={scope === entry.id}
                    title={entry.note || undefined}
                    onClick={() => { if (scope !== entry.id) onScopeChange(entry.id); }}
                  >{entry.label}</button>
                ))}
              </div>
            </div>
          ) : null}
          {currentScope && currentScope.note ? <p className="pkg-fine">{currentScope.note}</p> : null}

          {job ? (
            <div className="cli-job-card" role="status" data-state={job.state}>
              <div className="cli-job-head">
                <span className={"cli-job-state is-" + job.state}>{JOB_STATE[job.state] || job.state}</span>
                <span>{(JOB_ACTION[job.action] || job.action) + " · " + cliName}</span>
              </div>
              {job.message ? <p>{job.message}</p> : null}
              {job.command && JOB_ENDED.includes(job.state) && job.state !== "succeeded" ? <Problem entry={{ message: "", command: job.command }} /> : null}
              {job.output ? <details className="cli-job-log"><summary>Output</summary><pre>{job.output}</pre></details> : null}
              {job.state === "interrupted" ? <button type="button" className="btn btn-ghost btn-sm" onClick={() => { setJob(null); load({ refresh: true }); }}>Check result</button> : null}
              {JOB_ENDED.includes(job.state) && job.state !== "interrupted" ? <button type="button" className="btn btn-ghost btn-sm" onClick={() => setJob(null)}>Dismiss</button> : null}
            </div>
          ) : null}

          {caps.install ? (
            <form className="pkg-by-source gpkg-install" data-align-row data-align-wrap noValidate onSubmit={event => { event.preventDefault(); install(source, null); }}>
              <input
                ref={sourceField}
                className="dlg-input"
                value={source}
                onChange={event => setSource(event.target.value)}
                placeholder="npm module, git URL, or name@marketplace"
                aria-label="Plugin source"
                disabled={guard}
              />
              <button type="submit" className="btn btn-primary btn-sm" disabled={guard || !source.trim()}>{busy && busy.verb === "install" && !busy.id ? "Installing…" : "Install"}</button>
            </form>
          ) : null}
          <Problem entry={sourceProblem} />
          <p className="pkg-fine">Plugins run with full access. Only install what you review.</p>

          {/* The blank cells of the capability table, in the vendor's words: one
              sentence per verb this CLI does not have, and never a dead control. */}
          {absence.length ? <ul className="gpkg-notes">{absence.map(text => <li key={text}>{text}</li>)}</ul> : null}

          {caps.available ? (
            <div className="pkg-tabs" role="tablist" aria-label="Plugins">
              <button type="button" role="tab" className="pkg-tab" aria-selected={tab === "installed"} onClick={() => setTab("installed")}>
                Installed{rows.length ? <span className="pkg-tab-count">{rows.length}</span> : null}
              </button>
              <button type="button" role="tab" className="pkg-tab" aria-selected={tab === "marketplace"} onClick={() => setTab("marketplace")}>Marketplace</button>
            </div>
          ) : null}

          {!marketplace ? (
            <>
              {rows.length > 1 ? (
                <div className="pkg-installed-toolbar" data-align-row>
                  <input
                    className="pkg-search"
                    value={installedFilter}
                    onChange={event => setInstalledFilter(event.target.value)}
                    placeholder="Filter installed plugins…"
                    aria-label="Filter installed plugins"
                  />
                  <span className="pkg-count">{shown.length === rows.length ? rows.length + " installed" : shown.length + " of " + rows.length}</span>
                </div>
              ) : null}
              {caps.update ? (
                <div className="gpkg-check" data-align-row>
                  <button type="button" className="btn btn-ghost btn-sm" disabled={guard || checking} onClick={() => checkUpdates(true)}>
                    {checking ? "Checking…" : checkedAt ? "Check again" : "Check for updates"}
                  </button>
                  {checkedAt ? <span className="pkg-fine">{behind ? behind + (behind === 1 ? " can be updated." : " can be updated.") : "Everything is up to date."}</span> : <span className="pkg-fine">Compare these with what the CLI offers.</span>}
                </div>
              ) : null}
              {rows.length === 0 ? (
                <div className="pkg-empty">
                  <p className="pkg-empty-title">Nothing installed.</p>
                  {caps.available ? (
                    <button type="button" className="btn btn-sm" onClick={() => setTab("marketplace")}>Open the Marketplace</button>
                  ) : caps.install ? (
                    <button type="button" className="btn btn-sm" onClick={() => sourceField.current && sourceField.current.focus()}>Install a plugin</button>
                  ) : null}
                </div>
              ) : shown.length === 0 ? (
                <p className="pkg-fine">{"No installed plugin matches “" + installedFilter + "”."} <button type="button" className="btn btn-ghost btn-sm" onClick={() => setInstalledFilter("")}>Clear filter</button></p>
              ) : (
                <ul className="gpkg-grid" role="tabpanel" aria-busy={guard ? "true" : undefined}>
                  {shown.map(row => (
                    <li key={keyOf(row)} className={"gpkg-card" + (row.enabled === false ? " is-off" : "")}>
                      <div className="gpkg-card-head">
                        <span className="gpkg-card-name" title={row.source || row.name || row.id}>{row.name || row.id}</span>
                        {row.version ? <span className="pkg-type">{row.version}</span> : null}
                        {row.updateAvailable && row.latest ? <span className="pkg-type is-update" title={"The CLI's catalog offers " + row.latest}>{"\u2192 " + row.latest}</span> : null}
                      </div>
                      {row.description ? <p className="gpkg-card-desc" title={row.description}>{row.description}</p> : null}
                      {row.note ? <p className="gpkg-row-note" title={row.note}>{row.note}</p> : null}
                      <div className="gpkg-card-meta">
                        {row.scope ? <span className="pkg-type">{row.scope}</span> : null}
                        {row.status ? <span className="gpkg-status">{row.status}</span> : null}
                        {row.managedByPiCode ? <span className="is-picode">Installed by PiCode</span> : null}
                        {row.source ? <span className="gpkg-card-src" title={row.source}>{row.source}</span> : null}
                      </div>
                      <Problem entry={rowError[keyOf(row)]} />
                      <div className="gpkg-card-foot">
                        {caps.toggle ? (
                          <button type="button" className="btn btn-sm" disabled={guard} onClick={() => toggle(row, !row.enabled)}>
                            {verbFor(row) === "toggle" ? busy.label : row.enabled ? "Disable" : "Enable"}
                          </button>
                        ) : null}
                        {rowUpdateState(caps, row, !!checkedAt) === "update" ? (
                          <button type="button" className="btn btn-sm" disabled={guard} title={"Update to " + row.latest} onClick={() => update(row)}>{verbFor(row) === "update" ? busy.label : "Update"}</button>
                        ) : null}
                        {caps.inspect ? (
                          <button type="button" className="btn btn-ghost btn-sm" disabled={guard} onClick={() => openInspect(row)}>{verbFor(row) === "inspect" ? busy.label : "Inspect"}</button>
                        ) : null}
                        {caps.remove ? (
                          <button type="button" className="btn btn-ghost btn-sm" disabled={guard} onClick={() => remove(row)}>{verbFor(row) === "remove" ? busy.label : "Remove"}</button>
                        ) : null}
                      </div>
                    </li>
                  ))}
                </ul>
              )}
            </>
          ) : (
            <>
              {marketProblem ? (
                <div className="cli-notice is-error" role="alert">
                  <span>{marketProblem.message}</span>
                  <button type="button" className="btn btn-ghost btn-sm" disabled={marketLoading} onClick={loadMarket}>{marketLoading ? "Retrying…" : "Try again"}</button>
                </div>
              ) : null}

              {caps.marketplace ? (
                <form className="gpkg-source" data-align-row data-align-wrap noValidate onSubmit={addMarketplace}>
                  <input
                    className="dlg-input"
                    value={addSource.name}
                    onChange={event => setAddSource(form => ({ ...form, name: event.target.value }))}
                    placeholder="Name, if this CLI needs one"
                    aria-label="Marketplace name"
                    disabled={guard}
                  />
                  <input
                    className="dlg-input"
                    value={addSource.source}
                    onChange={event => setAddSource(form => ({ ...form, source: event.target.value }))}
                    placeholder="owner/repo, git URL, or marketplace URL"
                    aria-label="Marketplace source"
                    disabled={guard}
                  />
                  <button type="submit" className="btn btn-sm" disabled={guard || !addSource.source.trim()}>Add source</button>
                </form>
              ) : null}
              {sources.length ? (
                <ul className="gpkg-sources" aria-label={marketplaces ? "Marketplace sources" : "Marketplace sources in this catalog"}>
                  {sources.map(name => (
                    <li key={name} className="gpkg-source-chip">
                      <span className="gpkg-source-name" title={name}>{name}</span>
                      <button type="button" className="btn btn-ghost btn-sm" disabled={guard} onClick={() => marketSource("update", name)}>{busy && busy.id === name && busy.verb === "source:update" ? busy.label : "Update"}</button>
                      <button type="button" className="btn btn-ghost btn-sm" disabled={guard} onClick={() => marketSource("remove", name)}>{busy && busy.id === name && busy.verb === "source:remove" ? busy.label : "Remove"}</button>
                    </li>
                  ))}
                </ul>
              ) : null}
              <Problem entry={sourceProblem} />

              <section className="pkg-toolbar" data-align-row>
                <input
                  className="pkg-search"
                  value={marketFilter}
                  onChange={event => setMarketFilter(event.target.value)}
                  placeholder="Filter this catalog…"
                  aria-label="Filter this catalog"
                />
                <span className="pkg-count">{marketLoading ? (marketRows.length ? "Updating…" : "Loading…") : marketRows.length ? (mneedle ? shownMarket.length + " of " + marketRows.length : marketRows.length + " shown") : null}</span>
              </section>

              {marketLoading && !marketRows.length ? (
                <div className="gpkg-grid gpkg-skel" role="status" aria-label="Loading the marketplace">
                  {Array.from({ length: 4 }, (_, index) => (
                    <div key={"mskel-" + index} className="gpkg-card" aria-hidden="true">
                      <div className="gpkg-card-head"><div className="skel-line w-50" /></div>
                      <div className="skel-line w-90" />
                      <div className="skel-line w-70" />
                      <div className="gpkg-card-foot"><div className="skel-line w-40" /></div>
                    </div>
                  ))}
                </div>
              ) : null}

              {market && market.note ? <p className="pkg-fine">{market.note}</p> : null}

              {market && !marketLoading && marketRows.length === 0 ? (
                <div className="pkg-empty">
                  <p className="pkg-empty-title">This marketplace lists nothing.</p>
                  <button type="button" className="btn btn-sm" onClick={loadMarket}>Read it again</button>
                </div>
              ) : null}

              {marketRows.length && shownMarket.length === 0 ? (
                <p className="pkg-fine">{"Nothing in this catalog matches “" + marketFilter + "”."} <button type="button" className="btn btn-ghost btn-sm" onClick={() => setMarketFilter("")}>Clear filter</button></p>
              ) : null}

              {shownMarket.length ? (
                <ul className="gpkg-grid" role="tabpanel" aria-busy={marketLoading ? "true" : undefined}>
                  {shownMarket.map(row => (
                    <li key={keyOf(row)} className="gpkg-card">
                      <div className="gpkg-card-head">
                        <span className="gpkg-card-name" title={row.source || row.name || row.id}>{row.name || row.id}</span>
                        {row.version ? <span className="pkg-type">{row.version}</span> : null}
                        {row.marketplace ? <span className="pkg-type">{row.marketplace}</span> : null}
                      </div>
                      {row.description ? <p className="gpkg-card-desc" title={row.description}>{row.description}</p> : null}
                      <div className="gpkg-card-meta">
                        {row.source ? <span className="gpkg-card-src" title={row.source}>{row.source}</span> : null}
                      </div>
                      <Problem entry={rowError[keyOf(row)]} />
                      <div className="gpkg-card-foot">
                        {row.installed ? (
                          <span className="gpkg-status is-on">Installed</span>
                        ) : catalogRowAction(caps, row) === "install" ? (
                          <button type="button" className="btn btn-primary btn-sm" disabled={guard} onClick={() => install(row.source, row)}>
                            {verbFor(row) === "install" ? busy.label : "Install"}
                          </button>
                        ) : null}
                      </div>
                    </li>
                  ))}
                </ul>
              ) : null}
            </>
          )}
        </fieldset>
      ) : null}

      {/* The vendor's own inspection text, verbatim: PiCode summarises nothing. */}
      {inspect ? (
        <Dialog.Root open onOpenChange={open => { if (!open) setInspect(null); }}>
          <Dialog.Portal>
            <Dialog.Overlay className="dlg-overlay" />
            <Dialog.Content className="dlg">
              <Dialog.Title className="dlg-title">{(cliName + " · ") + inspect.name}</Dialog.Title>
              {inspect.problem ? (
                <Dialog.Description className="pkg-job-err" role="alert">{inspect.problem}</Dialog.Description>
              ) : inspect.output ? (
                <Dialog.Description asChild><div className="dlg-body"><pre className="gpkg-output">{inspect.output}</pre></div></Dialog.Description>
              ) : (
                <Dialog.Description className="dlg-body">{"Reading " + inspect.name + "…"}</Dialog.Description>
              )}
              <div className="dlg-actions" data-align-row data-align-wrap>
                <Dialog.Close asChild><button type="button" className="btn btn-sm">Close</button></Dialog.Close>
              </div>
            </Dialog.Content>
          </Dialog.Portal>
        </Dialog.Root>
      ) : null}
    </PageFrame>
  );
}
