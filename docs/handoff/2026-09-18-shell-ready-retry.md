# 2026-09-18 — shell-ready-retry: the shell survives daemon restarts

Shipped: the owner's desktop app bricked during back-to-back deploys —
the shell parked on a Go 404 body forever. Two gates close the window:
Rust (`main_target` → `await_desktop_ready`, health.rs `status_ok`) waits
bounded for `/desktop/` to answer 200 before the shell's first paint
(all three build/rebuild paths), and the shared `reconnect.js` gates both
auto-reloads (downtime-up and fast bootId change) on this page's own path
answering 200 — bounded 20×750ms, fail-open, manual Reload button intact.
Verified: 3 new reconnect tests (deferred reload, fail-open, probe truth
table); `cargo xwin check --release --target x86_64-pc-windows-msvc`
clean; ci-scoped PASS. Visual: n/a (shell-native behavior; the owner
confirms the resident comes back healthy after `make desktop-restart`).
Merge: fast-forward ready.

## Next up

- Confirmed live by the owner (2026-09-18): after `make desktop-restart` the desktop app is healthy and showing the app. The fire test remains the next deploy — the auto-reload should wait out the restart window instead of bricking the shell.

## Debts

- docs/handoff/open/installed-webapps.md: standalone window stays refused (unchanged).
