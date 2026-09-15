### Added
- **The shell is becoming the only Windows resident (ADR-0142, slice 1).**
  `picode-shell.exe` now holds the WSL distro open with a supervised
  keepalive, polls daemon health every 5 seconds, and reports it in a tray
  status line and tooltip — the duties the Go tray owned. `--hidden` starts
  the resident with no window (the logon-task mode); a second launch brings
  the window up. `picode-desktop startup-repair --retarget-shell` moves the
  `PiCodeDesktop` task to the shell, refusing foreign tasks and missing
  executables; plain repair now accepts either resident.
