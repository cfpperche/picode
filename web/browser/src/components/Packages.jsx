import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import * as Dialog from "./ResponsiveDialog.jsx";
import { api, humanizeError } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import {
  packagesApi, packagesNotes, packagesSurface, paneWords, behindFor,
  catalogRowAction, refusalCommand, matchParts, groupInstalledRows, directMutation, laneMutation, anyLaneMutation, paneTabs, rowToggle, rowInspect,
  loadPiPackagesContext, packageContextKey, cliPackagesHash,
} from "@picode/shared/domain/cliPackages.js";
import { terminalCliLabel } from "@picode/shared/domain/terminalCli.js";
import { pkgName } from "@picode/shared/domain/pkgName.js";
import { displayAgentName } from "@picode/shared/domain/tree.js";
import PackagesConfig from "./PackagesConfig.jsx";
import PackageConfigGeneric from "./PackageConfigGeneric.jsx";
import PackageDescribe from "./PackageDescribe.jsx";
import PageFrame from "./PageFrame.jsx";
import PiSpinner from "./PiSpinner.jsx";
import { askConfirm } from "../lib/confirm.js";
import { toast } from "../lib/toast.js";

// One pane, every CLI (ADR-0176). It reads the unified report — what a CLI is
// (its scopes, its capabilities, its catalog) and what it holds — and draws a
// control only where the report declares that control can work: PiCode's own
// gallery where the catalog is the gallery, the vendor's marketplace where the
// vendor has one, Configure where the CLI has config descriptors, a toggle
// where its own verb exists, a per-row Install only where the catalog names the
// spec an install takes.
//
// The two panes this replaces both survive here. The vocabulary follows the
// surface the report declares (`Catalog`): a CLI whose installable list is
// PiCode's own gallery holds *packages* and keeps Pi's words, and one whose
// list is the vendor's holds *plugins* and keeps that pane's. The mutation
// transport follows the driver's own declaration, one verb at a time
// (`Caps.Lane`): a verb on it is a command the durable lane ADR-0087 built
// reserves and runs, and a verb off it is a mutation PiCode performs itself —
// a vendor config file it splices — which answers the CLI's fresh list, so the
// pane shows the transcript and re-reads the report the way Pi's own direct
// calls already do. Pi, whose mutations are all its own calls, declares none.
//
// Both apps carry this file (ADR-0072); the only difference is the modal
// primitive and the pane frame's context line.

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
// A row's key: the CLI's own id, or the name/source it prints when it has
// none. Row errors and pending verbs follow the row, never a shared "".
const keyOf = row => String((row && (row.id || row.name || row.source)) || "");
const newKey = () => (typeof crypto !== "undefined" && crypto.randomUUID ? crypto.randomUUID() : Date.now() + "-" + Math.random());
const json = (method, body) => ({ method, headers: { "Content-Type": "application/json" }, body: body == null ? undefined : JSON.stringify(body) });

// Pi's one-line purpose for packages with a known config editor (ADR-0033
// amendment #3). Unknown packages stay honest: source and scope only.
const PKG_DESC = {
  "pi-roles": "Model roles — route vision, plan or named presets to their own model.",
};

