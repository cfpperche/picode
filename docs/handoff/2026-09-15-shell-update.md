# 2026-09-15 — feat/shell-update (ADR-0142 slice 3)

One release updates both exes. `release.yml` builds and publishes
`picode-shell-windows-amd64.exe` (dtolnay toolchain, apt llvm/clang
with clang-cl symlink fallback, pinned cargo-xwin 0.23.1, tag stamped
into shell; recipe proven in a fresh Ubuntu 24.04 docker container).
`update.go` does a two-asset SHA256SUMS-verified update with
swapExe/unswapExe rollback and shell-missing tolerance.
`Release.URLs` map: one fetch, no mixed versions. `release_test`
asserts the ShellAsset workflow asset. `desktop-swap.sh` relaunches
the task's target (resident tray/shell); DRY_RUN verified live with
resident=tray and both exes running pre-migration.

## Gates
ci-scoped PASS (full), GOOS=windows vet, bash -n, YAML parsed, seds tested.

## Debts
- Release recipe runs for real only on the next tag push.
- resident==shell swap path verified after migration.
- Install still registers the tray (slice 4 rewires it).

## Next up
Slice 4: tray deletion + docs + live migration with owner.
