// The tab strip scrolls horizontally and never shows a layout scrollbar
// (docs/benchmarks/2026-09-06-tab-strip-overflow.md), so the active tab
// has to be brought into view by code — Firefox `scrollIntoView(nearest)`,
// VS Code `doLayoutTabsNonWrapping`. The strip moves as little as possible:
// a tab clipped on the right is aligned to the right edge, one clipped on
// the left to the left edge, and a visible tab leaves the offset alone.
//
// `strip` is the scroll box ({ scrollLeft, clientWidth, scrollWidth }) and
// `tab` the tab's box relative to the strip's content ({ left, width }).
// Returns the scrollLeft that reveals the tab, or null when nothing moves.
export function revealLeft(strip, tab) {
  const { scrollLeft, clientWidth, scrollWidth } = strip;
  const { left, width } = tab;
  const max = Math.max(0, scrollWidth - clientWidth);
  const clamp = (x) => Math.min(max, Math.max(0, x));
  // Wider than the viewport: its start is what the user can act on.
  if (width >= clientWidth) return clamp(left) === scrollLeft ? null : clamp(left);
  if (left < scrollLeft) return clamp(left);
  const right = left + width;
  if (right > scrollLeft + clientWidth) return clamp(right - clientWidth);
  return null;
}
