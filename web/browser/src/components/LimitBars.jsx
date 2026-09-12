import { resetsIn } from "@picode/shared/domain/dashboardStats.js";

// Quota windows a CLI reported against its own plan (ADR-0097).
//
// This is where Codex's cost cell went. It never prices a token, and PiCode
// refuses to invent a price table that would age silently — but Codex does
// record the thing that actually constrains a plan: how much of a window is
// spent and when it resets. That is the more useful number anyway; nobody
// on a subscription is watching dollars.
//
// The meter is deliberately the same one-hue track the spend rankings use.
// A red band past 80% would be a threshold, and PiCode has no budget
// concept to justify one (ADR-0041's standing refusal).
export default function LimitBars({ limits, now }) {
  const rows = (limits || []).filter((l) => l && l.windowMinutes > 0);
  if (rows.length === 0) {
    return <p className="dash-empty">No agent CLI here reports a quota window.</p>;
  }
  return (
    <ul className="spend-list">
      {rows.map((l) => {
        const pct = Math.max(0, Math.min(100, Number(l.usedPercent) || 0));
        const reset = resetsIn(l.resetsAt, now);
        return (
          <li key={l.cli + ":" + l.windowMinutes} className="spend-row">
            <span className="spend-name" title={l.cli + (l.plan ? " · " + l.plan + " plan" : "")}>
              <span className="spend-label">{l.label || l.windowMinutes + "m"}</span>
              <span className="spend-sub">{l.cli}</span>
            </span>
            <span className="spend-bar-track">
              <span className="spend-bar-fill" style={{ width: Math.max(2, pct) + "%" }} />
            </span>
            <span className="spend-amount">{Math.round(pct)}%</span>
            {reset ? <span className="limit-reset">{reset}</span> : null}
          </li>
        );
      })}
    </ul>
  );
}
