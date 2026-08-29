# ADR-0020: PiCode Desktop — Windows provisions the distro

- **Status**: accepted (supersedes ADR-0018's "no Windows task", "no linger")
- **Date**: 2026-08-29

## Context

ADR-0018 decided PiCode follows **Linux**, not the PC: `picode install`
writes a systemd **user** unit, and "a full Windows reboot still needs the
user to start WSL" was accepted as a cost — the owner decides when the
distro starts.

The requirement changed the same day. The owner runs PiCode in WSL and
wants the opposite contract: **install one thing on Windows, open the
browser, PiCode is there.** No terminal, no manual step — and on a machine
that already has a live PiCode environment to protect.

Three gaps block that:

| Gap | Today |
|---|---|
| The distro does not start with Windows | Nothing exists on the Windows side; `install_windows.go` is a stub that returns an error |
| A user unit does not start without a login | `Linger=no`, so the user manager never boots and `WantedBy=default.target` never fires |
| `systemd=true` is a prerequisite, not an action | `install.Install()` returns "need systemd (user). In WSL set systemd=true in /etc/wsl.conf" and stops |

The rest of the Linux half is already built: `internal/install` copies the
binary, writes the unit, stops a stray process (SIGTERM, 5 s, SIGKILL) and
enables the service — and no path in it writes to `~/.picode`.

Two facts make unattended provisioning possible. `wsl.exe -u root` runs as
root inside the distro without a password, so the Windows side can do
root-level distro config. And `GET /api/health` already answers
`{status, bootId}`, so "did it actually come up" is a probe, not a guess.

## Decision

PiCode ships a Windows binary, `picode-desktop.exe`, that **provisions the
distro and then lives in the tray**. It owns only the Windows/WSL boundary:
WSL presence and version, distro selection, root-level distro config, the
logon task, and a keepalive child that stops the idle timeout from
reclaiming the VM. Everything inside the distro is done by a new `picode
provision` subcommand that reuses `install.Install()`. The `.exe` never
learns systemd; `provision` never learns Windows.

`provision` converges: every step is check → fix → verify (`/api/health`),
idempotent and additive, with `--dry-run` and `--json` so the Windows side
can show the plan before it runs. It enables lingering for the installing
user and **never disables it**. It edits `/etc/wsl.conf` as a line editor,
never as a serializer.

## Consequences

- **Easier**: install on Windows, get PiCode in the browser. `provision`
  also serves native Linux — one command instead of `picode install` plus
  `scripts/setup-cert.sh`. Certificate setup moves from a repo script into
  the binary, so users who install from a release get it too.
- **Harder**: Windows-only surface to maintain (Task Scheduler, machine
  cert store, UTF-16LE output from `wsl.exe`, no console window on spawn).
  `setup-cert.sh` must be ported to Go. The clean-machine path has two
  discontinuities — `wsl --install` needs a Windows reboot, and adding
  `systemd=true` needs `wsl --shutdown`.
- **Preservation contract.** Provisioning runs against live environments,
  so these are guarantees, not intentions:

  | Subject | Guarantee |
  |---|---|
  | `~/.picode/` | Never written. Only `uninstall --purge` deletes it, and provisioning never purges. |
  | tmux sessions | Survive. The tmux server is a separate process, and agents live in tmux, not in picode. |
  | `/etc/wsl.conf` | Backed up, then merged line by line. A key already correct is not rewritten, so comments and spacing survive byte for byte. |
  | TLS cert and CA | Reissued only when missing or expiring within 30 days. |
  | Existing unit | Backed up before `writeUnit()` overwrites it. |
  | Existing distro | Adopted, never recreated — `--unregister` is not in the installer. |
  | `wsl --shutdown` | Only when `systemd=true` had to be added, and on an adopted distro only after explicit confirmation, because it costs the tmux sessions. |
  | Linger | Enabled, never disabled. |

- **If we're wrong**: `picode uninstall` still removes the unit and the
  binary, and the Windows side removes its own logon task. What is left
  behind is `systemd=true` and linger — both correct settings on their own,
  and both what a WSL user would have set by hand anyway.

## Alternatives considered

| Alternative | Why not |
|---|---|
| Keep ADR-0018 unchanged | Cannot satisfy "install it and it works"; every Windows reboot leaves PiCode down |
| Toggle linger with install/uninstall | ADR-0018's objection stands: linger is shared per-user state. Enabling it is additive; disabling it breaks whoever else relies on it |
| `[boot] command=` with `runuser … systemctl --user start` | Works around systemd instead of using it, and leaves no `/run/user/1000`, so no `XDG_RUNTIME_DIR` |
| systemd **system** unit with `User=` | Avoids linger but loses the user manager: HOME, PATH and `XDG_RUNTIME_DIR` all have to be rebuilt by hand |
| Windows service (NSSM) around `wsl.exe` | Would run before logon, which WSL does not support — it needs a user session |
| MSI / WiX installer | Needs a Windows build host; the binary installing itself keeps the single-binary rule (ADR-0001) and still cross-compiles from WSL |
| Electron or a native GUI | A tray icon and a browser tab do not justify a runtime |
| `vmIdleTimeout=-1` instead of a keepalive child | Documented but reported inconsistent across Windows builds; a live child process is deterministic |
