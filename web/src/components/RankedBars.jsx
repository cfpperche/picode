// Ranked horizontal bar list, one flat hue (--accent via .spend-bar-fill) —
// the row label is already the identity, so no per-row categorical color
// is needed (dataviz form: magnitude comparison with a direct label ->
// sequential one hue, not categorical). Generalised from v1's provider
// list so model / workspace / tool rankings all read the same way.
//   items: [{ key, label, sub?, face?, value, display }]
//   empty: the one-line empty state for this ranking
//   limit / more: past `limit` rows the tail collapses into one muted row
//     labelled by more(count) with the tail's summed value — a dashboard
//     row shows the head of a ranking, not the whole ledger.
export default function RankedBars({ items, empty, limit, more, format }) {
  if (!items || items.length === 0) {
    return <p className="dash-empty">{empty || "No session activity in this period."}</p>;
  }
  let rows = items;
  if (limit && items.length > limit + 1) {
    const tail = items.slice(limit);
    const value = tail.reduce((s, i) => s + i.value, 0);
    rows = items.slice(0, limit).concat([{ key: "\u0000more", label: more ? more(tail.length) : "+" + tail.length + " more", value, display: format ? format(value) : "", muted: true }]);
  }
  const max = Math.max(...rows.map((i) => i.value), 0.01);
  return (
    <ul className="spend-list">
      {rows.map((i) => (
        <li key={i.key} className={"spend-row" + (i.muted ? " is-muted" : "")}>
          <span className="spend-name" title={i.title || (i.sub ? i.label + " · " + i.sub : i.label)}>
            {i.face ? <span className="spend-face">{i.face}</span> : null}
            <span className="spend-label">{i.label}</span>
            {i.sub ? <span className="spend-sub">{i.sub}</span> : null}
          </span>
          <span className="spend-bar-track">
            <span className="spend-bar-fill" style={{ width: Math.max(2, (i.value / max) * 100) + "%" }} />
          </span>
          <span className="spend-amount">{i.display}</span>
        </li>
      ))}
    </ul>
  );
}
