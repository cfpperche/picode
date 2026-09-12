# Windows and WSL

## Next

- Windows clean-machine install (ADR-0098): phase 1 stages in `picode-desktop.exe`, phase 2 `install.ps1` + winget, no paid signing (`docs/plans/windows-clean-install.md`).
- WSL control (P0/P1 shipped): P2 is the Storage app plus the Windows-facts route, which needs its own ADR; P3's actions — compact, `--set-sparse`, `--move` — cost the distro's sessions (`docs/plans/wsl-control.md`).

## Debts

- WSL disk: the tray warns in words only (no alert icon asset); docker's storage is the Docker app's, not measured here.
- Windows `Close` kills only its direct child; `internal/server` tests swap package-level probes, so `t.Parallel` would race (`scripts/go-test.sh` shards by process).
