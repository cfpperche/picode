# 2026-09-21 — tray-menu-cleanup

Tray menu reorganized: product actions up front, labs tucked away.

## What changed

- Removed `New browser tab` (redundant with the work browser's new-tab) and
  `Test notification` (the Phase 1 notification spike).
- Promoted `Management…` up beside Open / Restart / View logs.
- Moved Browser lab + Computer lab under one `Labs ▸` submenu.
- All in `desktop-shell/src/main.rs` (menu construction + handler arms).

## Verification

- `make ci-scoped` green (fmt, vet, hooks, desktop-test, xwin).
- `cargo xwin check`: no new warnings.

## Next up

- Visual check on Windows (owner): tray shows
  Running · … / Open · Restart · View logs · Management… / Labs ▸ / Quit,
  and both labs open from the submenu.
