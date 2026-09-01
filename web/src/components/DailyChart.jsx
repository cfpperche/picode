import { barChart } from "../lib/barchart.js";
import { dayLabel } from "../lib/dashboardStats.js";
import { formatMoney } from "../lib/providerUsage.js";

const METRICS = [
  { key: "cost", label: "Spend", fmt: (v) => formatMoney(v, "usd") },
  { key: "messages", label: "Messages", fmt: (v) => Number(v || 0).toLocaleString() },
  { key: "turns", label: "Turns", fmt: (v) => Number(v || 0).toLocaleString() },
];

// One bar per calendar day of the period. Hover reads the exact day and
// value through the SVG's native <title> — no tooltip library, and it is
// what screen readers announce too. The viewBox is stretched to the
// container (preserveAspectRatio="none"), so the axis labels live in HTML
// underneath instead of distorting inside the SVG.
export default function DailyChart({ series, metric, onMetric }) {
  const m = METRICS.find((x) => x.key === metric) || METRICS[0];
  const chart = barChart((series || []).map((d) => d[m.key]), { width: 600, height: 72, gap: 2, minH: 1.5 });
  const first = series && series.length ? series[0].date : "";
  const last = series && series.length > 1 ? series[series.length - 1].date : "";
  return (
    <div className="dash-chart">
      <div className="dash-chart-head">
        <span className="dash-section-label">Daily</span>
        <div className="dash-range dash-metric" role="radiogroup" aria-label="Daily chart metric">
          {METRICS.map((opt) => (
            <label key={opt.key} className="dash-range-opt">
              <input type="radio" name="dash-metric" value={opt.key} checked={m.key === opt.key} onChange={() => onMetric(opt.key)} />
              <span className="dash-range-face">{opt.label}</span>
            </label>
          ))}
        </div>
      </div>
      {!chart || chart.max === 0 ? (
        <p className="dash-empty">No session activity in this period.</p>
      ) : (
        <>
          <svg className="dash-chart-svg" viewBox={"0 0 " + chart.width + " " + chart.height} preserveAspectRatio="none" role="img" aria-label={m.label + " per day"}>
            {chart.bars.map((b) => (
              <rect key={b.i} x={b.x} y={b.y} width={b.w} height={b.h} className={"dash-chart-bar" + (b.i === chart.bars.length - 1 ? " is-last" : "")}>
                <title>{dayLabel(series[b.i].date) + " · " + m.fmt(b.v)}</title>
              </rect>
            ))}
          </svg>
          <div className="dash-chart-axis" aria-hidden="true">
            <span>{dayLabel(first)}</span>
            <span>{chart.max > 0 ? "peak " + m.fmt(chart.max) : ""}</span>
            <span>{last ? dayLabel(last) : ""}</span>
          </div>
        </>
      )}
    </div>
  );
}

