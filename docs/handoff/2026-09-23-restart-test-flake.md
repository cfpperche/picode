# 2026-09-23 — feat/restart-test-flake: `panePIDSoon` retries tmux pid reads under load

Shipped (251900e87, test-only):
- `panePIDSoon` retries the pane-pid read for up to 5 s, failing with the real error instead of dropping it (internal/server/cli_inspection_test.go).

Verified: `TestCLIRestartPreparationFailureAndWorkspaceCleanup` failed once in full 4-shard `make close` (2026-09-23, "preparation failure killed the old process") and passed on retry; 0 of 24 under three parallel loops alone. Fix is by construction, not measured: both reads dropped their error, so a tmux call that failed under load read as pid 0 and looked like a killed process. `make close` PASS (fmt, vet, hooks, go, living-docs). Blind spot: not reproduced; addresses a suspected timing gap.
visual-review: n/a (test-only, no UI change).
Not done / debts: none new.
Merge: fast-forward ready (`git merge --ff-only feat/restart-test-flake && make ci`).
