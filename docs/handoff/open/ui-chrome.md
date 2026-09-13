# App chrome

## Debts

- Fullscreen (2026-09-11): the browser's reserved chords (Ctrl+T…) never reach a CLI; a revealed top strip covers the Canvas chrome pill; the right strip's reveal misses an embedded frame (PDF); icon-only controls have no hover-free label.
- Notices (2026-09-07): the finish card only fires for the agent whose socket is open.
- App-wide: the toast covers a surface's Close; a grow-resize leaves an idle cursor; Inspector branch chips truncate to `· fe… ·…`; a panel's Run reads as a disabled chip.
- Native surfaces (ADR-0109): `host` has no `openTerminal` — the Canvas POSTs `/api/terminals/{id}/open` itself; a tab closing under a panel remounts the body with a fresh xterm.
- Fetches (2026-09-13): with a file, tree, git or app tab selected the shell still asks `/api/agents/<tab id>/role-state` and `/slash` — the guards only exclude terminal tabs (`App.jsx` `fetchRoleState` and the slash effect), so each such selection costs two 404s. Pre-existing, harmless (both `.catch`), noisy.
