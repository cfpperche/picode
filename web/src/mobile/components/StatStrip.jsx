import { formatMoney } from "../../lib/providerUsage.js";
import { deltaPercent } from "../../lib/dashboardStats.js";

// Today's spend and activity in one line — the dashboard's headline, not
// the dashboard. Tapping goes nowhere: the desktop has the breakdowns.
export default function StatStrip({ stats }) {
  if (!stats) return null;
  const cost = stats.current ? stats.current.cost : 0;
  const msgs = stats.current ? stats.current.messages : 0;
  const sessions = stats.current ? stats.current.sessions : 0;
  const d = stats.prior ? deltaPercent(cost, stats.prior.cost) : null;
  return (
    <div className="m-stats" aria-label="Today">
      <div className="m-stat">
        <span className="m-stat-label">Spend today</span>
        <span className="m-stat-value">{formatMoney(cost, "usd")}</span>
        {d != null ? <span className="m-stat-delta">{(d >= 0 ? "↑ " : "↓ ") + Math.abs(d).toFixed(0) + "% vs. yesterday"}</span> : null}
      </div>
      <div className="m-stat">
        <span className="m-stat-label">Messages</span>
        <span className="m-stat-value">{Number(msgs).toLocaleString()}</span>
      </div>
      <div className="m-stat">
        <span className="m-stat-label">Sessions</span>
        <span className="m-stat-value">{Number(sessions).toLocaleString()}</span>
      </div>
    </div>
  );
}
