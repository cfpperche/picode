# Desktop v2 shell

## Traps

- **Never trust the window registry for the main window.** Measured
  2026-09-16 (Tauri 2.11.5, resident started by the logon task with
  `--hidden`): the native window was alive while
  `app.get_webview_window("main")` answered `None`, so tray ▸ *Open PiCode*
  and a second launch both no-op'd silently. The shell keeps the
  `WebviewWindow` in a `OnceLock` from `build_main_window` and `show()` also
  calls Win32 `ShowWindow`/`SetForegroundWindow`; trust that pair, not the
  lookup. Diagnosing it from WSL: enumerate the shell's top-level windows
  with `EnumWindows` + `GetWindowText` — `tasklist /V` shows the
  single-instance helper `app.picode.shell-siw` and tells you nothing about
  whether `main` is hidden, parked off-screen (`-21333,-21333`) or on screen.

## Next

- Policy UI for the desktop-v2 gates (ADR-0120). The daemon endpoint and the navigation gate are tracked in `work-browser-tabs.md`.

## Debts

- Named-window OAuth popups (`window.open(url, 'picode-mcp-auth')`) still use the native path in the shell's main window — needs its own auth-flow verification before routing out (the `_blank` bridge covers the rest since 2026-09-15).
