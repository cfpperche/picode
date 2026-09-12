# Desktop v2 — the Rust/Tauri shell (ADR-0120)

Status: **Phase 1 in build** (shell skeleton). Owner approved the direction on
2026-09-11; this document is the canonical version of the approved plan.

## Vision

A work application for Windows: an own window (Tauri 2 + WebView2) rendering
the PiCode UI served by the daemon in WSL, a tray with status and actions, and
a browsing context isolated from the personal browser — where **human and
agents** act, through CDP. The Go daemon in WSL remains the single source of
truth; the shell is a client and a supervisor, never a second backend.

## Phases

- **Phase 1 — skeleton (this branch):** window loading the address from
  `server.json` (discovered through `wsl.exe`), tray (Open/Quit),
  single-instance, offline fallback. Windows cross-build from WSL
  (`cargo-xwin`, `x86_64-pc-windows-msvc`). Spike-gate before calling it done:
  pairing in the embedded profile, mkcert, xterm.js/WebGL, SSE, native
  notification.
- **Phase 2 — Storage app + migrated actions:** the WSL management page (disk
  from both sides, give back, reviewed cache prunes) inside the window; the
  Windows-facts route reported by the app; keepalive/watchdog migrate from Go.
- **Phase 3 — Work Browser + CDP:** Chrome/Edge launched by the app (own
  `user-data-dir`, `--remote-debugging-pipe` — no open port); agents reach
  pages through CDP exposed to the daemon over the authenticated channel; the
  **browser policy** (domains × actions per agent) is visible and revocable in
  the UI. `ext/` + `browserhost` + ADR-0043 deprecated here.
- **Phase 4 — conditional:** embedded CEF (our tabs/omnibox, pinned engine,
  human and agent on the same tab) only if sharing the tab is a hard
  requirement.

## Decisions taken

- **Tauri 2 + WebView2** (ADR-0120): the Chromium engine, Microsoft's
  Evergreen runtime, Rust bindings; CEF is deferred to Phase 4.
- **CDP instead of the extension**: the extension existed because Chrome
  blocks CDP on the user's default profile; in a browser the product launches
  and owns, CDP over a pipe covers navigate/read/act with no open port. The
  extension's trust model (visible, scoped, revocable) becomes shell policy.
- **Cross-build from WSL** with `cargo-xwin`; the default CI gates do not
  require Rust (`make desktop-shell` is a dedicated target).

## Phase 1 risks / spikes

| Risk | Plan |
|---|---|
| Google blocks login in embedded views | Try Gmail in the embedded profile; workaround: UA override or external login |
| xterm.js/WebGL in WebView2 | Terminal render with and without acceleration |
| Feed SSE in webview | Live feed for 30 minutes |
| Pairing in the embedded profile | QR/token once; session persists under `%LOCALAPPDATA%` |
| Duplicated keepalive during transition | Go tray and shell coexist; one-wins in Phase 2 |

## Spike run — 2026-09-11 (evidence + owner checklist)

| Spike | Status |
|---|---|
| Feed SSE (server side) | **PASS** — verified live: `GET /api/events` streams (`event: hello`, bootId + latest); the UI updates without reload. |
| Native notification | **Implemented** (tray item **Test notification**); toast click-through pending owner. Windows shows toasts for unpackaged apps only with a Start Menu shortcut carrying the app identity — a silent drop is the Phase 2 installer requirement. |
| mkcert/HTTPS | **PASS** — owner: no certificate warning, UI over `https://localhost:8445` (2026-09-11). |
| Pairing | **PASS** — owner: UI fully usable, no pair screen (loopback auto-pairs, ADR-0049). |
| xterm.js/WebGL | **PASS** — owner: terminal renders and types, including the live session (2026-09-11). |

**Phase 1 closed 2026-09-11** — five of five spikes settled (one pending the
toast click-through, which only changes the Phase 2 installer scope).

## Conscious debt

- Optimize-VHD from the tray/app (needs an elevation design).
- Taskbar identity for the managed Work Browser (it is Chrome's icon).
- Icon resolution: the shell's `icons/icon.ico` now carries 16–256 px
  (tools/mkicon.go, the shell variant of the tray generator) — the v1 tray
  ladder topped at 64 px and the taskbar upscaled it blurry.
- Disk history/thresholds (P4 of `wsl-control`).
