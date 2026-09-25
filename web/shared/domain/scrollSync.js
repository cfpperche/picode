// Scroll sync between a source editor and its rendered preview (the Split
// view), after VS Code's markdown preview: the preview's blocks carry the
// source line they start on, and a position on either side is interpolated
// between the two blocks around it. Lines are fractional (12.5 = halfway
// through line 12); `top` is a block's offset inside the preview's scroll
// content.

// syncBlocks orders measured blocks by position and keeps only the ones whose
// line moves forward, so the interpolation never runs backwards (a nested
// block can start on its parent's line, raw HTML can carry odd positions).
export function syncBlocks(entries) {
  const sorted = [...entries].filter((b) => Number.isFinite(b.line) && Number.isFinite(b.top)).sort((a, b) => a.top - b.top || a.line - b.line);
  const out = [];
  for (const b of sorted) {
    if (out.length && b.line <= out[out.length - 1].line) continue;
    out.push({ line: b.line, top: b.top, height: Math.max(0, b.height || 0) });
  }
  return out;
}

// previewTopFor answers "where in the preview is source line `line`".
export function previewTopFor(blocks, line) {
  if (!blocks.length) return 0;
  const first = blocks[0];
  if (line <= first.line) return first.line > 1 ? first.top * Math.max(0, line - 1) / (first.line - 1) : first.top;
  let i = 0;
  while (i + 1 < blocks.length && blocks[i + 1].line <= line) i++;
  const prev = blocks[i];
  const next = blocks[i + 1];
  if (!next) return prev.top + Math.min(1, line - prev.line) * prev.height;
  return prev.top + (line - prev.line) / (next.line - prev.line) * (next.top - prev.top);
}

// lineFor is the inverse: the source line at preview offset `top`. `lastLine`
// bounds the tail after the final block.
export function lineFor(blocks, top, lastLine = Infinity) {
  if (!blocks.length) return 1;
  const first = blocks[0];
  if (top <= first.top) return first.top > 0 ? 1 + (first.line - 1) * Math.max(0, top) / first.top : first.line;
  let i = 0;
  while (i + 1 < blocks.length && blocks[i + 1].top <= top) i++;
  const prev = blocks[i];
  const next = blocks[i + 1];
  if (next) return prev.line + (top - prev.top) / (next.top - prev.top) * (next.line - prev.line);
  const span = Math.max(1, Math.min(lastLine, prev.line + 1) - prev.line);
  return prev.line + Math.min(1, (top - prev.top) / Math.max(1, prev.height)) * span;
}
