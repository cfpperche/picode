// Pure geometry for the dashboard's daily bar chart — no React, no DOM.
// Same split as lib/sparkline.js + StatTile.jsx and lib/gitgraph.js +
// GitGraph.jsx: this module computes rectangles, the component draws them.

// barChart(values, opts) -> { bars: [{x, y, w, h, i, v}], width, height, max } | null
//   - null when there are no values: nothing to draw, the caller shows an
//     empty line instead of an empty axis.
//   - Bars share the width evenly with `gap` between them; a bar's height
//     is proportional to max. An all-zero series yields zero-height bars
//     (max 0 — the caller's cue to say "nothing happened" instead of
//     drawing a baseline), never a divide-by-zero.
//   - Coordinates are in the viewBox space; the component stretches the
//     viewBox to the container (preserveAspectRatio="none"), so nothing
//     text-like belongs inside the SVG.
export function barChart(values, opts) {
  const width = (opts && opts.width) || 600;
  const height = (opts && opts.height) || 80;
  const gap = (opts && opts.gap) != null ? opts.gap : 2;
  const minH = (opts && opts.minH) != null ? opts.minH : 0;

  const vals = (values || []).map((v) => (typeof v === "number" && !Number.isNaN(v) ? v : 0));
  if (vals.length === 0) return null;

  const n = vals.length;
  const max = Math.max(...vals, 0);
  const slot = width / n;
  const w = Math.max(1, slot - gap);

  const bars = vals.map((v, i) => {
    const h = max === 0 ? 0 : Math.max(v > 0 ? minH : 0, (v / max) * height);
    return { i, v, x: +(i * slot + gap / 2).toFixed(2), y: +(height - h).toFixed(2), w: +w.toFixed(2), h: +h.toFixed(2) };
  });

  return { bars, width, height, max };
}
