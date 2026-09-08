# 2026-09-08 — feat/mobile-standalone-fix: mobile tab pill + iOS standalone bottom strip
Shipped: the active-tab pill wraps icon + label (was icon-only, its edge cut
the label). iOS standalone (home-screen install) now routes every bottom
inset through `--sa-bottom`, zeroed there: WebKit parks an unreachable
status-bar-sized strip below the layout viewport and still reports
env(bottom) inside it (WebKit 313800/254868) — ~81pt of dead space. The
canvas is painted so the strip continues the surface above it (tab-bar
panel on tabs, content background when pushed, via `html[data-pushed]`);
flag set at bootstrap from `navigator.standalone`.
Verified: `make ci-scoped` + `make close` PASS; scratch instance 390×844,
light + dark, tab and pushed screens, pill geometry asserted
(label inside pill), overlayAudit ok.
visual-review: PASS (mobilefix-tabbar-*.png, mobilefix-pushed-dark.png — read)
Not done / debts: real-device re-check on the owner's iPhone (iOS 26/27
standalone) after the next deploy — Chromium cannot reproduce the WebKit
letterbox; the 47px strip was simulated via `#m-app{bottom:47px}`. If
Apple ever fixes 313800, `--sa-bottom: 0px` is the one-line revert.
Merge: fast-forward ready.
