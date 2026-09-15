# 2026-09-14 — feat/docs-refresh: refresh stale README and architecture claims
Shipped: one commit `docs: refresh stale README and architecture claims` (565f52a6 on a596a6cb), 9 files 34+/20-: AGENTS.md, README.md, docs/architecture.md, docs/architecture/{broker,direct-session-communication,notices,routes}.md, docs/guidelines.md, docs/changelog.d/docs-refresh.md.
Verified: confirmed the 4-agent docs-vs-code audit's P1/P2 findings against code (CLI catalog internal/clilaunch/config.go, MCP server internal/communication/mcp.go, canvas routes internal/server/canvas.go, shell paths web/shared/client/shell.js + web/tools/vite-config.mjs, close.sh order, dialog/toast/notice paths under web/browser) and applied minimal patches; re-verified two audit-flagged lists as correct and left them untouched (agent-manager session index, presence/activity-scoped CLI lists).
visual-review: n/a
Merge: fast-forward ready.

## Debts

- P3 follow-up: add short docs/architecture sections for work-browser internal/browser/hub.go and the Tauri shell Makefile target.
- Unaudited remainder: ADR bodies, api.md/commands.md, 29/33 docs-site guides, macOS/Docker/upgrade guides, external URLs, app-fleet.png freshness.
