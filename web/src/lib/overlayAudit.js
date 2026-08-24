const SELECTORS = [
  "[data-radix-popper-content-wrapper]",
  ".cockpit-pop",
  ".session-pop",
  ".slash-menu",
  ".toast",
];

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
  return {
    ok: hits.every((h) => !h.clipTop && !h.clipBottom && !h.clipLeft && !h.clipRight),
    hits,
  };
}
