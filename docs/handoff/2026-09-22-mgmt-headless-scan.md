# 2026-09-22 — feat/mgmt-headless-scan: headless Management with a streamed disk scan

Owner report: opening the desktop Management window popped a visible Windows Terminal titled `C:\WINDOWS\system32\wsl.exe`, and the page showed only dashes while it measured.

Cause: `cmd/picode-desktop/clean.go` spawned `wsl.exe` with a bare `exec.Command`; `picode-desktop.exe` links `-H=windowsgui`, so a console child without CREATE_NO_WINDOW gets its own window. It now goes through `newCmd`; `TestNoBareExecCommand` guards the package. The shell's `resolve_user` / `discover_server` wsl.exe calls gained `hide_console` too.

Shipped (d12bfaed):
- `picode-desktop disk --json --stream`: one progress line per half (Windows, distro) with its data, report last. Measured live: Windows 0.6 s, distro 15.5 s.
- Rust: one `stream_cli`/`read_stream` for scan, compact and clean — `mgmt-progress` events tagged op + run token, one job at a time (global mutex), lossy UTF-8, non-stream fallback for an older exe. Fixed a pre-existing bug: the page listened on `disk-progress` while Rust emitted `mgmt-progress`, so compact steps never showed.
- `management.html`: activity rows with live clocks, skeletons, stale dimming on rescan, Clean tab fed by the same scan (one du walk instead of two), held-space explanation, failure states.

Verified: on Windows (EnumWindows probe), the main-branch binary launched from a GUI process opened a visible `wsl.exe` window; the new binary opened none. `read_stream` tests pass as a Windows test exe via interop. Adversarial review: no blockers; fixed a false "distro was restarted" claim on early errors, cross-run event mixing, the dropped held-space line, lossy UTF-8, mixed numbers after a failed rescan, and a listener registered after the first scan.
Confirmed live by the owner 2026-09-23 after `make deploy` + `make desktop-restart`: Management opens with no terminal window and the scan streams (Windows ≈1 s, distro ≈15 s).
visual-review: PASS after two FAIL rounds (12 states in `var/screenshots/mgmt/`).
Debt recorded in `docs/handoff/open/windows-wsl.md`: `open_management_window` runs `discover_server` on the tray event thread.
Merge: fast-forward ready.
