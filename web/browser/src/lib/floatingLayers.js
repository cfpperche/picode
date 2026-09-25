// Floating-layer geometry: which HTML layers sit over a rectangle. The shell
// keeps the native page live beneath them (nativeLayers.js, ADR-0161); the
// hide-the-page subscription for older shells was retired 2026-09-25.
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
