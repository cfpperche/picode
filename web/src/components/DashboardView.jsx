import { useEffect, useRef, useState } from "react";
import { api } from "../lib/api.js";
import { formatMoney } from "../lib/providerUsage.js";
import { deltaPercent, fleetStats, compareLabel } from "../lib/dashboardStats.js";
import { readDashboardRange, writeDashboardRange } from "../lib/openTabs.js";
import { relTime, absTime } from "../lib/relTime.js";
import StatTile from "./StatTile.jsx";
import SpendByProvider from "./SpendByProvider.jsx";
import DateRangePicker from "./DateRangePicker.jsx";

// The landing surface for "no tab open" once real data exists (ADR on the
// session observability dashboard) — spend/activity/fleet metrics, not
// another list of the workspaces/agents the sidebar already shows. App.jsx
// only mounts this when workspaces+freeAgents+terminals adds up to more
// than zero; the true empty-environment card is untouched and lives beside
// it.
export default function DashboardView({ workspaces, freeAgents }) {
  const [range, setRange] = useState(() => readDashboardRange());
  const [stats, setStats] = useState(null);
  const [busy, setBusy] = useState(false);
  const [loadErr, setLoadErr] = useState("");
  const [fetchedAt, setFetchedAt] = useState(null);
  const mounted = useRef(false);

  useEffect(() => {
    const keep = mounted.current; // false on first mount (full skeleton), true on a range switch (keep old numbers)
    mounted.current = true;
    return load(keep);
  }, [range]);

  function load(keep) {
    let cancelled = false;
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

  const fleet = fleetStats(workspaces, freeAgents);
  const firstLoad = stats === null;

  return (
    <div className="dashboard-view">
      <div className="dash-head">
        <DateRangePicker value={range} onChange={onRange} />
        {fetchedAt ? <span className="dash-asof" title={absTime(fetchedAt)}>updated {relTime(fetchedAt)}</span> : null}
      </div>

      {loadErr && firstLoad ? (
        <p className="file-pane-msg">{loadErr} <button type="button" className="btn btn-sm" onClick={() => load(false)}>Retry</button></p>
      ) : (
        <>
          {loadErr ? (
            <p className="dash-refresh-err">{loadErr} <button type="button" className="btn-link" onClick={() => load(true)}>Retry</button></p>
          ) : null}
          <div className="dash-kpi-row" style={{ opacity: busy && !firstLoad ? 0.6 : 1 }}>
            <StatTile
              label="Spend"
              value={firstLoad ? "" : formatMoney(stats.current.cost, "usd")}
              delta={firstLoad ? null : deltaPercent(stats.current.cost, stats.prior && stats.prior.cost)}
              compareLabel={compareLabel(range)}
              points={firstLoad ? null : stats.series.map((d) => d.cost)}
              loading={firstLoad}
            />
            <StatTile
              label="Activity"
              value={firstLoad ? "" : stats.current.messages.toLocaleString() + " msgs"}
              delta={firstLoad ? null : deltaPercent(stats.current.messages, stats.prior && stats.prior.messages)}
              compareLabel={compareLabel(range)}
              points={firstLoad ? null : stats.series.map((d) => d.messages)}
              loading={firstLoad}
            />
            <StatTile
              label="Fleet"
              value={fleet.running + " / " + fleet.total + " running"}
              delta={null}
              compareLabel=""
              points={null}
              loading={false}
            />
          </div>

          <div className="dashboard-section">
            <div className="dash-section-label">Spend by provider</div>
            {firstLoad ? (
              <div className="spend-skel" aria-hidden="true">
                <span className="skel-line w-80" />
                <span className="skel-line w-50" />
              </div>
            ) : (
              <SpendByProvider items={stats.byProvider} />
            )}
          </div>
        </>
      )}
    </div>
  );
}
