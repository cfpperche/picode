# 2026-09-06 — extra-keys overlay scrollbar

Owner screenshot: a gray vertical strip on the right while dragging the
extra-keys row sideways. iOS overlay scrollbar on the terminal screen
(`.m-screen` still had `-webkit-overflow-scrolling: touch`).

Fix: `touch-action: pan-x` on the row, overflow hidden on the bar,
overlay scrollbars off on the screen, the scroller and xterm viewport.
