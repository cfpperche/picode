# 2026-09-13 — peer-stop-child-race: the server suite is deterministic again

Shipped: three fixture fixes behind `main`'s red CI. The stubborn-child stop test now waits for the pane's installed traps and targets tmux without the ambient client (20× green); the CLI resume row reads the resumed process's own argv instead of the creation launch's stale file; the SIGHUP pane fixtures use process-unique session names so two concurrent test binaries cannot share a pane. The board's red line is replaced by the paid record and a naming rule for future tmux fixtures.
Verified: `go test ./internal/server/` full package green twice (~190 s each); peer-stop 20×, resume 5×, SIGHUP 20×; no leaked tmux sessions; `make close` next.
visual-review: n/a
Not done / debts: none from this branch; tmux fixture naming rule recorded in `docs/handoff/open/terminal.md`.
Merge: fast-forward ready after this close.
