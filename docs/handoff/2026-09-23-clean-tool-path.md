# 2026-09-23 — feat/clean-tool-path: cleanup finds go, uv and pnpm the way the person's terminal would

Owner clicked "Clean selected" on the Go build cache (72 GB) in the desktop Management window and got "failed: go is not installed": `cmd/picode/clean.go` `runHow` used `exec.LookPath` in the bare PATH `wsl.exe` gives (no `~/go-sdk`, nvm, `~/.local/bin`).

Shipped (75483ac6):
- `toolPath` resolves a tool from the process PATH → the service drop-in PATH → the unit PATH (new `install.ParseEnvironment`, systemd quoting rules) → `$SHELL -lic 'command -v'`, bounded by context + `WaitDelay`, names on an allowlist.
- `/mnt/*` (Windows-drive) tools are skipped; the tool's dir is prepended to the child's PATH.

Verified: probe in the real bare env — go, uv, pnpm unresolvable before; after: `~/go-sdk/go/bin/go`, `~/.local/bin/uv`, nvm's pnpm. Tests: 8-row decision table, hermetic fake shell, syntax refusal, bound test (6 s vs 30 s without `WaitDelay`). Adversarial review fixed: timeout not bounding, unquoted/quoted unit parsing, Windows-drive npm, drop-in fallthrough, non-hermetic test.
Blind spot: not exercised through the desktop Management window; the new binary reaches the distro only via `make deploy` (no desktop-restart needed).
visual-review: n/a
Debts added to `docs/handoff/open/windows-wsl.md`: pnpm row measures `~/.cache/pnpm` while prune works on the store; a GOCACHE elsewhere reports 0 freed.
Merge: on `main` at e3587a88, `make ci` green.

## Next up

- Owner runs `make deploy`, then retries "Clean selected" on the Go build cache in the Management window.
