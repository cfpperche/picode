import { limitReading, resetsIn, observedAge } from "@picode/shared/domain/dashboardStats.js";
import { relTime, absTime } from "@picode/shared/domain/relTime.js";

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
//
// Two honesty rules live here, both from reading the panel on a real machine
// (2026-09-14): a percentage is a *reading of a window*, so a window that has
// already reset says so instead of counting down to a past instant, and the
// reading's age is always recoverable from the row (hover) because a quota
// snapshot can be days old.
export default function LimitBars({ limits, now }) {
  const rows = (limits || []).filter((l) => l && l.windowMinutes > 0);
  if (rows.length === 0) {
    return <p className="dash-empty">No agent CLI here reports a quota window.</p>;
  }
  const at = now || Date.now();
  return (
    <ul className="spend-list">
      {rows.map((l) => {
        const pct = Math.max(0, Math.min(100, Number(l.usedPercent) || 0));
        const reading = limitReading(l, at);
        const age = observedAge(l, at);
        const ageText = age == null ? "" : relTime(l.observedAt, at);
        const reset = reading === "current" ? resetsIn(l.resetsAt, at) : "";
        const title = [
          l.cli + (l.plan ? " · " + l.plan + " plan" : ""),
          ageText ? "read " + ageText + " ago" : "",
          ageText ? absTime(l.observedAt) : "",
        ].filter(Boolean).join(" · ");
        return (
          <li key={l.cli + ":" + l.windowMinutes} className={"spend-row" + (reading === "expired" ? " is-stale" : "")}>
            <span className="spend-name" title={title}>
              <span className="spend-label">{l.label || l.windowMinutes + "m"}</span>
              <span className="spend-sub">{l.cli}</span>
            </span>
            <span className="spend-bar-track">
              <span className="spend-bar-fill" style={{ width: Math.max(2, pct) + "%" }} />
            </span>
            <span className="spend-amount">{Math.round(pct)}%</span>
            {reading === "expired" ? (
              // Not a countdown: the window this reading describes is over, so
              // "resets in" would point at a past instant and the percentage
              // would read as the current window's usage.
              <span className="limit-reset" title={"Read " + (ageText ? ageText + " ago" : "before this window ended") + " — the window reset " + relTime(l.resetsAt, at) + " ago"}>
                stale &middot; reset {relTime(l.resetsAt, at)} ago
              </span>
            ) : reset ? <span className="limit-reset">{reset}</span> : null}
          </li>
        );
      })}
    </ul>
  );
}
