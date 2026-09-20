// Legacy shells: when must the work browser get out of the way?
// Current shells use nativeLayers.js (ADR-0161); rectOf remains shared.
//
// A WebView2 is a native child window: it paints over every HTML pixel in the
// same region, whatever the z-index says. So an HTML layer that intersects the
// browser's rectangle is invisible until the native view hides — the owner's
// tab-strip menu, the command palette, a toast, any dropdown (2026-09-16: the
// list keeps growing, which is why the rule is geometry, not a hand-picked set
// of "the overlays we remembered").
//
// The layer vocabulary itself is shared with the clipping audit
// (web/shared/domain/overlayAudit.js): one list, two consumers, so a new
// floating surface is seen by both or by neither.
//
// What is deliberately NOT a layer:
//   - tooltips: hiding the page every time a tooltip crosses it makes the
//     surface flicker for a hint nobody needs to read over the page;
//   - the focus-mode edge strips: they are always on, so they would hide the
//     page for as long as focus mode lasts (see the topic's debts).

import { OVERLAY_SELECTORS } from "@picode/shared/domain/overlayAudit.js";

// rectOf is the geometry this module decides on: a viewport rectangle in the
// same coordinate space as the browser host's.
export function rectOf(el, win = globalThis) {
  if (!el || typeof el.getBoundingClientRect !== "function") return null;
  const style = win.getComputedStyle ? win.getComputedStyle(el) : null;
  if (style && (style.display === "none" || style.visibility === "hidden")) return null;
  const r = el.getBoundingClientRect();
  if (!r || r.width < 2 || r.height < 2) return null;
  const vw = win.innerWidth || 0;
  const vh = win.innerHeight || 0;
  // Fully off-screen chrome is not over the page (the focus-mode sidebar
  // parked outside the viewport, a menu mid-transition): ignore it.
  if (vw && vh && (r.right < 1 || r.bottom < 1 || r.left > vw - 1 || r.top > vh - 1)) return null;
  return { left: r.left, top: r.top, right: r.right, bottom: r.bottom };
}

export function overlaps(a, b) {
  if (!a || !b) return false;
  return a.left < b.right && b.left < a.right && a.top < b.bottom && b.top < a.bottom;
}

// layerRects collects every visible floating layer, in viewport coordinates.
export function layerRects(doc = globalThis.document, win = globalThis) {
  if (!doc || !doc.querySelectorAll) return [];
  const out = [];
  for (const sel of OVERLAY_SELECTORS) {
    for (const el of doc.querySelectorAll(sel)) {
      const rect = rectOf(el, win);
      if (rect) out.push({ ...rect, sel });
    }
  }
  return out;
}

// overlapsLayers answers the tab's question: does anything visible sit over
// this rectangle right now?
export function overlapsLayers(rect, layers = []) {
  return layers.some((layer) => overlaps(rect, layer));
}

// subscribeFloatingLayers calls back with the current layers whenever the DOM
// or the viewport changes.
//
// Coalesced with a floor between scans: the app mutates constantly while an
// agent streams, and a full-DOM scan per mutation would cost more than the
// page it protects. ~100 ms is below noticing for a menu and cheap enough to
// run through a stream.
const MIN_SCAN_GAP_MS = 100;

export function subscribeFloatingLayers(fn, { doc = globalThis.document, win = globalThis, gapMs = MIN_SCAN_GAP_MS } = {}) {
  if (!doc || typeof fn !== "function") return () => {};
  const now = () => (typeof win.performance?.now === "function" ? win.performance.now() : Date.now());
  let queued = 0;
  let last = 0;
  const run = () => {
    queued = 0;
    last = now();
    fn(layerRects(doc, win));
  };
  const schedule = () => {
    if (queued) return;
    // A hidden document cannot have a layer over the page, and nothing will
    // be painted until it comes back — scan then.
    if (doc.hidden) return;
    const wait = Math.max(0, gapMs - (now() - last));
    queued = win.setTimeout ? win.setTimeout(run, wait) : 1;
  };
  const observer = new win.MutationObserver(schedule);
  // data-state covers Radix's open/closed, style/class cover the app's own
  // layers, childList covers a portal landing in the body.
  observer.observe(doc.body || doc.documentElement, {
    childList: true,
    subtree: true,
    attributes: true,
    attributeFilter: ["data-state", "class", "style", "hidden"],
  });
  win.addEventListener("resize", schedule);
  win.addEventListener("scroll", schedule, true);
  doc.addEventListener?.("visibilitychange", schedule);
  schedule();
  return () => {
    observer.disconnect();
    win.removeEventListener("resize", schedule);
    win.removeEventListener("scroll", schedule, true);
    doc.removeEventListener?.("visibilitychange", schedule);
    if (!queued) return;
    if (win.clearTimeout) win.clearTimeout(queued);
    queued = 0;
  };
}
