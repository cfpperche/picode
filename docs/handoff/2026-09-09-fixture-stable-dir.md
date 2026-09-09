# 2026-09-09 — feat/fixture-stable-dir: deterministic inspector capture

Shipped: `cmd/picode-docs-fixture` names its data directory after the listen
port (`$TMPDIR/picode-docs-fixture-18740` by default) instead of a random
`MkdirTemp` suffix. The inspector renders the workspace path, so every
recapture differed by the suffix alone and became a new PNG (373 px between
the two deploys of 2026-09-09). Concurrent worktrees use separate ports and
keep separate directories; the previous run's directory on the same port is
removed first (the port is exclusive; `make docs-shots` kills its holder).
`portSuffix` has a table test. No changelog fragment: developer tooling,
nothing user-visible.
Verified: `go test ./cmd/picode-docs-fixture`, `make close`, then
`make docs-shots` on main — the inspector changes once (path now fixed) and
must stay identical on the next capture.
visual-review: n/a
Not done / debts: a fixture killed with SIGKILL leaves its directory behind
until the next run on that port removes it.
Merge: fast-forward ready.
