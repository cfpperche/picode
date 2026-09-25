// The app's vocabulary of floating layers — everything that can paint over the
// content well. Two consumers, one list, on purpose:
//
//   - `overlayAudit` checks them for clipping (the visual-review gate);
//   - the shell uses the same vocabulary for native chrome regions above the
//     live page (`web/browser/src/lib/nativeLayers.js`, ADR-0161).
//
// A new floating surface belongs here, or both checks lose sight of it.
//
// Toasts are matched per item (`[data-sonner-toast]`), not by sonner's
// container: the container stays mounted with a box of its own, and an
// always-present layer would keep the work browser hidden forever.
export const OVERLAY_SELECTORS = [
  // Radix's own semantics first: role + data-state catches every portal the
  // app opens, whatever class it wears. The palette's `.palette` (a raw Radix
  // Dialog, not the app's `.dlg` wrapper) is the case that proved the point —
  // a class-only list had no idea it was there (2026-09-16).
  "[data-radix-popper-content-wrapper]",
  '[role="dialog"][data-state="open"]',
  '[role="alertdialog"][data-state="open"]',
  '[role="menu"][data-state="open"]',
  '[role="listbox"][data-state="open"]',
  // The app's own layers, which are not Radix content.
  ".cockpit-pop",
  ".session-pop",
  ".slash-menu",
  ".toast",
  "[data-sonner-toast]",
  ".dlg",
  ".create-drawer",
  ".rail-pop",
  ".pkg-job",
  ".pkg-job-card",
  ".img-lite",
  "#inspector",
];

const SELECTORS = OVERLAY_SELECTORS;

export function overlayAudit(win = globalThis) {
  const doc = win.document;
  const vh = win.innerHeight || 0;
  const vw = win.innerWidth || 0;
  const hits = [];
  for (const sel of SELECTORS) {
    for (const el of doc.querySelectorAll(sel)) {
      const s = win.getComputedStyle(el);
      if (s.display === "none" || s.visibility === "hidden") continue;
      const r = el.getBoundingClientRect();
      if (r.width < 2 || r.height < 2) continue;
      hits.push({
        sel,
        clipTop: r.top < -1,
        clipBottom: r.bottom > vh + 1,
        clipLeft: r.left < -1,
        clipRight: r.right > vw + 1,
        top: Math.round(r.top),
        bottom: Math.round(r.bottom),
        height: Math.round(r.height),
      });
    }
  }
  // A floating layer needs an acknowledged native chrome region.
  const uncovered = [];
  for (const host of doc.querySelectorAll(".web-tab-host")) {
    const hs = win.getComputedStyle(host);
    if (hs.display === "none" || hs.visibility === "hidden") continue;
    const r = host.getBoundingClientRect();
    if (r.width < 2 || r.height < 2) continue;
    if (host.getAttribute?.("data-native-layers-ready") === "true") continue;
    if (layerRectsOver(doc, win).some((layer) => intersects(r, layer))) {
      uncovered.push({ sel: ".web-tab-host", top: Math.round(r.top), bottom: Math.round(r.bottom) });
    }
  }

  const rows = [];
  for (const row of doc.querySelectorAll("[data-align-row]")) {
    const rs = [];
    for (const el of row.children) {
      const s = win.getComputedStyle(el);
      if (s.display === "none" || s.visibility === "hidden") continue;
      const r = el.getBoundingClientRect();
      if (r.width < 2 || r.height < 2) continue;
      rs.push({ top: r.top, height: r.height });
    }
    if (rs.length < 2) continue;
    // Wrapped controls still share a height and align within each visual line.
    // A label beside a multi-line control group is not itself a control row.
    const wraps = row.hasAttribute?.("data-align-wrap");
    const misaligned = rs.some((r) => Math.abs(r.height - rs[0].height) > 1)
      || (wraps
        ? rs.some((r, i) => rs.slice(i + 1).some((other) => Math.abs(r.top - other.top) > 1
          && Math.max(r.top, other.top) < Math.min(r.top + r.height, other.top + other.height)))
        : rs.some((r) => Math.abs(r.top - rs[0].top) > 1));
    rows.push({
      misaligned,
      heights: rs.map((r) => Math.round(r.height)),
      tops: rs.map((r) => Math.round(r.top)),
    });
  }
  return {
    ok: hits.every((h) => !h.clipTop && !h.clipBottom && !h.clipLeft && !h.clipRight)
      && rows.every((r) => !r.misaligned)
      && uncovered.length === 0,
    hits,
    rows,
    uncovered,
  };
}

// layerRectsOver is the geometry half of the same vocabulary, kept local so
// this module stays dependency-free for the mobile/tests harness.
function layerRectsOver(doc, win) {
  const out = [];
  for (const sel of SELECTORS) {
    for (const el of doc.querySelectorAll(sel)) {
      const s = win.getComputedStyle(el);
      if (s.display === "none" || s.visibility === "hidden") continue;
      const r = el.getBoundingClientRect();
      if (r.width < 2 || r.height < 2) continue;
      out.push({ left: r.left, top: r.top, right: r.right, bottom: r.bottom });
    }
  }
  return out;
}

function intersects(a, b) {
  return a.left < b.right && b.left < a.right && a.top < b.bottom && b.top < a.bottom;
}
