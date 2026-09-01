import { sparklinePath } from "../lib/sparkline.js";

// One KPI tile: label, a pre-formatted value, an optional neutral delta
// chip, and an optional sparkline. Dumb by design — DashboardView does all
// the formatting/period math, this just lays it out.
export default function StatTile({ label, value, delta, compareLabel, points, loading }) {
  const spark = points && points.length >= 2 ? sparklinePath(points, { width: 72, height: 22, pad: 2 }) : null;

  return (
    <div className="stat-tile">
      <div className="stat-tile-head">
        <span className="stat-tile-label">{label}</span>
        {spark ? (
          <svg className="stat-tile-spark" viewBox={"0 0 " + spark.width + " " + spark.height} aria-hidden="true">
            {spark.mainPath ? <path d={spark.mainPath} className="stat-tile-spark-hist" /> : null}
            <path d={spark.headPath} className="stat-tile-spark-cur" />
            <circle cx={spark.dot.x} cy={spark.dot.y} r="1.6" className="stat-tile-spark-dot" />
          </svg>
        ) : null}
      </div>
      {loading ? (
        <div className="stat-tile-skel" aria-hidden="true">
          <span className="skel-line w-70" />
        </div>
      ) : (
        <>
          <div className="stat-tile-value">{value}</div>
          {delta != null ? (
            <div className="stat-tile-delta">
              <span>{delta >= 0 ? "↑" : "↓"} {Math.abs(delta).toFixed(0)}%</span>
              {compareLabel ? <span className="stat-tile-compare"> {compareLabel}</span> : null}
            </div>
          ) : compareLabel ? (
            <div className="stat-tile-delta stat-tile-delta-muted">{compareLabel}</div>
          ) : null}
        </>
      )}
    </div>
  );
}
