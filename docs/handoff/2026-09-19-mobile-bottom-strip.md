# 2026-09-19 — mobile-bottom-strip: use the iOS letterbox on pushed screens

Shipped: a pushed mobile screen (terminal, agent) grows `#m-app` into the
WebKit standalone letterbox (`--letterbox` = `screen.height - innerHeight`).
Tab screens still paint the strip as the tab bar. "Screen edge" uses the
same variable instead of `env(safe-area-inset-top)` (0 under an opaque
status bar).

Verified: `make close` green. Scratch `bottom-strip` 390×844: overlayAudit
ok; with `data-standalone` + `--letterbox: 47px`, computed `#m-app.bottom`
is `-47px`. Blind spot: Chromium has no WebKit 313800 letterbox.

visual-review: PASS (mobile-term-letterbox.png + overlayAudit ok; card 5/5).

Merge: fast-forward ready (`feat/mobile-bottom-strip`).
