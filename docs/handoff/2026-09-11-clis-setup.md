# 2026-09-11 — feat/clis-setup: Settings, Packages and Connectors nest on the CLI
Shipped: Agent CLIs inner panes Settings / Packages / Connectors (MCP) sit next
to Launch / Terminals / Sessions / Providers. Webhooks stay `#/integrations/webhooks`.
Canonical hashes `#/clis/<cli>/{settings,packages,connectors}`. `go()` and menus
pass `workspaceId`/`agentId`; ConnectorsPane uses `loadPiPackagesContext`. Legacy
`#/packages`/`#/settings`/`#/mcps` wait for the fleet; `#/clis/connectors` is
reserved (rewrites to Pi); extra Settings path shows an invalid notice. ADRs
0101/0102/0075 amended for navigation only. Fragment `docs/changelog.d/clis-setup.md`.
Commits `50344246`, `e284eee7` (main merged as `995f9927`, `76cdb582`).
Verified: `make close` PASS (ci-scoped fmt,vet,hooks,test-js,build,docs). Scratch
http://localhost:8473 session `clis-setup-bugs`: Composer More and user-menu
Connectors keep `workspaceId`/`agentId`; pane shows Atlas · QA; `#/packages`
adopts the pane; `#/clis/connectors` rewrites to Pi; extra Settings path shows
the notice; overlayAudit ok on composer More and user menu. Screenshots
`var/screenshots/clis-setup-bugs/` (not committed).
visual-review: PASS
Not done / debts: OAuth callback HTML in `internal/server/mcp.go` still emits
`#/mcps` (rewrite saves the user). Mobile More still has no `onReloadAgent`.
Mobile `hasPackageUpdates` still desktop-only. Agent CLIs still absent from
docs-shots SURFACE_PROFILES.
Merge: fast-forward ready (main merged in as 76cdb582).
