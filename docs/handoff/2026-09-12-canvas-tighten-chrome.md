# 2026-09-12 — an empty canvas draws nothing

`feat/canvas-tighten-chrome` → `main`. Two small owner calls, both in the
canvas chrome.

- `.cv-blank` and its three CSS rules are gone: an empty canvas is the plane
  and the toolbar, nothing else.
- `.cv-cluster > .cv-switch` loses `min-width: 104px`. Measured: 81 px for
  "aaa" against 104 before. The floor was load-bearing only while the group
  was centred; it is top-left now and grows to the right. `max-width: 240px`
  stays so a long name ellipsizes instead of pushing the `⋯` off the pane.

## Watch for

If the switcher ever moves back to a centred group, the floor has to come
back with it — otherwise every rename shifts every button beside it.
