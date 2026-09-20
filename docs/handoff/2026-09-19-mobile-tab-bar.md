# 2026-09-19 — mobile-tab-bar: dock tabs into the iOS letterbox

Shipped: Auto on a home-screen iPhone resolves to edge so the tab bar
sits on the physical bottom instead of floating above a blank strip.
Opaque status-bar measurement splits bar from strip (`gap/2`). Low is
the fallback if a label clips. Pushed screens still grow even on Low.

Verified: `make close` green. Scratch `tab-bar` 390×844 Work: overlayAudit
ok; with `--letterbox: 47px` + edge, `#m-app.bottom` is `-47px` and tabs
are `flex-end`. Blind spot: Chromium has no WebKit 313800 letterbox.

visual-review: PASS (mobile-work-tabbar-low.png + overlayAudit ok; card 5/5).

Merge: fast-forward ready (`feat/mobile-tab-bar`).
