import { tokenSegments, formatTokens } from "../lib/dashboardStats.js";

// One stacked bar of the period's token volume in four fixed slices, each
// a tone of the single accent (not a categorical palette — the slices are
// parts of one whole), with the slice labelled directly in the rows below.
// Cache hit is the composer status bar's own formula (cacheRead over
// prompt), so the two surfaces never disagree.
export default function TokenBar({ tokens }) {
  const seg = tokenSegments(tokens);
  if (seg.total === 0) {
    return <p className="dash-empty">No token usage in this period.</p>;
  }
  const hit = tokens && tokens.cacheHit != null ? tokens.cacheHit : null;
  return (
    <div className="token-bar">
      <div className="token-bar-total">
        <span className="token-bar-value">{formatTokens(seg.total)}</span>
        <span className="token-bar-hit">{hit == null ? "no cache reads" : "cache hit " + hit.toFixed(0) + "%"}</span>
      </div>
      <div className="token-bar-track" role="img" aria-label={"Tokens: " + seg.parts.map((p) => p.label + " " + formatTokens(p.value)).join(", ")}>
        {seg.parts.map((p) => (p.pct > 0 ? <span key={p.key} className={"token-seg token-seg-" + p.key} style={{ width: p.pct + "%" }} title={p.label + " · " + formatTokens(p.value)} /> : null))}
      </div>
      <ul className="token-rows">
        {seg.parts.map((p) => (
          <li key={p.key} className="token-row">
            <span className={"token-swatch token-seg-" + p.key} aria-hidden="true" />
            <span className="token-row-label">{p.label}</span>
            <span className="token-row-value">{formatTokens(p.value)}</span>
          </li>
        ))}
        {tokens && tokens.reasoning ? (
          <li className="token-row token-row-muted">
            <span className="token-swatch" aria-hidden="true" />
            <span className="token-row-label">reasoning (within output)</span>
            <span className="token-row-value">{formatTokens(tokens.reasoning)}</span>
          </li>
        ) : null}
      </ul>
    </div>
  );
}
