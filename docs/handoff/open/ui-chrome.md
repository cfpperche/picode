# App chrome

## Debts

- Fullscreen (2026-09-11): browser-reserved chords (Ctrl+T…) never reach a CLI; a revealed top strip covers the Canvas pill; the right strip misses an embedded frame (PDF); icon-only controls have no hover label.
- Notices (2026-09-07): the finish card only fires for the agent whose socket is open.
- App-wide: the toast covers a surface's Close; a grow-resize leaves an idle cursor; Inspector branch chips truncate to `· fe… ·…`; a panel's Run reads as a disabled chip.
- Native surfaces (ADR-0109): `host` has no `openTerminal` — the Canvas POSTs `/api/terminals/{id}/open` itself; a tab closing under a panel remounts the body with a fresh xterm.
- Fetches (2026-09-13): selecting a file/tree/git/app tab still asks `/api/agents/<id>/role-state` and `/slash` — only terminal tabs are excluded (`App.jsx`), two caught 404s per selection. Pre-existing.
