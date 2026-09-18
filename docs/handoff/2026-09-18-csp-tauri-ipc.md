# 2026-09-18 — feat/csp-tauri-ipc: Tauri IPC on the desktop-shell CSP

Shipped: `/desktop/` and `/desktop/management.html` `connect-src` now names
Tauri 2's IPC (`ipc: http://ipc.localhost https://ipc.localhost`) so WebView2
`invoke()` works again. Browser and mobile shells keep the narrower policy.
ADR-0052 amendment; `appCSP` in `internal/server/csp.go`.
Verified: `make close` green; `TestDesktopShellAllowsTauriIPC` and
`TestDesktopShellPath` (desktop vs `/`, `/browser/`, `/mobile/`, and
`/desktop-evil`). Blind spot: not run inside the Windows shell.
visual-review: n/a
Merge: fast-forward ready.