export default function Packages({ hidden, route, catalog, describe = false, onPackageUpdates }) {
  const cli = route.id || "pi";
  const scope = route.scope || "user";
  const workspaceId = route.workspaceId || "";
  const agentId = route.agentId || "";
  const wanted = route.pkg || "";
  const paths = useMemo(() => packagesApi(cli, { workspaceId, agentId, scope }), [cli, workspaceId, agentId, scope]);
  const cliName = terminalCliLabel(cli);
  // The badge read reports to whoever asked for it (the app's own update list).
  // It is held in a ref: the caller hands over a fresh closure on every render,
  // and an effect keyed on it would read the CLI again on each one.
  const updateSink = useRef(onPackageUpdates);
  updateSink.current = onPackageUpdates;

  const [report, setReport] = useState(null);
  const [problem, setProblem] = useState(null);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState("installed");
  const [behind, setBehind] = useState([]);
  const [run, setRun] = useState(null);            // a direct mutation's transcript
  const [job, setJob] = useState(null);            // a lane job
  const [busy, setBusy] = useState(null);
  const [rowError, setRowError] = useState({});
  const [sourceProblem, setSourceProblem] = useState(null);
  const [source, setSource] = useState("");
  const [poolFilter, setPoolFilter] = useState("");
  const [galleryQ, setGalleryQ] = useState("");
  const [hits, setHits] = useState([]);
  const [searching, setSearching] = useState(true);
  const [market, setMarket] = useState(null);
  const [marketProblem, setMarketProblem] = useState(null);
  const [marketLoading, setMarketLoading] = useState(false);
  const [marketFilter, setMarketFilter] = useState("");
  const [addSource, setAddSource] = useState({ name: "", source: "" });
  const [sources, setSources] = useState(null);
  const [inspect, setInspect] = useState(null);
  const live = useRef(true);
  const sequence = useRef(0);
  const sourceField = useRef(null);

  const caps = (report && report.caps) || {};
  const onTheLane = anyLaneMutation(caps);
  const surface = packagesSurface(report);
  const words = paneWords(surface);
  const rows = (report && Array.isArray(report.rows) && report.rows) || [];
  const jobRunning = !!job && !JOB_ENDED.includes(job.state);
  const guard = !!busy || !!run || jobRunning;

  const load = useCallback(async (opts = {}) => {
    const request = ++sequence.current;
    setLoading(true);
    try {
      const next = await api(paths.report(opts).path);
      if (!live.current || request !== sequence.current) return;
      setReport(next);
      setProblem(null);
    } catch (ex) {
      // The CLI's own words, verbatim: a refused or unparsable roster is never
      // an empty list, and an offline CLI keeps the last one on screen.
      if (live.current && request === sequence.current) setProblem({ message: ex.message, status: ex.status });
    } finally {
      if (live.current && request === sequence.current) setLoading(false);
    }
  }, [paths]);

  // The badge read: the CLI's own catalog against its roster. A check that
  // cannot run leaves every row unmarked — never "up to date" (ADR-0167).
  const pullUpdates = useCallback(async (opts = {}) => {
    try {
      const page = await api(paths.updates(opts).path);
      if (!live.current) return;
      const next = page.updates || [];
      setBehind(next);
      updateSink.current?.(next, workspaceId);
    } catch { /* best effort: the rows simply carry no badge */ }
  }, [paths, workspaceId]);

  useEffect(() => {
    live.current = true;
    return () => { live.current = false; sequence.current++; };
  }, []);

  useEffect(() => { if (!hidden) load(); }, [hidden, load]);

  // The workspace and the agent this target names may change under the pane
  // (a rename, a removal, another window): the feed says when to read again.
  useEffect(() => {
    if (!workspaceId && !agentId) return undefined;
    let timer;
    const unsubscribe = subscribeFeed(event => {
      if (/^(agent|workspace)\.(updated|deleted)$|^feed\.(open|reset)$/.test(event.type)) {
        clearTimeout(timer); timer = setTimeout(() => load(), 80);
      }
    });
    return () => { clearTimeout(timer); unsubscribe(); };
  }, [workspaceId, agentId, load]);

  // The lane keeps one active job per CLI, so a reload mid-install finds it
  // there instead of pretending nothing is running. A terminal state is what
  // re-reads the CLI's own roster: success is only success after that read.
  useEffect(() => {
    if (!onTheLane) return undefined;
    let timer;
    const settle = () => { clearTimeout(timer); timer = setTimeout(() => { load(); if (tab === "marketplace") { loadMarket(); loadSources(); } }, 80); };
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
  }, [cli, onTheLane, load, tab]);

  // PiCode's own gallery: the catalog a `gallery` report declares is searched
  // through its own route, once the Marketplace tab is open.
  useEffect(() => {
    if (hidden || surface !== "gallery" || tab !== "marketplace") return undefined;
    const t = setTimeout(async () => {
      setSearching(true);
      try {
        const page = await api(paths.gallery(galleryQ).path);
        setHits(page.hits || []);
      } catch { setHits([]); }
      finally { setSearching(false); }
    }, galleryQ ? 280 : 0);
    return () => clearTimeout(t);
  }, [hidden, surface, tab, galleryQ, paths]);

  useEffect(() => { if (!hidden && caps.update) pullUpdates(); }, [hidden, caps.update, pullUpdates]);

  const loadMarket = useCallback(async () => {
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
  }, [paths]);

  // The CLI's own configured marketplace sources. A 400 is the contract's "this
  // CLI manages no sources" (Hermes), not a failure.
  const loadSources = useCallback(async () => {
    if (!caps.marketplace) return;
    try {
      const next = await api(paths.marketplaces().path);
      if (live.current) setSources(Array.isArray(next.marketplaces) ? next.marketplaces : []);
    } catch (ex) {
      if (live.current) setSources(ex.status === 400 ? [] : null);
    }
  }, [paths, caps.marketplace]);

  // The vendor's catalog is one vendor call: read it when the tab is opened,
  // and again when a mutation invalidates it — never on a timer.
  useEffect(() => {
    if (hidden || surface === "gallery" || tab !== "marketplace") return;
    if (report && report.catalog) loadMarket();
    if (caps.marketplace) loadSources();
  }, [hidden, surface, tab, report, caps.marketplace, loadMarket, loadSources]);

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
    if (!target || guard || !report) return;
    if (scope === "project" && !workspaceId) return;
    if (scope === "agent" && !agentId) return;
    const id = row ? keyOf(row) : "";
    if (id) setRowError(prev => ({ ...prev, [id]: null })); else setSourceProblem(null);
    // A direct mutation answers the CLI's fresh list, so the pane shows the
    // transcript PiCode ran and re-reads the report (Pi's own transport); the
    // agent scope is that kind of write for every CLI, and `directMutation` is
    // where that rule lives (ADR-0176 slice 4).
    if (directMutation(caps, { scope })) {
      setRun({ action: "install", source: target, scope, cwd: scope === "project" ? (report.workspacePath || "") : "", step: 0, error: "", done: false });
      const tick = startJobTick(setRun, 2);
      try {
        const req = paths.direct.install({ source: target, scope });
        await api(req.path, json(req.method, req.body));
        setRun(j => j && { ...j, step: 2, done: true });
        setSource("");
        await load(); await pullUpdates();
        setTimeout(() => setRun(null), 520);
      } catch (ex) {
        setRun(j => j && { ...j, step: 0, error: humanizeError(ex.message || String(ex)) });
      } finally {
        clearInterval(tick);
      }
      return;
    }
    setBusy({ verb: "install", id, label: "Installing…" });
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

  async function toggle(row, on) {
    if (guard) return;
    const name = row.name || row.id;
    setBusy({ verb: "toggle", id: keyOf(row), label: on ? "Enabling…" : "Disabling…" });
    setRowError(prev => ({ ...prev, [keyOf(row)]: "" }));
    // The vendor answers a toggle with the new roster, so the row moves now and
    // the answer either confirms it or is rolled back with the reason shown.
    setReport(cur => (cur ? { ...cur, rows: cur.rows.map(r => (keyOf(r) === keyOf(row) ? { ...r, enabled: on } : r)) } : cur));
    try {
      const res = await send(build => paths.toggle({ ...build, name, source: row.source || "" }, on), {});
      if (res && Array.isArray(res.rows)) setReport(res);
    } catch (ex) {
      setReport(cur => (cur ? { ...cur, rows: cur.rows.map(r => (keyOf(r) === keyOf(row) ? { ...r, enabled: row.enabled } : r)) } : cur));
      fail(keyOf(row), ex);
    } finally {
      setBusy(null);
    }
  }

  async function runJob(verb, label, build, fields, id) {
    if (guard) return;
    setBusy({ verb, id, label });
    setRowError(prev => ({ ...prev, [id]: "" }));
    try {
      const res = await send(build, fields);
      const started = acceptedJob(res);
      if (started) setJob(started);
      // A driver can be mixed per row — an Omp `extensions` entry is a write
      // while the same CLI's plugin rows are lane commands — so a mutation
      // answered with the CLI's fresh list rather than a job is PiCode's own
      // write: the pane keeps the list the answer carried.
      else if (res && Array.isArray(res.rows)) setReport(res);
    } catch (ex) {
      fail(id, ex);
    } finally {
      setBusy(null);
    }
  }

  // A direct update is PiCode's own call and answers the fresh list — including
  // any row in the agent's own list, which is PiCode's store for every CLI
  // (`directMutation`, ADR-0176 slice 4).
  async function updateRow(row, entry) {
    if (guard || !report) return;
    if (directMutation(caps, { row })) {
      const layer = row.scope === "workspace" ? "project" : row.scope === "agent" ? "agent" : "user";
      setRun({ action: "update", source: row.source, scope: layer, cwd: layer === "project" ? (report.workspacePath || "") : "", step: 0, error: "", done: false });
      const tick = startJobTick(setRun, 2);
      try {
        const req = paths.direct.update({ source: row.source, scope: layer });
        await api(req.path, json(req.method, req.body));
        setRun(j => j && { ...j, step: 2, done: true });
        await load(); await pullUpdates();
        setTimeout(() => setRun(null), 520);
      } catch (ex) {
        setRun(j => j && { ...j, step: 0, error: humanizeError(ex.message || String(ex)) });
      } finally {
        clearInterval(tick);
      }
      return;
    }
    await runJob("update", "Updating…", build => paths.update({ ...build, name: row.name || row.id }), { requestKey: newKey() }, keyOf(row));
  }

  async function removeRow(row) {
    if (guard || !report) return;
    const name = row.name || row.id || row.source;
    if (directMutation(caps, { row })) {
      const ok = await askConfirm({
        title: "Remove package",
        message: "Remove " + row.source + " from " + cli + "? This does not uninstall " + cli + " itself.",
        confirmLabel: "Remove",
        danger: true,
      });
      if (!ok) return;
      const layer = row.scope === "workspace" ? "project" : row.scope === "agent" ? "agent" : "user";
      setRun({ action: "remove", source: row.source, scope: layer, cwd: layer === "project" ? (report.workspacePath || "") : "", step: 0, error: "", done: false });
      const tick = startJobTick(setRun, 2);
      try {
        const req = paths.direct.remove({ source: row.source, scope: layer });
        await api(req.path, json(req.method, req.body));
        setRun(j => j && { ...j, step: 2, done: true });
        await load(); await pullUpdates();
        setTimeout(() => setRun(null), 520);
      } catch (ex) {
        setRun(j => j && { ...j, step: 0, error: humanizeError(ex.message || String(ex)) });
      } finally {
        clearInterval(tick);
      }
      return;
    }
    // The confirm names the plugin, and — for the integration PiCode installed
    // itself — the consequence the vendor note spells out.
    const ok = await askConfirm({
      title: "Remove " + name + "?",
      message: cliName + " stops loading this plugin." + (row.managedByPiCode && row.note ? " " + row.note : ""),
      confirmLabel: "Remove",
      danger: true,
    });
    if (!ok) return;
    // A removal the driver performs itself has no argv for the lane: the
    // declaration says so before the request (`Caps.Lane.remove`), which is
    // what lets the pane show the work the way the direct path does instead of
    // waiting for an answer it cannot read. The route answers the CLI's fresh
    // list, and the write has no command line to show.
    if (!laneMutation(caps, "remove")) {
      setRun({ action: "remove", source: row.source, scope, cwd: scope === "project" ? (report.workspacePath || "") : "", written: true, step: 0, error: "", done: false });
      const tick = startJobTick(setRun, 2);
      try {
        const req = paths.remove({ name, source: row.source || "" });
        const res = await send(() => req, { requestKey: newKey() });
        if (res && Array.isArray(res.rows)) setReport(res);
        setRun(j => j && { ...j, step: 2, done: true });
        await load(); await pullUpdates();
        setTimeout(() => setRun(null), 520);
      } catch (ex) {
        setRun(j => j && { ...j, step: 0, error: humanizeError(ex.message || String(ex)) });
      } finally {
        clearInterval(tick);
      }
      return;
    }
    await runJob("remove", "Removing…", build => paths.remove({ ...build, name, source: row.source || "" }), { requestKey: newKey() }, keyOf(row));
  }

  async function openInspect(row) {
    if (busy) return;
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
    if (guard) return;
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
    if (guard || !name) return;
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
      if (res && Array.isArray(res.marketplaces)) setSources(res.marketplaces);
    } catch (ex) {
      fail("", ex);
    } finally {
      setBusy(null);
    }
  }

  // An empty installed list is worth showing the catalog in: one extra read,
  // cached server-side, and only while there is nothing to show.
  useEffect(() => {
    if (hidden || surface === "gallery" || rows.length > 0 || !report || !report.catalog || market || marketLoading) return;
    loadMarket();
  }, [hidden, surface, rows.length, report, market, marketLoading, loadMarket]);

  // PiCode's package pages (ADR-0119) are Pi's config descriptors: a CLI whose
  // report does not declare them refuses the link instead of drawing a roster
  // under a configuration hash.
  if (wanted && report && !caps.config) {
    return (
      <PageFrame id="packages-view" title="Packages" hidden={hidden} embedded>
        <div className="cli-notice" role="status"><span>{cliName + (surface === "gallery" ? " packages" : " plugins") + " have no configuration page."}</span></div>
      </PageFrame>
    );
  }
  if (wanted && caps.config) {
    return <PackageTarget route={route} wanted={wanted} describe={describe} catalog={catalog} hidden={hidden} />;
  }

  const needle = poolFilter.trim().toLowerCase();
  const shown = needle
    ? rows.filter(row => [row.name, row.id, row.source, row.version, row.description].some(value => String(value || "").toLowerCase().includes(needle)))
    : rows;
  // Groups only appear when the list actually mixes origins (the domain
  // decides; one group is the whole list and gets no header).
  const groups = groupInstalledRows(shown);
  const absence = packagesNotes(caps, (report && report.notes) || {});
  const marketRows = (market && market.rows) || [];
  const mneedle = marketFilter.trim().toLowerCase();
  const shownMarket = mneedle
    ? marketRows.filter(row => [row.name, row.id, row.source, row.description, row.marketplace].some(value => String(value || "").toLowerCase().includes(mneedle)))
    : marketRows;
  // The CLI's own source list when the route (or a marketplace call) answered;
  // an empty one is a real answer ("this CLI lists no sources"), so only a
  // missing answer falls back to the sources this catalog's rows name.
  const sourceNames = sources
    ? sources.map(row => row.name || row.id).filter(Boolean)
    : [...new Set(marketRows.map(row => row.marketplace).filter(Boolean))];
  const verbFor = row => (busy && busy.id && busy.id === keyOf(row) ? busy.verb : "");
  const marketplace = surface !== "gallery" && tab === "marketplace";
  const tabs = paneTabs(report, scope);
  // The empty state points at the CLI's own catalog instead of asking the user
  // to guess a name: real entries, and only where the CLI can install.
  const emptyPicks = rows.length === 0
    ? ((market && market.rows) || []).filter(row => catalogRowAction(caps, row) === "install").slice(0, 3)
    : [];

  return (
    <PageFrame id="packages-view" title="Packages" hidden={hidden} embedded>
      {problem ? (
        <div className="cli-notice is-error" role="alert">
          <span>{problem.message}</span>
          <button type="button" className="btn btn-ghost btn-sm" disabled={loading} onClick={() => load({ refresh: true })}>{loading ? "Retrying…" : "Try again"}</button>
          {scope !== "user" ? <button type="button" className="btn btn-ghost btn-sm" onClick={() => goToScope("user")}>{words.fallback}</button> : null}
        </div>
      ) : null}

      {!report && loading && !problem ? (
        <div className="gpkg-grid gpkg-skel" role="status" aria-label={words.loading}>
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
        <fieldset className="cli-packages-fields" disabled={guard || !!problem}>
          {report.scopes.length ? (
            <div className="gpkg-scope" data-align-row data-align-wrap>
              <span className="pkg-scope-label">{words.scopeLabel}</span>
              <div className="pkg-scope" role="radiogroup" aria-label={words.scopeGroup}>
                {report.scopes.map(entry => (
                  <button
                    key={entry.id}
                    type="button"
                    role="radio"
                    className="pkg-scope-btn"
                    aria-checked={scope === entry.vendor || scope === entry.id}
                    title={entry.note || undefined}
                    onClick={() => { const next = entry.vendor || entry.id; if (scope !== next) goToScope(next); }}
                  >{entry.label}</button>
                ))}
              </div>
            </div>
          ) : null}
          {currentScopeNote(report, scope) ? <p className="pkg-fine">{currentScopeNote(report, scope)}</p> : null}

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
                placeholder={words.sourcePlaceholder}
                aria-label={words.sourceLabel}
                disabled={guard}
              />
              <button type="submit" className="btn btn-primary btn-sm" disabled={guard || !source.trim()}>{busy && busy.verb === "install" && !busy.id ? "Installing…" : "Install"}</button>
            </form>
          ) : null}
          <Problem entry={sourceProblem} />

          {caps.isolatedSwitch && agentId ? (
            <label className="pkg-fine" style={{ display: "flex", gap: 8, alignItems: "center" }}>
              <input
                type="checkbox"
                checked={!!report.isolated}
                disabled={guard}
                onChange={async event => {
                  const on = event.target.checked;
                  try {
                    await api("/api/agents/" + agentId, {
                      method: "PATCH",
                      headers: { "Content-Type": "application/json" },
                      body: JSON.stringify({ packagesIsolated: on }),
                    });
                    await load();
                  } catch (ex) { setProblem({ message: ex.message, status: ex.status }); }
                }}
              />
              {words.isolation}
            </label>
          ) : null}

          <p className="pkg-fine">{words.access}</p>

          {/* The blank cells of the capability table, in the CLI's own words:
              one sentence per verb it does not have, never a dead control. */}
          {absence.length ? <ul className="gpkg-notes">{absence.map(text => <li key={text}>{text}</li>)}</ul> : null}

          {tabs ? (
            <div className="pkg-tabs" role="tablist" aria-label="Packages">
              <button type="button" role="tab" className="pkg-tab" aria-selected={tab === "installed"} onClick={() => setTab("installed")}>
                Installed{rows.length ? <span className="pkg-tab-count">{rows.length}</span> : null}
              </button>
              <button type="button" role="tab" className="pkg-tab" aria-selected={tab === "marketplace"} onClick={() => setTab("marketplace")}>Marketplace</button>
            </div>
          ) : null}

          {surface === "gallery" ? (
            <GalleryBody
              tab={tab} report={report} rows={rows} behind={behind} guard={guard} caps={caps}
              configHash={pkg => cliPackagesHash(cli, { ...route, pkg })} describeHash={pkg => describeHash(route, pkg)}
              needle={poolFilter} setNeedle={setPoolFilter} shown={shown} words={words}
              galleryQ={galleryQ} setGalleryQ={setGalleryQ} hits={hits} searching={searching}
              onInstall={install} onUpdate={updateRow} onRemove={removeRow} onOpenMarket={() => setTab("marketplace")} scope={scope}
            />
          ) : (
            <VendorBody
              tab={tab} marketplace={marketplace} rows={rows} shown={shown} groups={groups} behind={behind} guard={guard} caps={caps}
              needle={poolFilter} setNeedle={setPoolFilter} words={words} absence={absence}
              market={market} marketRows={marketRows} shownMarket={shownMarket} marketFilter={marketFilter} setMarketFilter={setMarketFilter}
              marketLoading={marketLoading} marketProblem={marketProblem} loadMarket={loadMarket} cliName={cliName}
              sourceNames={sourceNames} sourcesLoaded={!!sources} capsMarketplace={caps.marketplace}
              addSource={addSource} setAddSource={setAddSource} onAddSource={addMarketplace} onMarketSource={marketSource}
              emptyPicks={emptyPicks} busy={busy} verbFor={verbFor} rowError={rowError} sourceField={sourceField}
              onInstall={install} onToggle={toggle} onUpdate={updateRow} onInspect={openInspect} onRemove={removeRow} onOpenMarket={() => setTab("marketplace")}
            />
          )}
        </fieldset>
      ) : null}

      {run ? <JobOverlay job={run} onClose={() => setRun(null)} /> : null}

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

  function goToScope(next) {
    location.hash = cliPackagesHash(cli, { ...route, scope: next });
  }

  function describeHash(target, pkg) {
    const base = cliPackagesHash(cli, { ...target, pkg });
    return base + (base.includes("?") ? "&" : "?") + "describe=1";
  }
}

