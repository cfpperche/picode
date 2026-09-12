// Whether the Canvas shows its own controls, per viewer.
//
// The plane carries three floating things a reader does not always want: the
// zoom column bottom-left, the toolbar in the middle and the minimap on the
// right. They are the way *around* a canvas, not the canvas — so when the
// canvas itself is the thing being read (a wall of live terminals, a
// screenshot, a demo), they are three boxes over the work. This is the one
// switch that takes all three away and brings them back, reached from the
// plane's own context menu.
//
// It follows canvasPattern.js exactly — localStorage, a safe default, a
// window event so every open plane re-reads — because it is the same kind of
// choice: a reading preference owned by the browser, never by the store. A
// camera is not an edit (ADR-0113) and neither is this.
//
// Scope: the Canvas surface and nothing else (docs/architecture/canvas.md).
// It never hides a panel, and it never hides the context menu that undoes
// it: right-clicking the plane works with the chrome gone, which is the only
// reason hiding everything is safe to offer at all.

const KEY = "picode-canvas-chrome";

export const CANVAS_CHROME_EVENT = "picode-canvas-chrome";

// Shown, because a reader who has never opened the menu must not have to
// find it to get their controls back.
export const DEFAULT_CANVAS_CHROME = true;

export function readCanvasChrome() {
  let raw = null;
  try {
    raw = localStorage.getItem(KEY);
  } catch {
    // Private windows and blocked site data throw on read; showing the
    // controls is a correct answer, so the plane stays usable.
    return DEFAULT_CANVAS_CHROME;
  }
  // Only the string this module writes counts as "hidden". Anything else —
  // absent, stale, someone else's value — is the default.
  return raw === "hidden" ? false : DEFAULT_CANVAS_CHROME;
}

export function persistCanvasChrome(shown) {
  const next = !!shown;
  try {
    localStorage.setItem(KEY, next ? "shown" : "hidden");
  } catch {
    // Nothing to recover: the choice still applies to this page's planes.
  }
  if (typeof window !== "undefined") window.dispatchEvent(new Event(CANVAS_CHROME_EVENT));
  return next;
}
