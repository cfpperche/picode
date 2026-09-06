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

// Phase 2 — what the strip shows around its content. `overflow` decides
// whether arrows and the indicator exist at all; `atStart` / `atEnd`
// disable one arrow and drop one edge fade (Firefox `scrolledtostart` /
// `scrolledtoend`); `thumb` is the overlay indicator's box as fractions
// of the strip's width. A 1 px tolerance absorbs fractional layout.
export function stripState({ scrollLeft, clientWidth, scrollWidth }) {
  const overflow = scrollWidth - clientWidth > 1;
  if (!overflow) return { overflow, atStart: true, atEnd: true, thumb: { left: 0, width: 1 } };
  const max = scrollWidth - clientWidth;
  return {
    overflow,
    atStart: scrollLeft <= 1,
    atEnd: scrollLeft >= max - 1,
    thumb: { left: scrollLeft / scrollWidth, width: clientWidth / scrollWidth },
  };
}

// A vertical wheel over the strip scrolls it sideways (VS Code
// `scrollYToX`, Firefox arrowscrollbox, Ant dominant axis). Returns the
// horizontal distance in px, or 0 when the event is not ours: a trackpad
// already sending deltaX (the browser scrolls natively), a pinch
// (ctrlKey), or a gesture that is mostly horizontal. Line and page deltas
// are scaled the way Firefox does before reaching the scroller.
export function wheelToScroll({ deltaX = 0, deltaY = 0, deltaMode = 0, ctrlKey = false }, clientWidth) {
  if (ctrlKey || deltaX !== 0 || deltaY === 0) return 0;
  if (Math.abs(deltaY) <= Math.abs(deltaX)) return 0;
  if (deltaMode === 1) return deltaY * 40;
  if (deltaMode === 2) return deltaY * clientWidth;
  return deltaY;
}

// One arrow press moves most of a viewport so a tab clipped at the far
// edge lands well inside it (MUI scrolls a full width; Firefox one tab).
export function arrowStep(clientWidth) {
  return Math.max(80, Math.round(clientWidth * 0.6));
}
