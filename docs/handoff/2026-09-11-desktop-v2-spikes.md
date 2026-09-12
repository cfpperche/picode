# 2026-09-11 — feat/desktop-v2-spikes: notification spike + spike evidence

Shipped: `tauri-plugin-notification` in the shell with a tray item **Test
notification** (fires a native toast; on Windows, unpackaged toasts need a
Start Menu shortcut with the app identity — a silent drop is the installer
requirement for Phase 2, documented in the plan). Spike statuses recorded in
`docs/plans/desktop-v2.md`: SSE verified live server-side (`GET /api/events`
hello event); mkcert, pairing, xterm/WebGL are owner-visual checks listed in
the plan table.

Verified: `cargo xwin build` green with the plugin (6.09 MB exe, installed to
`AppData\Local\PicodeShell` and relaunched, PID confirmed); `make ci-scoped`
PASS (full) — first run hit the known cold Go test-cache flake, rerun green.

visual-review: n/a (Go repo); the toast and the window are owner-visual.

Debts: three owner-visual spikes open (pairing, mkcert, xterm/WebGL); toast
shortcut creation belongs to the Phase 2 installer.

Merge: fast-forward ready.
