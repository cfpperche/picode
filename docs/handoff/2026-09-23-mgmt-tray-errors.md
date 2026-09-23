# 2026-09-23 — feat/mgmt-tray-errors: Management opens off the tray thread; plain Management errors

Follow-up to `feat/mgmt-headless-scan` (merged 2026-09-22); pays the debt in `docs/handoff/open/windows-wsl.md`: `open_management_window` ran `discover_server()` (a hidden `wsl.exe`) on the tray event thread.

Shipped (f430fe02):
- `board.rs` keeps the URL the health loop last got answers from (`note_url` on success, cleared on failure). Management and main-window rebuilds open from it; an unknown address is looked up on a thread and the window built via `run_on_main_thread`, with an `AtomicBool` latch against double opens.
- Found in review: since 778fd8ba (2026-09-18) `main_target` passed the distro name to `await_desktop_ready`, so the gate probed `Ubuntu/desktop/`, never got a 200 and waited its full 30 s on every main-window build (startup and Open PiCode rebuilds); the restart-404 guard never worked. It now probes the origin.
- `disk-compact` outcome gains `stopped` (set right before `desktop.Compact`, which terminates first): post-stop failures say the sessions ended; early failures say "Nothing was stopped". Test: `TestStoppedTravelsOnTheWire`.
- `management.html`: an anchored regex table maps known tool errors to one plain line + one action, raw text beneath (`.tech`); the Clean tab shows a line after a failed first scan; unknown values are neutral dashes; `[hidden]` now beats `.actions { display:flex }` (the empty-list Clean button was never actually hidden).

Verified: Go test for `stopped` on the wire; adversarial review found no blockers, its medium (post-stop failure hidden) fixed. visual-review: PASS after one round (6 states in `var/screenshots/mgmt2/`, stubbed Tauri).
Confirmed live by the owner 2026-09-23 after the forced deploy + `make desktop-restart`: Management opens at once with no terminal, the scan streams, and the main window no longer waits ~30 s.
Not done: setup's first `main_target` still runs discovery on the setup path before the tray exists (startup only); a cold Management open with no known address shows nothing for seconds (no "opening…" feedback); the Management window's capability allows only `https://localhost:8445` / `127.0.0.1:8445`, so a daemon on 8446–8455 gets no IPC (pre-existing).
Merge: on `main` at 2d195616, `make ci` green.

## Debts

- [x] Management capability allows only port 8445; a daemon on 8446–8455 gets no IPC in the Management window. Paid by feat/mgmt-port-range: the daemon's exact origin is granted at runtime.
