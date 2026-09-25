# 2026-09-25 — feat/tmux-zero-badge: tmux app tabs drop the "0" count badge
Shipped: internal/apps/tmux.go formatted tab badges with strconv.Itoa, so the
Sockets tab showed "0" when no tmux server ran (and Sessions/Server at zero
sessions). New `countBadge(n)` returns "" at ≤ 0 and is used for all three
tabs. The Sockets list block also carried a Meta line "N socket(s)" that
repeated the group header's own item count — removed.
docs/architecture/tmux-app.md updated.
Verified: TestCountBadge and TestTmuxAppSocketsViewNoRunningServerHasNoBadge;
`go test ./internal/apps` ok; `make ci-scoped` PASS. Scratch instance,
screenshots read in subagents: tabs show no pill, the header shows the count
once.
visual-review: PASS
Pays the tab-overflow debt (docs/handoff/2026-09-24-tab-overflow.md).
Not done / debts: none.
Merge: fast-forward ready.
