import { useEffect, useMemo, useRef, useState } from "react";
import { api } from "@picode/shared/client/api.js";
import { formatMoney } from "@picode/shared/domain/providerUsage.js";
import {
  deltaPercent, fleetStats, compareLabel, formatTokens, percent, folderLabel,
  measured, coverageNote, billingBadge, billingTitle, formatDuration, formatLines,
  costPerTurn, costPerLine, signalState, spendState, NOT_REPORTED, PARTIAL,
  FLEET_ORDER, FLEET_LABELS, FLEET_HINTS, FLEET_NEEDS_YOU, FLEET_WORKING,
} from "@picode/shared/domain/dashboardStats.js";
import { readDashboardRange, writeDashboardRange, readDashboardStats, writeDashboardStats } from "../lib/openTabs.js";
import { termTabId } from "../lib/routes.js";
import { relTime, absTime } from "@picode/shared/domain/relTime.js";
import { shortModel } from "@picode/shared/domain/chip.js";
import { terminalCliLabel, terminalCliMark } from "@picode/shared/domain/terminalCli.js";
import StatTile from "./StatTile.jsx";
import RankedBars from "./RankedBars.jsx";
import DailyChart from "./DailyChart.jsx";
import TokenBar from "./TokenBar.jsx";
import TopSessions from "./TopSessions.jsx";
import DateRangePicker from "./DateRangePicker.jsx";
import CoveragePanel from "./CoveragePanel.jsx";
import LimitBars from "./LimitBars.jsx";
import { ProviderFace } from "./ProviderFaces.jsx";
import { IconReload } from "./Icons.jsx";

const REFRESH_MS = 60_000;
const TICK_MS = 30_000;

