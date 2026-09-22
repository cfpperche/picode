import { limitReading, resetsIn, observedAge, limitRows } from "@picode/shared/domain/dashboardStats.js";
import { providerName } from "@picode/shared/domain/credentials.js";
import { cliProvidersHash } from "@picode/shared/domain/cliProviders.js";
import { terminalCliLabel } from "@picode/shared/domain/terminalCli.js";
import { relTime, absTime } from "@picode/shared/domain/relTime.js";

// Quota windows: the ones a CLI reported against its own plan (ADR-0097)
// and the plan windows the Providers roster fetched per signed-in account
// (ADR-0031, from its cache — t3code's usage page reads plan limits for
// five vendors the same way; reading only Codex's rollouts left every other
// plan off this card).
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
export default function LimitBars({ limits, plans, now }) {
  const { rows, blocked } = limitRows(limits, plans, providerName);
  const at = now || Date.now();
  const fix = blocked.length ? (
    <ul className="limit-blocked">
      {blocked.map((b) => (
        <li key={b.key}>
          <span>{b.name}{b.label ? " · " + b.label : ""}: {b.status === "auth_required" ? "sign in again to read its plan." : "its plan could not be read."}</span>
          {b.cli ? <button type="button" className="btn-link" onClick={() => { location.hash = cliProvidersHash(b.cli); }}>Open {terminalCliLabel(b.cli)} providers</button> : null}
        </li>
      ))}
    </ul>
  ) : null;
  if (rows.length === 0) {
    return (
      <>
        <p className="dash-empty">No plan or CLI here reports a quota window.</p>
        {fix}
      </>
    );
  }
  return (
    <>
      <ul className="spend-list">
        {rows.map((l) => {
          const pct = Math.max(0, Math.min(100, l.usedPercent));
          const reading = limitReading(l, at);
          const age = observedAge(l, at);
          const ageText = age == null ? "" : relTime(l.observedAt, at);
          const reset = reading === "current" ? resetsIn(l.resetsAt, at) : "";
          const title = [
            l.sub + (l.plan ? " · " + l.plan + " plan" : ""),
            ageText ? (ageText === "now" ? "read just now" : "read " + ageText + " ago") : "",
            ageText ? absTime(l.observedAt) : "",
          ].filter(Boolean).join(" · ");
          return (
            <li key={l.key} className={"spend-row" + (reading === "expired" ? " is-stale" : "")}>
              <span className="spend-name" title={title}>
                <span className="spend-label">{l.label}</span>
                <span className="spend-sub">{l.sub}</span>
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
      {fix}
    </>
  );
}
