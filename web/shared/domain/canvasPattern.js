// Which texture the Canvas plane wears, per viewer.
//
// The plane is the one surface in the app that is mostly empty ground, so
// what that ground looks like is a reading preference, not a design
// constant: a grid helps someone lining panels up, dots stay out of the way,
// and plain is for whoever finds any texture noise. It follows theme.js
// exactly — localStorage, a safe default, and a window event so every open
// plane re-reads without a reload — because it is the same kind of choice.
//
// Scope: the Canvas plane and nothing else (docs/architecture/canvas.md).
// The rest of the app gets the theme's ground, never a texture.

const KEY = "picode-canvas-pattern";

// Every open plane listens; the preference page is not the only surface that
// has to agree with the value.
export const CANVAS_PATTERN_EVENT = "picode-canvas-pattern";

// "plain" first because it is the absence of the others, then the three
// React Flow draws. The order is the order the preference cards read in.
export const CANVAS_PATTERNS = Object.freeze(["plain", "dots", "lines", "cross"]);

// Dots is what the plane shipped with, so an existing reader who never opens
// the preference sees no change.
export const DEFAULT_CANVAS_PATTERN = "dots";

export function readCanvasPattern() {
  let raw = null;
  try {
    raw = localStorage.getItem(KEY);
  } catch {
    // Private windows and blocked site data throw on read; the default is
    // a correct answer, so a plane still draws.
    return DEFAULT_CANVAS_PATTERN;
  }
  return CANVAS_PATTERNS.includes(raw) ? raw : DEFAULT_CANVAS_PATTERN;
}

export function persistCanvasPattern(pattern) {
  const next = CANVAS_PATTERNS.includes(pattern) ? pattern : DEFAULT_CANVAS_PATTERN;
  try {
    localStorage.setItem(KEY, next);
  } catch {
    // Nothing to recover: the choice still applies to this page's planes.
  }
  if (typeof window !== "undefined") window.dispatchEvent(new Event(CANVAS_PATTERN_EVENT));
  return next;
}