// The landing surface for "no tab open" once real data exists (ADR-0041,
// v2 breakdowns in ADR-0042) — spend/activity/sessions/fleet metrics with
// the period's breakdown by model, workspace, tokens, tools, reliability
// and costliest sessions. Not another list of the workspaces/agents the
// sidebar already shows. App.jsx only mounts this when
// workspaces+freeAgents+terminals adds up to more than zero; the true
// empty-environment card is untouched and lives beside it.
export default function DashboardView({ workspaces, freeAgents, terminals, workingIds, waitingId, onOpen, inboxWaiting = 0, onOpenApp }) {
  const [range, setRange] = useState(() => readDashboardRange());
  const [metric, setMetric] = useState("cost");
  // F1: paint the last good payload instantly while the fresh fetch runs
  // underneath. The range key is read lazily once; a range switch still
  // keeps the previous range's numbers (keep=true) until its own fetch
  // lands, so the well never blanks on navigation.
  const [stats, setStats] = useState(() => readDashboardStats(readDashboardRange())?.data || null);
  const [busy, setBusy] = useState(false);
  const [loadErr, setLoadErr] = useState("");
  const [fetchedAt, setFetchedAt] = useState(() => readDashboardStats(readDashboardRange())?.at || null);
  const [fleetOpen, setFleetOpen] = useState(false);
  const [, setTick] = useState(0);
  const mounted = useRef(false);
  const inflight = useRef(null);

  useEffect(() => {
    // First mount with a cached payload is a refresh, not a cold load:
    // keep the cached numbers visible (dimmed) while revalidating.
    const keep = mounted.current || stats !== null;
    mounted.current = true;
    return load(keep);
  }, [range]);

  // Auto-refresh while the dashboard is on screen — the server answers a
  // poll from its fingerprint cache unless a session actually changed, so
  // this is cheap — and a 30s tick so "updated 2m" is never a lie. Both
  // pause while the tab is hidden.
  useEffect(() => {
    let refresh = null;
    let tick = null;
    const start = () => {
      if (refresh) return;
      refresh = setInterval(() => load(true), REFRESH_MS);
      tick = setInterval(() => setTick((t) => t + 1), TICK_MS);
    };
    const stop = () => {
      if (refresh) clearInterval(refresh);
      if (tick) clearInterval(tick);
      refresh = tick = null;
    };
    const onVis = () => {
      if (document.hidden) stop();
      else { start(); load(true); }
    };
    start();
    document.addEventListener("visibilitychange", onVis);
    return () => { stop(); document.removeEventListener("visibilitychange", onVis); };
  }, [range]);

  function load(keep) {
    if (inflight.current) inflight.current();
    let cancelled = false;
    inflight.current = () => { cancelled = true; };
    if (!keep) {
      setStats(null);
      setLoadErr("");
    }
    setBusy(true);
    api("/api/sessions/stats?range=" + encodeURIComponent(range))
      .then((rep) => {
        if (cancelled) return;
        setStats(rep);
        setLoadErr("");
        const now = new Date().toISOString();
        setFetchedAt(now);
        writeDashboardStats(range, rep);
      })
      .catch(() => {
        if (cancelled) return;
        setLoadErr(keep ? "Couldn't refresh — showing the last loaded numbers." : "Couldn't load spend data.");
        if (!keep) setStats(null);
      })
      .finally(() => { if (!cancelled) setBusy(false); });
    return () => { cancelled = true; };
  }

  function onRange(next) {
    setRange(next);
    writeDashboardRange(next);
  }

  // F3: the 30s "updated Xm" tick re-renders, but these derivations only
  // recompute when their inputs change — otherwise every tick re-maps
  // 40+ model rows, 100+ workspace rows and the fleet sort for nothing.
  const fleet = useMemo(
    () => fleetStats(workspaces, freeAgents, terminals, { workingIds, waitingId }),
    [workspaces, freeAgents, terminals, workingIds, waitingId],
  );
  // What is waiting on the reader, counted without double counting: questions
  // in the Inbox (server truth, machine-wide — an agent's ask becomes an item
  // there) plus agent-CLI terminals whose own hooks said `needs-you`. The
  // selected agent's `waiting` is deliberately not added: that agent asked
  // through the Inbox, so adding it would count one block twice.
  const waitingTerminals = useMemo(
    () => fleet.units.filter((u) => u.kind === "terminal" && u.state === FLEET_NEEDS_YOU),
    [fleet],
  );
  const questions = Number(inboxWaiting) || 0;
  const needsYou = questions + waitingTerminals.length;
  const firstLoad = stats === null;
  const coverage = useMemo(() => (firstLoad ? [] : (stats.coverage || [])), [firstLoad, stats]);
  const byCli = useMemo(() => (firstLoad ? [] : (stats.byCli || [])), [firstLoad, stats]);
  const cmp = compareLabel(range);
  const sessionsDelta = !firstLoad && stats.prior ? stats.current.sessions - stats.prior.sessions : null;
  const turns = firstLoad ? null : stats.turns;
  const impact = firstLoad ? null : stats.impact;
  const timing = firstLoad ? null : stats.timing;
  const desktop = firstLoad ? null : stats.desktop;
  // An empty panel has two different reasons — nothing ran, or nothing that
  // ran records this — and saying the wrong one sends the reader looking
  // for a CLI feature they cannot fix.
  const idle = firstLoad ? false : stats.current.messages === 0;
  const nothingRan = "No agent activity in this period.";
  // Every panel whose number does not cover every active CLI says so. A
  // footnote that always shows is noise; one that shows only when something
  // is missing is a fact (ADR-0097).
  const costNote = firstLoad ? "" : coverageNote(coverage, byCli, "cost");
  const spend = firstLoad ? null : spendState(byCli);
  const impactNote = firstLoad ? "" : coverageNote(coverage, byCli, "impact");
  const timingNote = firstLoad ? "" : coverageNote(coverage, byCli, "timing");
  const toolsNote = firstLoad ? "" : coverageNote(coverage, byCli, "tools");
  const errorsNote = firstLoad ? "" : coverageNote(coverage, byCli, "errors");
  // Quota windows are the one panel only some CLIs fill (Codex today), so it
  // owes the same "Covers X only." footnote the others do — the reader must
  // not read "1 window" as "the machine has one limit".
  const limitsNote = firstLoad ? "" : coverageNote(coverage, byCli, "limits");

  // A zero-cost "unknown" model row is pi bookkeeping (a turn with no
  // usage block), not a lever anyone can pull — keep the ranking honest
  // by dropping only that.
  const modelItems = useMemo(() => (firstLoad ? [] : stats.byModel.filter((m) => !(m.model === "unknown" && m.cost === 0)).map((m) => ({
    key: (m.cli || "") + "/" + m.provider + "/" + m.model,
    label: shortModel(m.model),
    sub: m.provider,
    face: <ProviderFace id={m.provider} />,
    // The same model reached through two CLIs is two rows, so the row has
    // to say which one it is.
    badge: m.cli ? terminalCliMark(m.cli) : "",
    badgeTitle: m.cli ? cliLabelOf(coverage, m.cli) : "",
    value: m.cost,
    // A model row from a CLI that records no price shows why, not $0.00.
    unmeasured: !measured(signalState(coverage, m.cli, "cost")) && !m.cost,
    display: measured(signalState(coverage, m.cli, "cost")) || m.cost ? money(m.cost) : "not priced",
    title: (m.cli ? cliLabelOf(coverage, m.cli) + " · " : "") + m.provider + " · " + m.model + " · " + m.messages.toLocaleString() + " turns",
  }))), [firstLoad, stats, coverage]);

  // The pivot this release exists for: which CLI the money and the work went
  // to. A CLI that records no price shows why instead of a $0.00 that would
  // read as "free".
  // A CLI with no activity in this window contributed nothing to rank. It
  // still owes the coverage panel a row — that is where "installed and
  // silent" belongs — but padding the spend list with $0.00 bars would say
  // six CLIs were free rather than that none of them ran.
  const cliItems = useMemo(() => (firstLoad ? [] : stats.byCli.filter((c) => c.messages > 0).map((c) => {
    const priced = measured(c.costState);
    return {
      key: c.cli,
      label: c.label || c.cli,
      badge: billingBadge(c.billing),
      badgeTitle: billingTitle(c.billing),
      value: priced ? c.cost : 0,
      unmeasured: !priced,
      display: priced ? money(c.cost) : "not priced",
      title: (c.label || c.cli) + " · " + c.messages.toLocaleString() + " messages · " +
        c.sessions.toLocaleString() + (c.sessions === 1 ? " session" : " sessions") +
        (priced ? "" : " · this CLI records no cost"),
    };
  })), [firstLoad, stats]);
  const workspaceItems = useMemo(() => (firstLoad ? [] : stats.byWorkspace.map((w) => ({
    key: w.cwd,
    label: w.workspace || folderLabel(w.cwd) || w.cwd,
    value: w.cost,
    // A folder's cost mixes CLIs, so it borrows the window's verdict: when
    // nothing that ran was priced, the row says so instead of "$0.00".
    unmeasured: spend === NOT_REPORTED && !w.cost,
    display: spend === NOT_REPORTED && !w.cost ? "not priced" : money(w.cost),
    title: w.cwd + " · " + w.sessions + (w.sessions === 1 ? " session" : " sessions") + (w.workspace ? "" : " · outside your workspaces"),
  }))), [firstLoad, stats, spend]);
  // Tool names are not normalised across CLIs — "Bash", "bash" and "shell"
  // are three vendors' words — so the mark disambiguates rather than a
  // merge asserting an equivalence we cannot back.
  const toolItems = useMemo(() => (firstLoad ? [] : stats.tools.map((t) => ({
    key: (t.cli || "") + ":" + t.name,
    label: t.name,
    badge: t.cli ? terminalCliMark(t.cli) : "",
    badgeTitle: t.cli ? cliLabelOf(coverage, t.cli) : "",
    value: t.calls,
    display: t.calls.toLocaleString(),
  }))), [firstLoad, stats, coverage]);

  return (
    <div className="dashboard-view">
      <div className="dash-inner">
      <div className="dash-head">
        <DateRangePicker value={range} onChange={onRange} />
        <span className="dash-asof-wrap">
          {fetchedAt ? <span className="dash-asof" title={absTime(fetchedAt)}>updated {relTime(fetchedAt)}</span> : null}
          <button type="button" className={"ws-icon-btn dash-refresh" + (busy ? " is-busy" : "")} title="Refresh now" aria-label="Refresh now" disabled={busy} onClick={() => load(true)}>
            <IconReload size={14} />
          </button>
        </span>
      </div>

      {loadErr && firstLoad ? (
        <p className="file-pane-msg">{loadErr} <button type="button" className="btn btn-sm" onClick={() => load(false)}>Retry</button></p>
      ) : (
        <div className="dash-body" style={{ opacity: busy && !firstLoad ? 0.6 : 1 }}>
          {loadErr ? (
            <p className="dash-refresh-err">{loadErr} <button type="button" className="btn-link" onClick={() => load(true)}>Retry</button></p>
          ) : null}
          {needsYou ? (
            <div className="dash-attention" role="status">
              <span className="dash-attention-dot" aria-hidden="true" />
              <span className="dash-attention-lead">{needsYou} {needsYou === 1 ? "needs" : "need"} you</span>
              <span className="dash-attention-where">
                {[
                  questions ? questions + (questions === 1 ? " question in Inbox" : " questions in Inbox") : "",
                  waitingTerminals.length
                    ? waitingTerminals.length + (waitingTerminals.length === 1 ? " terminal" : " terminals") + " waiting"
                    : "",
                ].filter(Boolean).join(" \u00b7 ")}
              </span>
              {questions ? (
                <button type="button" className="btn btn-sm dash-attention-act" onClick={() => onOpenApp && onOpenApp("inbox")}>Open Inbox</button>
              ) : (
                <button type="button" className="btn btn-sm dash-attention-act" onClick={() => onOpen && onOpen(termTabId(waitingTerminals[0].id))}>
                  Open {waitingTerminals[0].name}
                </button>
              )}
            </div>
          ) : null}
          <div className="dash-kpi-row">
            <StatTile
              label="Spend"
              value={firstLoad ? "" : spend === NOT_REPORTED ? EM : money(stats.current.cost)}
              delta={firstLoad || spend === NOT_REPORTED ? null : deltaPercent(stats.current.cost, stats.prior && stats.prior.cost)}
              compareLabel={spend === NOT_REPORTED ? "" : cmp}
              points={firstLoad ? null : stats.series.map((d) => d.cost)}
              loading={firstLoad}
            >
              {!firstLoad && spend === NOT_REPORTED ? (
                <div className="stat-tile-delta stat-tile-delta-muted">not priced by any CLI that ran</div>
              ) : !firstLoad && spend === PARTIAL ? (
                <div className="stat-tile-delta stat-tile-delta-muted">a floor &mdash; some sessions are unpriced</div>
              ) : null}
            </StatTile>
            <StatTile
              label="Activity"
              value={firstLoad ? "" : stats.current.messages.toLocaleString() + " msgs"}
              delta={firstLoad ? null : deltaPercent(stats.current.messages, stats.prior && stats.prior.messages)}
              compareLabel={cmp}
              points={firstLoad ? null : stats.series.map((d) => d.messages)}
              loading={firstLoad}
            />
            <StatTile
              label="Sessions"
              value={firstLoad ? "" : stats.current.sessions.toLocaleString()}
              deltaText={sessionsDelta == null ? null : (sessionsDelta >= 0 ? "+" : "−") + Math.abs(sessionsDelta)}
              compareLabel={cmp}
              points={firstLoad ? null : stats.series.map((d) => d.turns)}
              loading={firstLoad}
            />
            <StatTile
              label="Fleet"
              value={fleet.live + " live"}
              // "live now" only when something is: a timestamp under a zero
              // would be the third way of saying there is nothing to see.
              compareLabel={fleet.live ? "live now" : ""}
              points={null}
              loading={false}
            >
              {fleet.live ? (
                <>
                  <div className="fleet-strip">
                    {FLEET_ORDER.filter((state) => fleet[state] > 0).map((state) => (
                      <span key={state} className={state === FLEET_NEEDS_YOU ? "is-waiting" : ""} title={FLEET_HINTS[state]}>
                        {fleet[state]} {FLEET_LABELS[state]}
                      </span>
                    ))}
                    {fleet.shells.running ? (
                      <span className="fleet-shells" title="Plain terminals, not agents — no CLI is running in them">
                        {fleet.shells.running} {fleet.shells.running === 1 ? "shell" : "shells"}
                      </span>
                    ) : null}
                  </div>
                  <ul className={"fleet-agents" + (fleetOpen ? " is-open" : "")}>
                    {(fleetOpen ? fleet.units : fleet.units.slice(0, 4)).map((u) => (
                      <li key={u.key}>
                        <button
                          type="button"
                          className={"fleet-agent is-" + u.state}
                          title={u.name + " · " + (u.kind === "terminal" ? terminalCliLabel(u.detail) : u.detail || "agent") + " · " + FLEET_LABELS[u.state]}
                          onClick={() => onOpen && onOpen(u.kind === "terminal" ? termTabId(u.id) : u.id)}
                        >
                          <span className="fleet-agent-name">{u.name}</span>
                          <span className="fleet-agent-where">
                            {u.kind === "terminal" ? terminalCliLabel(u.detail) : shortModel(u.detail)}
                          </span>
                          <span className="fleet-agent-state">{FLEET_LABELS[u.state]}</span>
                        </button>
                      </li>
                    ))}
                  </ul>
                  {fleet.units.length > 4 ? (
                    <button type="button" className="fleet-more" aria-expanded={fleetOpen} onClick={() => setFleetOpen((open) => !open)}>
                      {fleetOpen ? "Show less" : "+" + (fleet.units.length - 4) + " more"}
                    </button>
                  ) : null}
                </>
              ) : (
                <div className="stat-tile-delta stat-tile-delta-muted">
                  {fleet.agents.total + fleet.terminals.total + fleet.shells.total ? "nothing running" : "no agents yet"}
                </div>
              )}
            </StatTile>
          </div>

          <div className="dashboard-section">
            {firstLoad ? (
              <div className="spend-skel" aria-hidden="true">
                <span className="skel-line w-40" />
                <span className="skel-line w-90" />
              </div>
            ) : (
              <DailyChart series={stats.series} metric={metric} onMetric={setMetric} />
            )}
          </div>

          <div className="dash-grid-2">
            <div className="dashboard-section">
              <div className="dash-section-label">By CLI</div>
              {firstLoad ? <Skel /> : (
                <>
                  <RankedBars items={cliItems} empty={nothingRan} format={money} />
                  {costNote ? <p className="dash-note">{costNote}</p> : null}
                </>
              )}
            </div>
            <div className="dashboard-section">
              <div className="dash-section-label">Spend by model</div>
              {firstLoad ? <Skel /> : <RankedBars items={modelItems} limit={6} more={(n) => n + " more models"} format={money} />}
            </div>
            <div className="dashboard-section">
              <div className="dash-section-label">Spend by workspace</div>
              {firstLoad ? <Skel /> : <RankedBars items={workspaceItems} limit={6} more={(n) => n + " more folders"} format={money} />}
            </div>
            <div className="dashboard-section">
              <div className="dash-section-label">Tokens</div>
              {firstLoad ? <Skel /> : <TokenBar tokens={stats.tokens} />}
            </div>
            <div className="dashboard-section">
              <div className="dash-section-label">Tools</div>
              {firstLoad ? <Skel /> : (
                <>
                  <RankedBars items={toolItems} empty="No tool calls in this period." />
                  {toolsNote ? <p className="dash-note">{toolsNote}</p> : null}
                </>
              )}
            </div>
            <div className="dashboard-section">
              <div className="dash-section-label">Code impact</div>
              {firstLoad ? <Skel /> : impact ? (
                <>
                  <dl className="dash-facts">
                    <div><dt>Lines added</dt><dd>+{formatLines(impact.linesAdded)}</dd></div>
                    <div><dt>Lines removed</dt><dd>&minus;{formatLines(impact.linesRemoved)}</dd></div>
                    <div><dt>Cost per line</dt><dd>{fmtRate(costPerLine(stats.current.cost, impact.linesAdded, impact.linesRemoved))}</dd></div>
                  </dl>
                  {impactNote ? <p className="dash-note">{impactNote}</p> : null}
                </>
              ) : <p className="dash-empty">{idle ? nothingRan : "No agent CLI here counts the lines it changed."}</p>}
            </div>
            <div className="dashboard-section">
              <div className="dash-section-label">Agent time</div>
              {firstLoad ? <Skel /> : timing ? (
                <>
                  <dl className="dash-facts">
                    <div><dt>Waiting on models</dt><dd>{formatDuration(timing.apiMs)}</dd></div>
                    <div><dt>Running tools</dt><dd>{formatDuration(timing.toolMs)}</dd></div>
                    <div><dt>Sessions open</dt><dd>{formatDuration(timing.sessionMs)}</dd></div>
                  </dl>
                  <p className="dash-note">
                    Agent time, not elapsed &mdash; sessions run at once, so these add up past the clock.
                    {timingNote ? " " + timingNote : ""}
                  </p>
                </>
              ) : <p className="dash-empty">{idle ? nothingRan : "No agent CLI here records how long a request took."}</p>}
            </div>
            <div className="dashboard-section">
              <div className="dash-section-label">Desktop</div>
              {firstLoad ? <Skel /> : desktop && (desktop.steps || desktop.refused || desktop.failed) ? (
                <>
                  <dl className="dash-facts">
                    <div><dt>Steps on the computer</dt><dd>{desktop.steps}</dd></div>
                    <div><dt>Time acting</dt><dd>{formatDuration(desktop.ms)}</dd></div>
                    <div><dt>Refused or failed</dt><dd>{desktop.refused + desktop.failed}</dd></div>
                  </dl>
                  <p className="dash-note">Computer use through the desktop app (ADR-0148): every step the shell ran for an agent, and the calls it turned away.</p>
                </>
              ) : <p className="dash-empty">No agent used the computer in this period.</p>}
            </div>
            <div className="dashboard-section">
              <div className="dash-section-label">Limits</div>
              {firstLoad ? <Skel /> : (
                <>
                  <LimitBars limits={stats.limits} plans={stats.plans} />
                  {limitsNote ? <p className="dash-note">{limitsNote}</p> : null}
                </>
              )}
            </div>
            <div className="dashboard-section">
              <div className="dash-section-label">Efficiency</div>
              {firstLoad ? <Skel /> : (
                <dl className="dash-facts">
                  <div><dt>Cache hit</dt><dd>{stats.tokens.cacheHit == null ? EM : Math.round(stats.tokens.cacheHit) + "%"}</dd></div>
                  <div><dt>Cost per turn</dt><dd>{fmtRate(costPerTurn(stats.current.cost, turns.assistant))}</dd></div>
                  <div><dt>Cache reads</dt><dd>{idle ? EM : formatTokens(stats.tokens.cacheRead)}</dd></div>
                  <div><dt>Spent on cache</dt><dd>{stats.costSplit && stats.costSplit.cacheRead ? formatMoney(stats.costSplit.cacheRead, "usd") : EM}</dd></div>
                </dl>
              )}
            </div>
            <div className="dashboard-section">
              <div className="dash-section-label">Reliability</div>
              {firstLoad ? <Skel /> : (
                <dl className="dash-facts">
                  <div><dt>Turns</dt><dd>{turns.assistant.toLocaleString()}</dd></div>
                  <div><dt>Errors</dt><dd>{turns.errors.toLocaleString()}{percent(turns.errors, turns.assistant) ? <span className="dash-fact-sub"> {percent(turns.errors, turns.assistant)}</span> : null}</dd></div>
                  <div><dt>Aborted</dt><dd>{turns.aborted.toLocaleString()}</dd></div>
                  <div><dt>Refusals</dt><dd>{Number(turns.refusals || 0).toLocaleString()}</dd></div>
                  <div><dt>Compactions</dt><dd>{turns.compactions.toLocaleString()}</dd></div>
                  <div><dt>Prompts</dt><dd>{turns.user.toLocaleString()}</dd></div>
                  <div><dt>Output tokens</dt><dd>{formatTokens(stats.tokens.output)}</dd></div>
                </dl>
              )}
              {errorsNote ? <p className="dash-note">{errorsNote}</p> : null}
            </div>
            <div className="dashboard-section">
              <div className="dash-section-label">Top sessions</div>
              {firstLoad ? <Skel /> : <TopSessions items={stats.topSessions} />}
            </div>
            <div className="dashboard-section dash-coverage-section">
              <div className="dash-section-label">What each CLI reports</div>
              {firstLoad ? <Skel /> : <CoveragePanel coverage={coverage} />}
            </div>
          </div>
        </div>
      )}
      </div>
    </div>
  );
}

