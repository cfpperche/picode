# Retire the Go tray — the shell becomes the only resident

Scope: [ADR-0142](../decisions/0142-retire-go-tray.md). Kill the Go systray
resident (`picode-desktop.exe --tray`) and consolidate every Windows-resident
duty in `picode-shell.exe`: one tray icon, one autostart, one keepalive.
The systemd unit inside WSL, linger, and `picode
install/provision/deploy` are untouched — the retirement is the Windows
resident, not daemon supervision.

Do not: run `wsl --shutdown`, drop the keepalive for even a minute,
re-point the task at an unverified exe, ship a shell without supervision
tests, or fold the ADR-0098 clean-machine chain into this work (its target
just becomes "the shell is the resident").

## Duty inventory (what moves, what stays a Go tool)

| Duty | Today | After |
|---|---|---|
| Logon autostart (`PiCodeDesktop` task, ADR-0071 policy) | Go tray exe | shell exe, same task name + policy |
| WSL keepalive (`sleep infinity` + Job Object, re-arm after compact) | Go tray | Rust in the shell |
| Health poll 5s, disk poll 5min, tooltip, status menu | Go tray | shell tray |
| Give-back, Restart (`systemctl --user restart picode`), Logs (`wt.exe` journal) | Go tray menu | shell tray menu (same argv) |
| Window close hides / tray Quit exits | n/a (tray has no window) | shell (ADR-0142 contract) |
| `install doctor disk disk-compact clean startup-check startup-repair update` | Go CLI | Go CLI, driven by the shell via `run_cli` |
| `update` + `release.yml` | tray exe only | shell + tool exes, same tag, `SHA256SUMS`-verified |

## Slices (one branch each, delete last)

0. **ADR + this runbook** (this branch): `docs/decisions/0142-*`,
   `docs/plans/retire-go-tray.md`, changelog fragment.
1. **Resident core**: Rust keepalive + hidden-start at logon + health poll +
   tooltip/status parity; `startup-repair` accepts the shell exe.
2. **Disk/action parity**: disk line + `low` warning, tray Give-back
   (DeployReady interlock + confirm + measured result), Restart, Logs.
3. **Update + release + single swap**: `release.yml` publishes the shell exe,
   `update` swaps both exes, `desktop-swap.sh` relaunches one resident.
4. **Delete + docs + live migration**: remove the Go tray loop and the
   systray dep, rewrite `docs-site/guide/windows-desktop.md`, update
   `docs/plans/desktop-v2.md` and the desktop section of
   `docs/architecture.md`, run the migration below with the owner.

## Handover decision table

| Old tray | Shell keepalive | Task points at | Action |
|---|---|---|---|
| running | not yet verified | Go tray | start the shell, verify `Running` + health; touch nothing else |
| running | verified (health ok) | Go tray | Quit the old tray; retarget the task; verify policy |
| stopped | verified | Go tray | retarget the task; verify policy |
| any | fails to come up / health fails | — | **abort**: task stays on the Go tray; nothing else to undo |

At least one `sleep infinity` must be alive at every instant of the swap.

## Migration runbook (owner machine, agents running)

Probes first (Windows terminal, `%LOCALAPPDATA%\PiCode`):

```powershell
tasklist /FI "IMAGENAME eq picode-desktop.exe"
tasklist /FI "IMAGENAME eq picode-shell.exe"
schtasks /query /tn PiCodeDesktop
wsl --list --verbose
.\picode-desktop.exe startup-check
```

Inside WSL: `tmux ls` (record the sessions), `systemctl --user is-active
picode`.

Steps:

1. Install the new shell + tool exes next to the running ones (swap script,
   slice 3); verify versions.
2. Start the shell resident; confirm one new tray icon, health green,
   `wsl --list --verbose` shows `Running`.
3. Quit the **old** tray from its own menu (exit 0, no task retry).
   Re-verify: exactly one Pi icon, health still green, `tmux ls` unchanged.
4. Retarget the task: `startup-repair` (backs the XML up under
   `%LOCALAPPDATA%\PiCode\task-backups`), then `startup-check` — policy
   issues must be zero.
5. Reboot test (owner's call on timing): after sign-in, exactly one icon,
   window hidden, `/api/health` answering, `tmux ls` intact. Quit + relaunch
   the shell: the pane comes back at the same page.

Rollback (any step fails): `schtasks /run /tn PiCodeDesktop` relaunches
whatever the task points at; reimport the backed-up XML to return the task
to the Go tray; the previous release's `picode-desktop.exe` restores the
old resident. No repo state needs reverting — the migration is all runtime.

## Acceptance matrix (slice 4, live)

- [ ] Logon → one icon, window hidden, health green, no second `sleep infinity`
- [ ] Window close → tray alive, WSL `Running`; reopen → work resumes
- [ ] Tray Quit → process exits 0; WSL idles down (documented behavior)
- [ ] Give-back → keepalive re-armed, distro stays `Running`
- [ ] `startup-check` clean; `startup-repair` no-op on a healthy task
- [ ] `tmux ls` identical before/after every step