// currentScopeNote is the line under the scope radios: the declaration's own
// sentence about the layer the pane is looking at.
function currentScopeNote(report, scope) {
  const entry = (report.scopes || []).find(row => row.vendor === scope || row.id === scope);
  return entry ? entry.note || "" : "";
}

// Highlight shows *why* a card survived the filter: the matched run in the
// accent colour, the rest untouched. Backed by matchParts, so the needle is a
// string and not a pattern.
function Highlight({ text, needle }) {
  const parts = matchParts(text, needle);
  if (!parts.some(part => part.hit)) return parts[0].text;
  return <>{parts.map((part, index) => (part.hit ? <mark key={index} className="pkg-mark">{part.text}</mark> : part.text))}</>;
}

// Problem is one failed action: the CLI's own words, and — when the fix is a
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

// GalleryBody is the surface whose catalog is PiCode's own npm gallery: the
// installed rows are PiCode's packages, with a preview frame, the descriptor
// link, the update badge and a per-row Install on each catalog hit.
function GalleryBody({ tab, report, rows, behind, guard, caps, configHash, describeHash, needle, setNeedle, shown, words, galleryQ, setGalleryQ, hits, searching, onInstall, onUpdate, onRemove, onOpenMarket, scope }) {
  if (tab === "marketplace") {
    return (
      <>
        <section className="pkg-toolbar" data-align-row>
          <input
            className="pkg-search"
            value={galleryQ}
            onChange={event => setGalleryQ(event.target.value)}
            placeholder="Filter packages…"
            aria-label="Search gallery"
          />
          <span className="pkg-count">{searching && !hits.length ? "Loading…" : searching ? "Updating…" : hits.length ? hits.length + " shown" : "No matches"}</span>
          <a className="settings-link" href={report.gallery || "https://pi.dev/packages"} target="_blank" rel="noopener noreferrer">pi.dev ↗</a>
        </section>

        <ul className="pkg-grid" role="tabpanel" aria-busy={searching && !hits.length}>
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
          {hits.map(hit => {
            const on = rows.some(row => row.source === hit.source && row.vendor === scope);
            return (
              <li key={hit.source} className="pkg-card">
                <div className={"pkg-preview" + (hit.image ? " has-media" : "")} aria-hidden="true">
                  <div className="pkg-preview-frame">
                    {hit.image ? <img src={hit.image} alt="" loading="lazy" /> : <><span /><span /><span /></>}
                  </div>
                </div>
                <div className="pkg-card-body">
                  <div className="pkg-card-head">
                    <span className="pkg-card-name">{hit.name}</span>
                    {hit.kind ? <span className="pkg-type">{hit.kind}</span> : null}
                  </div>
                  {hit.description ? <p className="pkg-card-desc">{hit.description}</p> : <p className="pkg-card-desc"> </p>}
                  <div className="pkg-card-meta">
                    {hit.publisher ? <span>{hit.publisher}</span> : null}
                    {hit.downloads ? <span>{fmtDown(hit.downloads)}</span> : null}
                    {hit.updated ? <span>{fmtAge(hit.updated)}</span> : null}
                    {hit.version ? <span>{hit.version}</span> : null}
                  </div>
                  <div className="pkg-card-foot">
                    <code className="pkg-cmd">{cliInstallLine(hit.source)}</code>
                    <button type="button" className="btn btn-primary btn-sm" disabled={guard || on} onClick={() => onInstall(hit.source)}>
                      {on ? "Installed" : "Install"}
                    </button>
                  </div>
                </div>
              </li>
            );
          })}
        </ul>
      </>
    );
  }
  return (
    <>
      {rows.length === 0 ? (
        <div className="pkg-empty">
          <p className="pkg-empty-title">{words.emptyTitle}</p>
          {report.catalog ? <button type="button" className="btn btn-sm" onClick={onOpenMarket}>Open the Marketplace</button> : null}
        </div>
      ) : (
        <>
          {rows.length > 1 ? (
            <div className="pkg-installed-toolbar" data-align-row>
              <input
                className="pkg-search"
                value={needle}
                onChange={event => setNeedle(event.target.value)}
                placeholder={words.installedFilter}
                aria-label={words.installedFilterLabel}
              />
              <span className="pkg-count">{shown.length === rows.length ? rows.length + " installed" : shown.length + " of " + rows.length}</span>
            </div>
          ) : null}
          {shown.length === 0 ? (
            <p className="pkg-fine">{words.noMatch(needle)} <button type="button" className="btn btn-ghost btn-sm" onClick={() => setNeedle("")}>Clear filter</button></p>
          ) : (
            <ul className="pkg-grid" role="tabpanel">
              {shown.map(row => (
                <li key={row.scope + ":" + row.source} className="pkg-card pkg-card-installed">
                  <div className="pkg-preview" aria-hidden="true">
                    <div className="pkg-preview-frame"><span /><span /><span /></div>
                  </div>
                  <div className="pkg-card-body">
                    <div className="pkg-card-head">
                      <span className="pkg-card-name" title={row.source}>{pkgName(row.source)}</span>
                      <span className="pkg-type">{layerLabel(row, report)}</span>
                    </div>
                    <p className="pkg-card-desc pkg-src" title={PKG_DESC[pkgName(row.source)] || row.source}>{PKG_DESC[pkgName(row.source)] || (row.kind === "path" && row.installedPath ? row.installedPath : row.source)}</p>
                    <div className="pkg-card-meta">
                      {row.kind ? <span>{row.kind}</span> : null}
                      {behindFor(behind, row) && behindFor(behind, row).current ? <span>{behindFor(behind, row).current}</span> : null}
                      {behindFor(behind, row) && behindFor(behind, row).latest ? <span className="pkg-behind">{behindFor(behind, row).latest} available</span> : null}
                      {row.installedPath ? <span className="pkg-path" title={row.installedPath}>{row.installedPath}</span> : null}
                    </div>
                    <div className="pkg-card-foot">
                      {row.configKind && caps.config ? (
                        <a className="btn btn-sm" href={configHash(row.configKind === "roles" ? "pi-roles" : row.configKind)}>Configure</a>
                      ) : caps.config ? (
                        <a className="btn btn-ghost btn-sm" href={describeHash(row.name)}>Describe config…</a>
                      ) : null}
                      {behindFor(behind, row) ? (
                        <button type="button" className="btn btn-primary btn-sm" disabled={guard} title={behindTitle(behindFor(behind, row))} onClick={() => onUpdate(row, behindFor(behind, row))}>Update</button>
                      ) : null}
                      <span className="pkg-foot-spacer" />
                      <button type="button" className="btn btn-ghost btn-sm" disabled={guard} onClick={() => onRemove(row)}>Remove</button>
                    </div>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </>
      )}
    </>
  );
}

// VendorBody is the surface whose catalog is the vendor's own: the roster the
// CLI prints, its catalog, and the sources it keeps — every control gated by
// the capability the report declares.
function VendorBody({
  tab, marketplace, rows, shown, groups, behind, guard, caps, needle, setNeedle, words, absence,
  market, marketRows, shownMarket, marketFilter, setMarketFilter, marketLoading, marketProblem, loadMarket, cliName,
  sourceNames, sourcesLoaded, capsMarketplace, addSource, setAddSource, onAddSource, onMarketSource,
  emptyPicks, busy, verbFor, rowError, sourceField, onInstall, onToggle, onUpdate, onInspect, onRemove, onOpenMarket,
}) {
  const grid = list => (
    <ul className="gpkg-grid" aria-busy={guard ? "true" : undefined}>
      {list.map(row => (
        <li key={keyOf(row)} className={"gpkg-card" + (row.enabled === false ? " is-off" : "")}>
          <div className="gpkg-card-head">
            <span className="gpkg-card-name" title={row.source || row.name || row.id}><Highlight text={row.name || row.id} needle={needle} /></span>
            {row.version ? <span className="pkg-type">{row.version}</span> : null}
            {behindFor(behind, row) ? <span className="pkg-type is-update" title={"The CLI's catalog offers " + behindFor(behind, row).latest}>{"\u2192 " + behindFor(behind, row).latest}</span> : null}
          </div>
          {row.description ? <p className="gpkg-card-desc" title={row.description}><Highlight text={row.description} needle={needle} /></p> : null}
          {row.note ? <p className="gpkg-row-note" title={row.note}>{row.note}</p> : null}
          <div className="gpkg-card-meta">
            {row.vendor ? <span className="pkg-type">{row.vendor}</span> : null}
            {row.status ? <span className="gpkg-status">{row.status}</span> : null}
            {row.managedByPiCode ? <span className="is-picode">Installed by PiCode</span> : null}
            {row.source ? <span className="gpkg-card-src" title={row.source}>{row.source}</span> : null}
          </div>
          <Problem entry={rowError[keyOf(row)]} />
          <div className="gpkg-card-foot">
            {rowToggle(caps, row) ? (
              <button type="button" className="btn btn-sm" disabled={guard} onClick={() => onToggle(row, !row.enabled)}>
                {verbFor(row) === "toggle" ? busy.label : row.enabled ? "Disable" : "Enable"}
              </button>
            ) : null}
            {behindFor(behind, row) ? (
              <button type="button" className="btn btn-sm" disabled={guard} title={"Update to " + behindFor(behind, row).latest} onClick={() => onUpdate(row)}>{verbFor(row) === "update" ? busy.label : "Update"}</button>
            ) : null}
            {rowInspect(caps, row) ? (
              <button type="button" className="btn btn-ghost btn-sm" disabled={guard} onClick={() => onInspect(row)}>{verbFor(row) === "inspect" ? busy.label : "Inspect"}</button>
            ) : null}
            {caps.remove ? (
              <button type="button" className="btn btn-ghost btn-sm" disabled={guard} onClick={() => onRemove(row)}>{verbFor(row) === "remove" ? busy.label : "Remove"}</button>
            ) : null}
          </div>
        </li>
      ))}
    </ul>
  );

  if (marketplace) {
    return (
      <>
        {marketProblem ? (
          <div className="cli-notice is-error" role="alert">
            <span>{marketProblem.message}</span>
            <button type="button" className="btn btn-ghost btn-sm" disabled={marketLoading} onClick={loadMarket}>{marketLoading ? "Retrying…" : "Try again"}</button>
          </div>
        ) : null}

        {capsMarketplace ? (
          <form className="gpkg-source" data-align-row data-align-wrap noValidate onSubmit={onAddSource}>
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
        {sourceNames.length ? (
          <ul className="gpkg-sources" aria-label={sourcesLoaded ? "Marketplace sources" : "Marketplace sources in this catalog"}>
            {sourceNames.map(name => (
              <li key={name} className="gpkg-source-chip">
                <button
                  type="button"
                  className="gpkg-source-name"
                  aria-pressed={marketFilter === name}
                  title={"Show only " + name + "'s plugins"}
                  onClick={() => setMarketFilter(marketFilter === name ? "" : name)}
                >{name}</button>
                <button type="button" className="btn btn-ghost btn-sm" disabled={guard} onClick={() => onMarketSource("update", name)}>{busy && busy.id === name && busy.verb === "source:update" ? busy.label : "Update"}</button>
                <button type="button" className="btn btn-ghost btn-sm" disabled={guard} onClick={() => onMarketSource("remove", name)}>{busy && busy.id === name && busy.verb === "source:remove" ? busy.label : "Remove"}</button>
              </li>
            ))}
          </ul>
        ) : null}

        {/* The catalog runs to hundreds of rows: the filter rides along. */}
        <div className="gpkg-sticky">
          <section className="pkg-toolbar" data-align-row>
            <input
              className="pkg-search"
              value={marketFilter}
              onChange={event => setMarketFilter(event.target.value)}
              placeholder="Filter this catalog…"
              aria-label="Filter this catalog"
            />
            <span className="pkg-count">{marketLoading ? (marketRows.length ? "Updating…" : "Loading…") : marketRows.length ? (marketFilter.trim() ? shownMarket.length + " of " + marketRows.length : marketRows.length + " shown") : null}</span>
          </section>
        </div>

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
                  <span className="gpkg-card-name" title={row.source || row.name || row.id}><Highlight text={row.name || row.id} needle={marketFilter} /></span>
                  {row.version ? <span className="pkg-type">{row.version}</span> : null}
                  {row.marketplace ? (
                    <button
                      type="button"
                      className="pkg-type is-filter"
                      aria-pressed={marketFilter === row.marketplace}
                      title={"Show only " + row.marketplace + "'s plugins"}
                      onClick={() => setMarketFilter(marketFilter === row.marketplace ? "" : row.marketplace)}
                    >{row.marketplace}</button>
                  ) : null}
                </div>
                {row.description ? <p className="gpkg-card-desc" title={row.description}><Highlight text={row.description} needle={marketFilter} /></p> : null}
                <div className="gpkg-card-meta">
                  {row.source ? <span className="gpkg-card-src" title={row.source}>{row.source}</span> : null}
                </div>
                <Problem entry={rowError[keyOf(row)]} />
                <div className="gpkg-card-foot">
                  {row.installed ? (
                    <span className="gpkg-status is-on">Installed</span>
                  ) : catalogRowAction(caps, row) === "install" ? (
                    <button type="button" className="btn btn-primary btn-sm" disabled={guard} onClick={() => onInstall(row.source, row)}>
                      {verbFor(row) === "install" ? busy.label : "Install"}
                    </button>
                  ) : null}
                </div>
              </li>
            ))}
          </ul>
        ) : null}
      </>
    );
  }

  return (
    <>
      <div className="gpkg-sticky">
        {rows.length > 1 ? (
          <div className="pkg-installed-toolbar" data-align-row>
            <input
              className="pkg-search"
              value={needle}
              onChange={event => setNeedle(event.target.value)}
              placeholder={words.installedFilter}
              aria-label={words.installedFilterLabel}
            />
            <span className="pkg-count">{shown.length === rows.length ? rows.length + " installed" : shown.length + " of " + rows.length}</span>
          </div>
        ) : null}
      </div>

      {rows.length === 0 ? (
        <div className="pkg-empty">
          <p className="pkg-empty-title">{words.emptyTitle}</p>
          {emptyPicks.length ? (
            <>
              <p className="pkg-fine">{cliName + " offers these — install one, or name any source above."}</p>
              <ul className="gpkg-picks">
                {emptyPicks.map(row => (
                  <li key={keyOf(row)} className="gpkg-pick">
                    <span className="gpkg-pick-name" title={row.source || row.name}>{row.name || row.id}</span>
                    {row.version ? <span className="pkg-type">{row.version}</span> : null}
                    <button type="button" className="btn btn-sm" disabled={guard} onClick={() => onInstall(row.source, row)}>
                      {verbFor(row) === "install" ? busy.label : "Install"}
                    </button>
                  </li>
                ))}
              </ul>
              {market ? <button type="button" className="btn btn-ghost btn-sm" onClick={onOpenMarket}>See the whole catalog</button> : null}
            </>
          ) : market ? (
            <button type="button" className="btn btn-sm" onClick={onOpenMarket}>Open the Marketplace</button>
          ) : caps.install ? (
            <button type="button" className="btn btn-sm" onClick={() => sourceField.current && sourceField.current.focus()}>Install a plugin</button>
          ) : null}
        </div>
      ) : shown.length === 0 ? (
        <p className="pkg-fine">{words.noMatch(needle)} <button type="button" className="btn btn-ghost btn-sm" onClick={() => setNeedle("")}>Clear filter</button></p>
      ) : groups.length ? groups.map(group => (
        <section key={group.key} className="gpkg-group">
          <h4 className="gpkg-group-head">
            <span>{group.label}</span>
            <span className="gpkg-group-count">{group.rows.length}</span>
          </h4>
          {grid(group.rows)}
        </section>
      )) : grid(shown)}
    </>
  );
}

// PackageTarget resolves PiCode's own workspace and agent objects for a package
// configuration page (ADR-0119): the editors take a file target, and a target
// that changed under an open draft is refused rather than written blind.
function PackageTarget({ route, wanted, describe, catalog, hidden }) {
  const [context, setContext] = useState(null);
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(true);
  const [retry, setRetry] = useState(0);
  const live = useRef(true);
  const targetKey = useRef(null);
  function validateTarget(next) {
    const key = packageContextKey(next);
    if (targetKey.current !== null && targetKey.current !== key) throw Object.assign(new Error("The package location changed. Reload before editing."), { status: 409 });
    targetKey.current = key;
  }
  const blocked = useRef(false);
  blocked.current = !!error;
  useEffect(() => { live.current = true; return () => { live.current = false; }; }, []);
  useEffect(() => {
    let active = true;
    setLoading(true);
    loadPiPackagesContext(route, api).then(next => {
      if (active) { validateTarget(next); setContext(next); setError(null); }
    }).catch(err => {
      if (active) { setError(err); if (err.status === 404) setContext(null); }
    }).finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [route.workspaceId, route.agentId, route.scope, retry]);
  useEffect(() => {
    if (!route.workspaceId && !route.agentId) return undefined;
    let timer;
    const unsubscribe = subscribeFeed(event => {
      if (/^(agent|workspace)\.(updated|deleted)$|^feed\.(open|reset)$/.test(event.type)) {
        clearTimeout(timer); timer = setTimeout(() => setRetry(value => value + 1), 80);
      }
    });
    return () => { clearTimeout(timer); unsubscribe(); };
  }, [route.workspaceId, route.agentId]);
  async function beforeMutation() {
    if (!live.current || blocked.current) throw new Error("Reload the package context before making changes.");
    try {
      const next = await loadPiPackagesContext(route, api);
      validateTarget(next);
      if (!live.current || blocked.current) throw new Error("The package context changed. Try again.");
    } catch (err) {
      if (live.current) { setError(err); if (err.status === 404) setContext(null); }
      throw err;
    }
  }
  async function recover() {
    if (error?.status !== 409) { setRetry(value => value + 1); return; }
    if (await askConfirm({ title: "Reload package context", message: "Discard local changes and reload the package location?", confirmLabel: "Reload" })) location.reload();
  }
  const notice = error ? (
    <div className="cli-notice is-error" role="alert">
      <span>{error.message}</span>
      <button type="button" className="btn btn-ghost btn-sm" disabled={loading} onClick={recover}>{loading ? "Retrying…" : error.status === 409 ? "Reload page" : "Try again"}</button>
      <a className="btn btn-ghost btn-sm" href={cliPackagesHash(route.id)}>Machine packages</a>
    </div>
  ) : null;
  if (!context) return <PageFrame id="packages-view" title="Packages" hidden={hidden} embedded>{notice || <div className="cli-loading" aria-label="Loading package context"><div /><div /><div /></div>}</PageFrame>;
  const workspace = context.workspace, agent = context.agent;
  const listHash = cliPackagesHash(route.id, { ...route, pkg: "" });
  const props = {
    embedded: true, hidden: false,
    workspaceId: workspace?.id || "", workspaceName: workspace?.name || "", workspacePath: workspace?.path || "",
    agentId: agent?.id || "", agentName: agent ? displayAgentName(agent, workspace) : "",
    beforeMutation,
  };
  return (
    <PageFrame id="packages-view" title="Packages" hidden={hidden} embedded>
      {notice}
      <fieldset className="cli-packages-fields" disabled={!!error}>
        {describe ? <PackageDescribe {...props} pkg={wanted} backHash={listHash} configHashFor={p => cliPackagesHash(route.id, { ...route, pkg: p })} /> :
          wanted === "pi-roles" ?
            <PackagesConfig {...props} pkg={wanted} catalog={catalog} backHash={listHash} initialScope={route.scope === "agent" ? "agent" : "workspace"} onScopeChange={scope => { location.hash = cliPackagesHash(route.id, { ...route, scope: scope === "agent" ? "agent" : "project" }); }} />
            : <PackageConfigGeneric {...props} pkg={wanted} backHash={listHash} describeHash={p => cliPackagesHash(route.id, { ...route, pkg: p }) + "&describe=1"} listHash={listHash} />}
      </fieldset>
    </PageFrame>
  );
}

// layerLabel names the layer a row lives in, in the pane's own words: the
// caller's name for the workspace or the agent when the report carries one,
// the class's bare word otherwise.
function layerLabel(row, report) {
  if (row.scope === "workspace") return report.workspaceName || "workspace";
  if (row.scope === "agent") return report.agentName || "agent";
  return "global";
}

function behindTitle(entry) {
  return entry && entry.current && entry.latest ? entry.current + " → " + entry.latest : undefined;
}

function startJobTick(setJob, stepCount) {
  return setInterval(() => {
    setJob(j => {
      if (!j || j.error || j.done) return j;
      if (j.step < stepCount - 1) return { ...j, step: j.step + 1 };
      return j;
    });
  }, 480);
}

// jobSteps is the transcript a direct mutation shows: the exact command PiCode
// ran, and the read that follows it. The line is the driver's own argv
// (`pipkg.MutateArgs`), spelled here because this transport answers the fresh
// list rather than a lane job carrying its command. A mutation the driver
// performs itself — a splice of the CLI's own config file — ran no command at
// all (`job.written`), so its step is the action and its subject, never a line
// PiCode did not run.
function jobSteps(job) {
  if (job.action === "update") {
    return [
      { id: "run", label: "Update " + job.source },
      { id: "list", label: "Reload installed packages" },
    ];
  }
  const bin = job.action === "remove" ? "remove" : "install";
  if (job.scope === "agent") {
    return [
      { id: "run", label: (bin === "remove" ? "Drop from this agent: " : "Attach to this agent: ") + job.source },
      { id: "list", label: "Reload. Takes effect the next time this agent starts." },
    ];
  }
  if (job.written) {
    return [
      { id: "run", label: (bin === "remove" ? "Remove " : "Install ") + job.source },
      { id: "list", label: "Reload installed packages" },
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

// JobOverlay is the transcript of one direct mutation: the steps above, the
// vendor's words on a failure, and — for a mutation that hands the CLI a
// command — the line about how long that command may take. A mutation the
// driver performs itself (`job.written`) has no such command, so it carries no
// such sentence either.
function JobOverlay({ job, onClose }) {
  const steps = jobSteps(job);
  const title = job.action === "remove" ? "Removing package" : job.action === "update" ? "Updating package" : "Installing package";
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
        ) : job.written ? null : (
          <p className="pkg-fine">Stays here until pi finishes. npm can take a minute.</p>
        )}
      </div>
    </div>
  );
}

// cliInstallLine is the line a gallery card shows for a hit: PiCode's own
// install verb for its own gallery, exactly as the CLI takes it.
function cliInstallLine(source) {
  return "pi install " + source;
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
