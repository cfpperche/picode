import { simplifyLines } from "./layout.js";

// Mobile commit subjects wrap. Map the existing DAG's row indices to the
// measured centers, so a long subject never detaches its dot from its row.
export function rowGeometry(heights) {
  let height = 0;
  const centers = heights.map(value => { const size = Math.max(1, Number(value) || 1); const center = height + size / 2; height += size; return center; });
  return { centers, height };
}
export function measuredBranchPath(lines, { centers, height }, step = 12, offset = 10) {
  const y = i => centers[i] ?? height + 12;
  let path = "";
  let previous = null;
  for (const line of simplifyLines(lines)) {
    const x1 = line.p1.x * step + offset, x2 = line.p2.x * step + offset;
    const y1 = y(line.p1.y), y2 = y(line.p2.y);
    if (!previous || previous.x !== x1 || previous.y !== y1) path += `M${x1},${y1}`;
    const bend = Math.min(24, Math.abs(y2 - y1) / 2);
    path += x1 === x2 ? `L${x2},${y2}` : `C${x1},${y1 + bend} ${x2},${y2 - bend} ${x2},${y2}`;
    previous = { x: x2, y: y2 };
  }
  return path;
}
