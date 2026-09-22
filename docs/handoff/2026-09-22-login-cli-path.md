# 2026-09-22 — feat/login-cli-path: put the intercept bin dir first on CLI launch PATHs
Shipped: `interceptCLIPath(dataDir, c)` in `internal/server/cli_launch.go` — the
intercept bin dir now leads both the launch script's `export PATH` and the pane's
tmux environment (`dropEnv` keeps a single PATH key). Follow-up to ADR-0180
(feat/login-open-url; base e72485ef, head 3401f814, one commit).
Measured failure: after the ADR-0180 deploy, `omp /login` still opened WSL chromium —
CLI panes are `/bin/sh` launch scripts, so the rcfile PATH prepend
(interactive shells only) never runs; omp resolved system wslview/xdg-open and
ignores `$BROWSER` (verified in its bundle: opener = wslview-if-in-PATH else
xdg-open).
Verified: `TestCLILaunchPutsTheInterceptBinFirstOnPATH` (launch.sh export +
tmux show-environment lead with the bin dir; probe inherits process env — the
same server the Manager used); `TestCLITerminalLifecycleDecisionTable` updated
to the new PATH contract. ci-scoped PASS. Blind spot: the OAuth open itself was
not observed end-to-end in a fresh WSL terminal.
visual-review: n/a
Not done / debts: none new; prior ADR-0180 debts stand
(docs/handoff/2026-09-22-login-open-url.md). Merge: fast-forward ready

## Next up

- after next `--force` deploy: /login in a NEW terminal opens outside WSL

## Debts
- (none new — see docs/handoff/2026-09-22-login-open-url.md)
