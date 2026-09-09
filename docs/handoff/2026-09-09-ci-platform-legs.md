# 2026-09-09 — feat/ci-platform-legs: macOS and Windows legs reach the tests

Shipped: `install.DeployForce` exists on Windows (delegates to `Deploy`, which
refuses without systemd), and `deploy_record_test.go` carries the `unix`
tag — `cmd/picode` did not compile on Windows since the ADR-0086 deploy
guard. The llama binary-guard test resolves its temp dir through
`EvalSymlinks` (macOS puts it under `/var` → `/private/var`, which the guard
rightly rejects). The workflow installs Homebrew tmux on macOS and keeps a
server alive there too, so the handoff and terminal suites run instead of
answering 503.
Verified: `CGO_ENABLED=0 GOOS=windows go vet ./cmd/picode ./internal/install`;
the llama test under a symlinked `TMPDIR` fails before and passes after;
`make close`; a manual full-matrix run on GitHub is the final proof.
visual-review: n/a
Not done / debts: macOS and Windows still run only on tags and manual runs
(ADR-0105); a green manual run is the first evidence they work at all.
Merge: fast-forward ready.
