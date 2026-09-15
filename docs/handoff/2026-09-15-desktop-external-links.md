# 2026-09-15 — feat/desktop-external-links: route _blank links to the system browser
Shipped: web/browser/src/lib/externalLinks.js bridge, wired in bootstrap.jsx boot() for /browser/ and /desktop/ (b4bb4e74).
In the Tauri shell, _blank anchor clicks and _blank window.open for cross-origin http(s) URLs route to btab_open_external; real browsers install nothing.
Fixes the dead Documentation menu item and every other _blank chrome link (setup guides, changelog, repo, Open in browser, terminal links) in the desktop app.
Verified: make ci-scoped PASS, make web build ok, 8/8 node tests in externalLinks.test.js; scratch instance verified then stopped.
visual-review: PASS (var/screenshots/usermenu-docs.png + overlayAudit ok, card 5/5)
Merge: fast-forward ready.

## Next up

- none — branch ready to merge.

## Debts

- MCP named-window OAuth popup (window.open 'picode-mcp-auth') still uses the native path and stays dead in the shell — needs its own auth-flow verification before routing.
