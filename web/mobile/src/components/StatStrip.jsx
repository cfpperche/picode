import { formatMoney } from "@picode/shared/domain/providerUsage.js";
import { deltaPercent, spendState, NOT_REPORTED } from "@picode/shared/domain/dashboardStats.js";

// Today's spend and activity in one line — the dashboard's headline, not
// the dashboard. Tapping goes nowhere: the desktop has the breakdowns.
export default function StatStrip({ stats }) {
  if (!stats) return null;
  const cost = stats.current ? stats.current.cost : 0;
  const msgs = stats.current ? stats.current.messages : 0;
  const sessions = stats.current ? stats.current.sessions : 0;
  const d = stats.prior ? deltaPercent(cost, stats.prior.cost) : null;
  // Unmeasured is not zero. A day whose only sessions are live Claude Code
  // ones has no price on disk yet, and "$0.00" here would be read as free —
  // on the surface with the least room to explain itself (ADR-0097).
  const priced = spendState(stats.byCli) !== NOT_REPORTED;
  return (
    <div className="m-stats" aria-label="Today">
      <div className="m-stat">
        <span className="m-stat-label">Spend today</span>
        <span className="m-stat-value" title={priced ? "" : "No CLI that ran today records a price yet"}>
          {priced ? (cost > 0 && cost < 0.01 ? "<$0.01" : formatMoney(cost, "usd")) : "\u2014"}
        </span>
        {priced && d != null ? <span className="m-stat-delta">{(d >= 0 ? "↑ " : "↓ ") + Math.abs(d).toFixed(0) + "% vs. yesterday"}</span> : null}
        {!priced ? <span className="m-stat-delta">not priced yet</span> : null}
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
