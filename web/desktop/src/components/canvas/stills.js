import { terms } from "../../lib/terms.js";

// Stills (plan docs/plans/matrix-canvas.md §4.3). Below the zoom band a
// canvas panel stops being a live terminal and becomes the *text* of its
// last screen: `term.buffer.active`, read straight out of the xterm the
// panel was showing. C0 measured a full 83 × 26 screen at **0.02 ms**
// (docs/benchmarks/2026-09-10-node-canvas.md), so the capture is free and
// the saving is the whole point — nine live `top` panes cost 4–14 % of a
// core, the same page at zoom 0.2 with 500 stills cost 0.0 %.
//
// Two rules the spike paid for:
//
//   * **Capture before the flip, never at unmount.** React renders the new
//     (still) body before it runs the live body's cleanup, so a capture in
//     the cleanup is always one crossing late — C0's first crossing painted
//     an empty still. The chunk loader calls `captureStills` with everything
//     that is about to stop being live, before it emits.
//   * **Memory only.** A still is never persisted and never leaves the tab:
//     it is a screen, and a screen belongs to the session that drew it.
//
// The map is module-level for the same reason `terms` is: one desktop, one
// set of xterm instances, one set of their last screens.

const stills = new Map(); // terminal or agent id → { text, at }

// captureStill(ref) -> the entry it stored, or null when there is no live
// xterm for that id (never loaded, or already disposed).
export function captureStill(ref) {
  const entry = ref ? terms.get("sh:" + ref) : null;
  const term = entry && entry.term;
  const buf = term && term.buffer && term.buffer.active;
  if (!buf || typeof buf.getLine !== "function") return null;
  const rows = term.rows || 0;
  const lines = [];
  for (let i = 0; i < rows; i++) {
    const line = buf.getLine(buf.viewportY + i);
    // trimRight: a terminal pads every row to its width with spaces.
    lines.push(line ? line.translateToString(true) : "");
  }
  while (lines.length && !lines[lines.length - 1]) lines.pop();
  const still = { text: lines.join("\n"), at: Date.now() };
  stills.set(ref, still);
  return still;
}

// captureStills(refs): the whole crossing in one pass.
export function captureStills(refs) {
  for (const ref of refs || []) captureStill(ref);
}

// readStill(ref) -> { text, at } | null.
export function readStill(ref) {
  return stills.get(ref) || null;
}

// forgetStill(ref): the panel left the canvas, or its target was deleted.
export function forgetStill(ref) {
  stills.delete(ref);
}

// QA hook, the pattern of window.__picodeTerms: what the canvas is holding.
if (typeof window !== "undefined") {
  window.__picodeCanvasStills = () => [...stills].map(([id, s]) => ({ id, at: s.at, len: s.text.length }));
}
