import { useEffect, useRef, useState } from "react";
import { api } from "../lib/api.js";
import { formatMoney } from "../lib/providerUsage.js";
import { deltaPercent, fleetStats, compareLabel, formatTokens, percent, folderLabel } from "../lib/dashboardStats.js";
import { readDashboardRange, writeDashboardRange } from "../lib/openTabs.js";
import { relTime, absTime } from "../lib/relTime.js";
import { shortModel } from "../lib/chip.js";
import StatTile from "./StatTile.jsx";
import RankedBars from "./RankedBars.jsx";
import DailyChart from "./DailyChart.jsx";
import TokenBar from "./TokenBar.jsx";
import TopSessions from "./TopSessions.jsx";
import DateRangePicker from "./DateRangePicker.jsx";
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
export default function DashboardView({ workspaces, freeAgents, workingIds, waitingId }) {
  const [range, setRange] = useState(() => readDashboardRange());
  const [metric, setMetric] = useState("cost");
  const [stats, setStats] = useState(null);
  const [busy, setBusy] = useState(false);
  const [loadErr, setLoadErr] = useState("");
  const [fetchedAt, setFetchedAt] = useState(null);
  const [, setTick] = useState(0);
  const mounted = useRef(false);
  const inflight = useRef(null);

  useEffect(() => {
    const keep = mounted.current; // false on first mount (full skeleton), true on a range switch (keep old numbers)
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
        setFetchedAt(new Date().toISOString());
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

  const fleet = fleetStats(workspaces, freeAgents, { workingIds, waitingId });
  const firstLoad = stats === null;
  const cmp = compareLabel(range);
  const sessionsDelta = !firstLoad && stats.prior ? stats.current.sessions - stats.prior.sessions : null;
  const turns = firstLoad ? null : stats.turns;

  // A zero-cost "unknown" model row is pi bookkeeping (a turn with no
  // usage block), not a lever anyone can pull — keep the ranking honest
  // by dropping only that.
  const modelItems = firstLoad ? [] : stats.byModel.filter((m) => !(m.model === "unknown" && m.cost === 0)).map((m) => ({
    key: m.provider + "/" + m.model,
    label: shortModel(m.model),
    sub: m.provider,
    face: <ProviderFace id={m.provider} />,
    value: m.cost,
    display: formatMoney(m.cost, "usd"),
    title: m.provider + " · " + m.model + " · " + m.messages.toLocaleString() + " turns",
  }));
  const workspaceItems = firstLoad ? [] : stats.byWorkspace.map((w) => ({
    key: w.cwd,
    label: w.workspace || folderLabel(w.cwd) || w.cwd,
    value: w.cost,
    display: formatMoney(w.cost, "usd"),
    title: w.cwd + " · " + w.sessions + (w.sessions === 1 ? " session" : " sessions") + (w.workspace ? "" : " · not a PiCode workspace"),
  }));
  const toolItems = firstLoad ? [] : stats.tools.map((t) => ({
    key: t.name,
    label: t.name,
    value: t.calls,
    display: t.calls.toLocaleString(),
  }));

  return (
    <div className="dashboard-view">
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
          <div className="dash-kpi-row">
            <StatTile
              label="Spend"
              value={firstLoad ? "" : formatMoney(stats.current.cost, "usd")}
              delta={firstLoad ? null : deltaPercent(stats.current.cost, stats.prior && stats.prior.cost)}
              compareLabel={cmp}
              points={firstLoad ? null : stats.series.map((d) => d.cost)}
              loading={firstLoad}
            />
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
              value={fleet.running + " / " + fleet.total + " running"}
              compareLabel=""
              points={null}
              loading={false}
            >
              {fleet.running ? (
                <div className="fleet-strip">
                  <span className={fleet.working ? "" : "is-zero"}>{fleet.working} working</span>
                  <span className={fleet.waiting ? "is-waiting" : "is-zero"}>{fleet.waiting} waiting</span>
                  <span className={fleet.idle ? "" : "is-zero"}>{fleet.idle} idle</span>
                </div>
              ) : (
                <div className="stat-tile-delta stat-tile-delta-muted">{fleet.total ? "nothing running" : "no agents yet"}</div>
              )}
              {fleet.agents.length ? (
                <ul className="fleet-agents">
                  {fleet.agents.slice(0, 3).map((a) => (
                    <li key={a.id} className={"fleet-agent is-" + a.state} title={a.name + (a.model ? " · " + a.model : "") + " · " + a.state}>
                      <span className="fleet-agent-name">{a.name}</span>
                      {a.model ? <span className="fleet-agent-model">{shortModel(a.model)}</span> : null}
                    </li>
                  ))}
                  {fleet.agents.length > 3 ? <li className="fleet-agent fleet-agent-more">+{fleet.agents.length - 3} more</li> : null}
                </ul>
              ) : null}
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
              <div className="dash-section-label">Spend by model</div>
              {firstLoad ? <Skel /> : <RankedBars items={modelItems} limit={6} more={(n) => n + " more models"} format={(v) => formatMoney(v, "usd")} />}
            </div>
            <div className="dashboard-section">
              <div className="dash-section-label">Spend by workspace</div>
              {firstLoad ? <Skel /> : <RankedBars items={workspaceItems} limit={6} more={(n) => n + " more folders"} format={(v) => formatMoney(v, "usd")} />}
            </div>
            <div className="dashboard-section">
              <div className="dash-section-label">Tokens</div>
              {firstLoad ? <Skel /> : <TokenBar tokens={stats.tokens} />}
            </div>
            <div className="dashboard-section">
              <div className="dash-section-label">Tools</div>
              {firstLoad ? <Skel /> : <RankedBars items={toolItems} empty="No tool calls in this period." />}
            </div>
            <div className="dashboard-section">
              <div className="dash-section-label">Reliability</div>
              {firstLoad ? <Skel /> : (
                <dl className="dash-facts">
                  <div><dt>Turns</dt><dd>{turns.assistant.toLocaleString()}</dd></div>
                  <div><dt>Errors</dt><dd>{turns.errors.toLocaleString()}{percent(turns.errors, turns.assistant) ? <span className="dash-fact-sub"> {percent(turns.errors, turns.assistant)}</span> : null}</dd></div>
                  <div><dt>Aborted</dt><dd>{turns.aborted.toLocaleString()}</dd></div>
                  <div><dt>Compactions</dt><dd>{turns.compactions.toLocaleString()}</dd></div>
                  <div><dt>Prompts</dt><dd>{turns.user.toLocaleString()}</dd></div>
                  <div><dt>Output tokens</dt><dd>{formatTokens(stats.tokens.output)}</dd></div>
                </dl>
              )}
            </div>
            <div className="dashboard-section">
              <div className="dash-section-label">Top sessions</div>
              {firstLoad ? <Skel /> : <TopSessions items={stats.topSessions} />}
            </div>
          </div>
        </div>
      )}
    </div>
  );
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
