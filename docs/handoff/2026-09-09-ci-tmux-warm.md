# 2026-09-09 — feat/ci-tmux-warm: first run of the ADR-0105 workflow

Shipped: the Ubuntu Go job keeps a detached tmux session alive before
`go test -race ./...`. The first run of the new workflow (run 34387220862)
was green everywhere except `internal/term`: `TestBridgeResizeControl`
failed with "tmux has-session: server exited unexpectedly" — two packages
racing to start the first tmux server on a fresh runner. Nobody calls
kill-server; the other tmux suites passed on the same runner. Earlier red
runs never reached this step (whitespace, then the Windows toolchain), so
the race predates ADR-0105.
Verified: YAML parses; `make close` on this branch; the run for this push is
the proof — check it before assuming green.
visual-review: n/a
Not done / debts: `internal/tmux.HasSession` could treat "server exited
unexpectedly" as a transient (retry once) so the product never sees the
same race; product change, not made here.
Merge: fast-forward ready.
