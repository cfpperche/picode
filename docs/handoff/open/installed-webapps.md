# Installed webapps (ADR-0147)

Open items accepted or deferred when the user-installed webapps shipped
(2026-09-17, branch `feat/installed-webapps`).

## Next

- Windows-shell verification of the desktop-only paths: tile click focuses or reopens the `w:app-<id>` work tab, `(N)` title badge renders on the tile, and a webapp login survives a shell restart via the shared WebView2 profile.
- Per-webapp WebView2 partitions (two accounts of one service) — v2 item from ADR-0147; revisit with the standalone-window PWA idea (`display: standalone`).
