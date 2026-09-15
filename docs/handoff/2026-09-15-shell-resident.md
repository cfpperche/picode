# 2026-09-15 — feat/shell-resident (ADR-0142 slice 1)

Shell becomes the resident; commit `12c20fb8`.

- `keepalive.rs`: WSL sleep + Job Object child tracking.
- `health.rs`: curl probe; `status.rs`: tooltip/title.
- `main.rs`: `--hidden`, status menu item, 5s poll thread, discover returns distro+URL.
- Go retarget: `task.go` RetargetTask/CanRetarget/ResidentKind/ShellExe, `task.ps1` retarget op, `startup-repair --retarget-shell`, Resident task label.

Gates: `make ci-scoped` PASS (full); xwin release build ok; Go tests ok; Rust unit tests via `rustc --test` + offline scratch crate (11 pass).

## Next up

- Slice 2: disk/action parity.
- Slice 3: update+release.
- Slice 4: tray deletion + live migration.

## Debts

- Host `cargo test` broken pre-existing (windows-future/windows-core skew in lockfile, untouched).
- Native Windows task tests compile in CI, run on owner Windows.
- Live keepalive/task acceptance owner-side per `docs/plans/retire-go-tray.md`.