const EM = "\u2014";

// cliLabelOf prefers the label the server sent over a client-side table, so
// a CLI added on the server names itself here without a web change.
function cliLabelOf(coverage, cli) {
  const row = (coverage || []).find((r) => r && r.cli === cli);
  return (row && row.label) || cli;
}

// money never prints "$0.00" for spend that happened. OpenCode's four
// messages in a real window cost $0.0014; rendered as $0.00 the row reads
// as free, which is the one thing a spend column must never say by
// accident. Same rule providerRows.formatSpend already applies to the
// credential roster.
function money(v) {
  const n = Number(v) || 0;
  if (n > 0 && n < 0.01) return "<$0.01";
  return formatMoney(n, "usd");
}

// fmtRate prints a per-unit cost, or an em dash when there was nothing to
// divide by — and also when the rate lands on exactly zero, which happens
// when the numerator came from a CLI that records no price. "$0.00 per
// line" reads as free; a dash reads as unknown, which is the truth.
function fmtRate(v) {
  if (v == null || !(v > 0)) return EM;
  return money(v);
}

function Skel() {
  return (
    <div className="spend-skel" aria-hidden="true">
      <span className="skel-line w-80" />
      <span className="skel-line w-50" />
      <span className="skel-line w-70" />
    </div>
  );
}
