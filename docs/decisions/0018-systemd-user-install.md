# ADR-0018: `picode install` — systemd user unit, not Windows

- **Status**: accepted
- **Date**: 2026-08-29

## Context

PiCode was a manual `bin/picode` process. A Windows reboot left it down
(stale lock, no service). The owner wants `install` / `uninstall` so it
comes up with **Linux** (WSL in their case), not with Windows logon. They
decide when the distro starts.

Options: Windows Task Scheduler + WSL, systemd system unit (`sudo`),
systemd **user** unit.

## Decision

`picode install` copies the binary to `~/.local/bin/picode` and enables a
**systemd user** unit (`~/.config/systemd/user/picode.service`,
`WantedBy=default.target`). `picode uninstall` stops and removes that
unit (and the copy). `--purge` also deletes `~/.picode`. No Windows
task. No system (`sudo`) unit. macOS/Windows native is later.

The unit snapshots `PATH` at install time so `pi` (nvm) still resolves.
Restart=on-failure. Logs go to the user journal.

## Consequences

- **Easier**: WSL already open → PiCode stays up; crash restarts; one
  command to undo.
- **Harder**: a full Windows reboot still needs the user to start WSL
  (accepted). A running stray `picode` must be stopped before the unit
  can bind the lock.
- **If wrong**: uninstall leaves the machine as before; data stays
  without `--purge`.

## Alternatives considered

| Alternative | Why not |
|---|---|
| Windows logon task | Owner: PiCode should follow Linux/WSL, not the PC |
| system systemd (`sudo`) | Personal tool; user-level is enough |
| linger always | Other apps share linger; don't toggle it |
