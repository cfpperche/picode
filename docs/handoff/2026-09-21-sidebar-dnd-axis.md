# 2026-09-21 — feat/sidebar-dnd-axis: vertical drag stays in the column

The sidebar reorder locked to the vertical axis (`restrictToVerticalAxis`, the dnd-kit modifier) and the row transform drops x and scale. Autoscroll no longer uses a zero horizontal threshold (that divides by zero and scrolls sideways) and does not compensate layout shift on x. `.side-scroll` clips overflow-x so a leftover translate cannot open a bar.

## Next up

- None.
