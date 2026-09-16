# 2026-09-16 — shell-hidden-window: Open PiCode opened nothing

Owner report: after the 15:12 desktop-restart, the tray could not bring the
app up — only the browser at localhost:8445 worked ("Open PiCode" opened
nothing, and so did a second launch).

Cause (measured, not guessed): the resident runs from the logon task with
`--hidden`. Its main window existed and was healthy, but the app's own
lookup — `app.get_webview_window("main")` — returned `None` (Tauri 2.11.5,
Windows). Every open path (tray item, single-instance forward) is built on
that lookup, so every one of them did nothing, silently. Proven by asking
Windows directly: `EnumWindows` for the shell's PID found `Tauri Window
title=[PiCode]`, and a native `ShowWindow` put it on screen instantly while
the app's path never did. (A first attempt — rebuilding the window — hit
"a webview with label `main` already exists", which is what the owner saw in
the alert box: the label was taken, the lookup was what missed.)

Fix: the handle is kept in a `OnceLock` the moment the window is built
(`build_main_window`, used by setup and by the rebuild fallback), `show_main`
drives that handle (registry, then rebuild, only as fallbacks), and `show()`
also calls Win32 `ShowWindow(SW_RESTORE|SW_SHOW)` + `SetForegroundWindow`.
The build comment about `visible(true)` then hide is now honest: at logon the
window ends up visible-but-parked off-screen either way — the build order
was never the cause.

Verified on the owner's machine (exe swapped + resident restarted through the
logon task, Tauri 2.11.5): logon state `vis=False rect=203,178,1578,1067`;
after the app's own open path `vis=True` at the same rect — the window is on
screen. `cargo xwin build` ✓. The tray click itself is the owner's to try.

Note: the exe was hot-swapped into `%LOCALAPPDATA%\PiCode` and the resident
restarted (taskkill + logon task) — no `make deploy` needed, the fix is
shell-only; the daemon was never touched.

## Next up

- Owner: right-click the tray → Open PiCode (or double-click the PiCode
  shortcut) — the window must come up in front, every time.

## Debts

- The open path has no automated test: driving the tray needs a click, and
  the window dance is Windows-only. The failure mode is at least loud now
  (an alert box, not silence) if the rebuild fallback is ever reached.
