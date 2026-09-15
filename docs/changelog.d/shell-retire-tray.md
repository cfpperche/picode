### Removed
- **The Go tray is deleted (ADR-0142).** `picode-shell.exe` is the only
  Windows resident — autostart, keepalive, health, disk line, Give-back,
  Restart, Logs — and `picode-desktop.exe` is its headless boundary tool
  (`doctor install disk disk-compact clean startup-check startup-repair
  update`). The systray dependency is gone, so a second tray cannot come
  back. A logon task that still points at the tray fails loud with the
  repair named; `install` registers the shell, staging it from the
  matching release when no sibling is beside the tool.
