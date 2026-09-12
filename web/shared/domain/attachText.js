// The attach composer's message field (TermAttachBar / TermAttachSheet): one
// line tall at rest — one control height (36px) — growing to four lines and
// then scrolling. Pure geometry, so the rule is testable under node; the DOM
// wrapper (`fitAttachField`) only measures the element and feeds this.

export const MAX_ATTACH_LINES = 4;

// Height in px for a textarea given its content and its chrome. The result
// is a border-box height (the CSS `height` / `min-height` the element uses);
// `scrollHeight` is a padding-box measurement, so the borders are added back.
// `minHeight` is the CSS floor (--ctl-h); one line is never cut below
// `lineHeight + padding + border` even when the CSS floor is absent.
export function attachFieldHeight({
  scrollHeight = 0,
  lineHeight = 0,
  padTop = 0,
  padBottom = 0,
  borderTop = 0,
  borderBottom = 0,
  minHeight = 0,
  maxLines = MAX_ATTACH_LINES,
} = {}) {
  const border = borderTop + borderBottom;
  const pad = padTop + padBottom;
  const line = Math.max(0, lineHeight);
  const lines = Math.max(1, maxLines);
  const one = line + pad + border;
  const max = line * lines + pad + border;
  const floor = Math.max(minHeight, one);
  return Math.max(floor, Math.min(scrollHeight + border, max));
}

// Enter sends, Shift+Enter keeps the newline (owner decision, 2026-09-12).
// A composing key never sends: `isComposing` and the legacy keyCode 229 that
// Safari and older Chromium report while an IME composition is open.
export function isAttachSendKey(event) {
  if (!event) return false;
  return event.key === "Enter"
    && !event.shiftKey
    && !event.isComposing
    && event.keyCode !== 229;
}

// DOM wrapper: measure, clamp, set. Height `auto` first so deleting text
// collapses the field again; overflow follows the clamp so a fifth line
// scrolls inside the field instead of pushing the terminal pane.
export function fitAttachField(el, maxLines = MAX_ATTACH_LINES) {
  if (!el || typeof window === "undefined" || !window.getComputedStyle) return;
  const s = window.getComputedStyle(el);
  const num = (v) => {
    const n = parseFloat(v);
    return Number.isFinite(n) ? n : 0;
  };
  const borderTop = num(s.borderTopWidth);
  const borderBottom = num(s.borderBottomWidth);
  el.style.height = "auto";
  const measured = el.scrollHeight;
  const next = attachFieldHeight({
    scrollHeight: measured,
    lineHeight: num(s.lineHeight),
    padTop: num(s.paddingTop),
    padBottom: num(s.paddingBottom),
    borderTop,
    borderBottom,
    minHeight: num(s.minHeight),
    maxLines,
  });
  el.style.height = next + "px";
  el.style.overflowY = measured + borderTop + borderBottom > next ? "auto" : "hidden";
}
