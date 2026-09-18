# Installed webapps (ADR-0147)

Open items accepted or deferred when the user-installed webapps shipped
(2026-09-17, branch `feat/installed-webapps`).

## Next

- Remaining shell checks: the `(N)` title badge renders on the tile while the webapp tab is open, and a webapp login survives a shell restart via the shared WebView2 profile. (Tile click opens the webapp in the shell with its login — confirmed by the owner live, 2026-09-18.)
- Per-webapp WebView2 partitions (two accounts of one service) — v2 item from ADR-0147; revisit with the standalone-window PWA idea (`display: standalone`).
