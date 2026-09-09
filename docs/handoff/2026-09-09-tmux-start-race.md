# 2026-09-09 — feat/tmux-start-race: retry the first-server race in internal/tmux

Shipped: `Manager` gets an injectable `exec`; `HasSession` and `NewSession`
go through `runStartup`, which retries up to three times (50/100 ms apart)
when the tmux client prints "server exited unexpectedly" — the client that
lost the race to start the first server. Every other failure surfaces at
once. Six-row table test with a scripted exec (no tmux needed); the
integration tests are unchanged. Docs: `docs/architecture/terminal-bridge.md`.
Origin: the first ADR-0105 CI run failed `TestBridgeResizeControl` this way;
the CI job now keeps a server alive, and the product no longer depends on it.
Verified: `go test ./internal/tmux ./internal/term ./internal/server`
(sharded), `make close` full matrix.
visual-review: n/a
Not done / debts: the race is reproduced by script, not by two live clients;
`respawn-pane` and the rest keep no retry (a dying server mid-command is a
different failure).
Merge: fast-forward ready.
