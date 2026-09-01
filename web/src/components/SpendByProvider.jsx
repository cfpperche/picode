import { formatMoney } from "../lib/providerUsage.js";

// Ranked horizontal bar list, one flat hue (--accent via .spend-bar-fill) —
// the provider name is already the identity label, so no per-provider
// categorical color is needed (dataviz form: magnitude comparison with a
// direct label -> sequential one hue, not categorical).
export default function SpendByProvider({ items }) {
  if (!items || items.length === 0) {
    return <p className="side-empty">No session activity in this period.</p>;
  }
  const max = Math.max(...items.map((i) => i.cost), 0.01);
  return (
    <ul className="spend-list">
      {items.map((i) => (
        <li key={i.provider} className="spend-row">
          <span className="spend-name">{i.provider}</span>
          <span className="spend-bar-track">
            <span className="spend-bar-fill" style={{ width: Math.max(2, (i.cost / max) * 100) + "%" }} />
          </span>
          <span className="spend-amount">{formatMoney(i.cost, "usd")}</span>
        </li>
      ))}
    </ul>
  );
}
