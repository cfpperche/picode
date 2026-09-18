# Installed webapps (ADR-0147)

Topic record for the user-installed webapps arc (shipped 2026-09-17/18;
branches `feat/installed-webapps`, `feat/webapp-head-truncate`,
`feat/webapp-pwa`, `feat/webapp-chromeless`, `feat/webapp-appmode-fullbleed`).

## Paid

- [x] Shell checks closed by the owner's call (2026-09-18): tile click opens the webapp in the shell with its login; login persisted across tile reinstall and shell reloads in live use; the `(N)` badge ships unit-tested (title parse) and was accepted without a separate shell screenshot.
- [x] Full-bleed app mode (no titlebar) confirmed live by the owner.

## Debts

- [ ] Standalone window (Chrome-style, its own taskbar icon) stays refused while the owner works in-shell; revisiting costs an ADR that re-opens the btab plumbing question.
