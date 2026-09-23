# 2026-09-23 — terminal-link-boundary: trim link punctuation

Shipped: Terminal link scanning excludes surrounding Markdown punctuation, including a trailing `)`; documented in `docs/architecture/devservers.md` with regression cases in `termLinks.test.js`.
Verified: `make web`, `make ci-scoped`, and `make close` passed after the regression commit. A scratch instance on `/desktop` opened the exact URL on Ctrl+click without the trailing `)`; screenshot review passed and `window.__picodeOverlayAudit()` returned `ok`.
visual-review: PASS (scratch browser screenshot).
Limit: A plain browser check does not prove native WebView2 rendering of the remote page.
Merge: fast-forward ready.
