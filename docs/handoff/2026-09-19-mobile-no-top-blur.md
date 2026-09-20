# 2026-09-19 — mobile-no-top-blur: opaque iOS status bar

Shipped: mobile PWA status bar is `black` (dark) / `default` (light), not
`black-translucent`. iOS 26 Liquid Glass was frosting the sticky head
(title measured ~132 vs `--text-primary` 236). Heads still pad with
`env(safe-area-inset-top)`. Files: `web/mobile/index.html`, `App.jsx`,
`mobile.css`; `docs/architecture/routes.md`.

Verified: `make close` green. Scratch `no-top-blur` at 390×844 on
`#/term/shell-*`; overlayAudit ok; header title at full contrast on
Chromium. Blind spot: iOS 26 standalone PWA is not available on this
host; `apple-mobile-web-app-status-bar-style` is read when the icon is
added to the Home Screen.

visual-review: PASS (mobile-term-header.png + overlayAudit ok; card 5/5).
iOS Liquid Glass itself unverified here.

Merge: fast-forward ready (`feat/mobile-no-top-blur`).
