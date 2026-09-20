# Windows and WSL

## Next

- Windows clean-machine install (ADR-0098): phase 1 stages in `picode-desktop.exe`, phase 2 `install.ps1` + winget, no paid signing (`docs/plans/windows-clean-install.md`).
- WSL control (P0/P1 shipped): P2 is the Storage app plus the Windows-facts route (own ADR); P3 — compact, `--set-sparse`, `--move` — costs the distro's sessions.

## Windows installer M1 — code landed, VM gate open (2026-09-19)

The branch that had been idle for four days (`feat/win-install-m1`) is
adopted and its WIP committed: the parent waits for the elevated child and
reports its exit code, the last error line lands in
`%ProgramData%\PiCode Desktop\install.log`, the elevated window pauses for
Enter on failure, `--user <name>` aims both the binary and pi at one account
(and `--user root` keeps the system-wide install), and the stage scripts no
longer carry a `$` that wsl.exe would eat. Plan: `docs/plans/windows-installer.md`.

**Open (owner, on the VM)**: M1's gate is the scenario run — clean-machine
path and adopted-Ubuntu `--user` path, idempotent re-runs, on `picode-test`
(Hyper-V, checkpoint `clean`), plus `startup-check`. M2 (`install.ps1` on the
site, the guide around the one-liner, the release-train pin bump) is the next
buildable milestone.

## Debts

- WSL disk: the tray warns in words only (no alert icon asset); docker's storage is the Docker app's, not measured here.
- Windows `Close` kills only its direct child; `internal/server` tests swap package-level probes, so `t.Parallel` would race (`scripts/go-test.sh` shards by process).
