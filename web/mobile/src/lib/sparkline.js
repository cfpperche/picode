// Pure SVG path geometry for a stat-tile trend line — no React, no DOM.
// Mirrors the split lib/gitgraph.js already uses for GitGraph.jsx: this
// module computes points and path strings, the component just draws them.

// sparklinePath(values, opts) -> { mainPath, headPath, dot, width, height } | null
//   - null when fewer than 2 points: a single value is a stat tile, not a
//     trend (dataviz form heuristic — nothing meaningful to draw).
//   - mainPath covers every point except the last segment (the settled,
//     historical run); headPath covers just the last segment (the
//     current/still-accumulating one), so a caller can stroke them in two
//     different tones (history muted, current in the accent).
//   - A flat series (min === max) renders a level line instead of dividing
//     by zero.
export function sparklinePath(values, opts) {
  const width = (opts && opts.width) || 64;
  const height = (opts && opts.height) || 20;
  const pad = (opts && opts.pad) != null ? opts.pad : 2;

  const vals = (values || []).filter((v) => typeof v === "number" && !Number.isNaN(v));
  if (vals.length < 2) return null;

  const min = Math.min(...vals);
  const max = Math.max(...vals);
  const span = max - min;
  const innerW = width - pad * 2;
  const innerH = height - pad * 2;
  const n = vals.length;

  const points = vals.map((v, i) => {
    const x = pad + (n === 1 ? 0 : (innerW * i) / (n - 1));
    const y = span === 0 ? pad + innerH / 2 : pad + innerH - ((v - min) / span) * innerH;
    return [x, y];
  });

  const toPath = (pts) => pts.map((p, i) => (i === 0 ? "M" : "L") + p[0].toFixed(2) + "," + p[1].toFixed(2)).join(" ");

  return {
    mainPath: toPath(points.slice(0, -1)),
    headPath: toPath(points.slice(-2)),
    dot: { x: points[n - 1][0], y: points[n - 1][1] },
    width,
    height,
  };
}
