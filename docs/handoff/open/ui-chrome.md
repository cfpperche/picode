# App chrome

## Debts

- Fullscreen (2026-09-11): browser chords never reach a CLI; a revealed top strip covers the Canvas pill; the right strip misses an embedded frame; icon-only controls have no hover label.
- App-wide: the toast covers a surface's Close; a grow-resize leaves an idle cursor; Inspector branch chips truncate to `· fe… ·…`; a panel's Run reads as a disabled chip.
- Native surfaces (ADR-0109): `host` has no `openTerminal` — the Canvas POSTs `/api/terminals/{id}/open` itself; a tab closing under a panel remounts the body with a fresh xterm.
- App surfaces (2026-09-14, [parity plan](../../plans/app-surface-parity.md)): `internal/server/apps.go` answers **500** when an app view's entity is gone (`inbox: no view at …`, the store's `not found`) where 404 is the honest status — the host matches on the message too until it does; the blocked card ("needs a newer PiCode") is copy-only and has no screenshot; Inbox's empty lines ("No answered items yet.") name no next action, unlike tmux's.

## Notes

- Notices (2026-09-07): the finish card only fires for the agent whose socket is open.
