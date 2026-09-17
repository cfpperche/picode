Task: Align sidebar scrollbar flush to the right edge of the sidebar panel.

Root cause: commit 41bc548 introduced `.side-scroll` inside `.side-section` to pin the header, but left `padding: 10px 8px` on `.side-section`. The scroller ended 8px before the right edge, causing its scrollbar to float inside the card area.

Shipped:
- `.side-section`: updated padding to `10px 0 0` so child scrollers can span full width to the right border.
- `.side-scroll`: added `padding: 0 8px 10px` so rows keep their 8px gutter while the scrollbar sits flush at `right: 0`.
- `.pins-head`: updated padding to `2px 8px 8px` to preserve header horizontal alignment.
- `.pins-pane` & `.pins-search`: adjusted padding/margins to preserve 8px gutters across Pins and Apps tabs.
- Verified `make ci-scoped` passes clean; changelog fragment added.
