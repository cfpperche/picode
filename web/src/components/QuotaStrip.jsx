import { barTone, formatMoney, formatReset } from "../lib/providerUsage.js";
import { STATE_LIVE, STATE_NONE, barWindows, formatAge, moneyWindows, quotaNote, quotaState } from "../lib/providerRows.js";

// The compact quota reading on a roster row. Three states, never two:
// a live bar, the same bar marked with its age, or a word saying which kind
// of nothing this is. A percentage is only ever drawn from a number a
// vendor actually returned (ADR-0031).
export default function QuotaStrip({ entry, onRefresh, busy }) {
  const state = quotaState(entry);
  if (state === STATE_NONE) {
    const note = quotaNote(entry);
    return (
      <span className="quota-strip quota-empty">
        <span className="quota-note">{note}</span>
        {onRefresh ? (
          <button type="button" className="quota-refresh" onClick={onRefresh} disabled={busy}>
            {busy ? "Checking" : "Check"}
          </button>
        ) : null}
      </span>
    );
  }
  const wins = barWindows(entry);
  const money = moneyWindows(entry);
  return (
    <span className="quota-strip">
      {money.map((w) => (
        <span key={w.id || w.label} className="quota-money" title={w.label}>
          {formatMoney(w.remaining, w.unit)} left
        </span>
      ))}
      {wins.map((w) => {
        const used = Math.max(0, Math.min(100, Number(w.usedPercent)));
        return (
          <span key={w.id || w.label} className={"quota-win " + barTone(used)} title={w.label + (w.resetsAt ? " · resets " + formatReset(w.resetsAt) : "")}>
            <span className="quota-win-id">{w.id || w.label}</span>
            <span className="quota-track"><span className="quota-fill" style={{ width: used + "%" }} /></span>
            <span className="quota-pct">{Math.round(used)}%</span>
          </span>
        );
      })}
      {state === STATE_LIVE && !busy ? (
        <span className="quota-live" title={"Fetched " + formatAge(entry.ageSec) + " ago"}>live</span>
      ) : (
        <button
          type="button"
          className="quota-refresh"
          onClick={onRefresh}
          disabled={busy || !onRefresh}
          title={"Fetched " + formatAge(entry.ageSec) + " ago — check again"}
        >
          {busy ? "Checking" : formatAge(entry.ageSec) + " old"}
        </button>
      )}
    </span>
  );
}
