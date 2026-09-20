# 2026-09-13 — fix-clis-stuck: the Agent CLIs page survives a slow terminal list

Shipped: `TerminalsCache` (internal/server/terminals_cache.go) — singleflight
+ 1s TTL behind `GET /api/terminals`, per-terminal facts on a bounded worker
pool (6), mutating terminal handlers invalidate the snapshot; AgentClis
refresh applies the last landed result (web/browser/src/components/AgentClis.jsx);
legacy agent ids go through `isAgentTab` (App.jsx) so `w:<n>` web-tab ids stop
404ing `/api/agents/{id}/*`.
Verified: `make ci-scoped` green (go, test-js, build); cache decision table
(8 rows) in terminals_cache_test.go; full server suite green after merge;
scratch instance `qa-scratch.sh` — page loads past the skeleton, overlay
audit ok, no console errors.
visual-review: PASS (clis-scratch-loaded.png read; card 5/5)
Not done / debts: none on this branch. Root cause was load-shaped, not a
regression: 72 tmux sessions + 10 managed terminals + ~1.5 terminal.state
events/s made the sequential list slower than the event cadence.
Merge: fast-forward ready (main merged in, gates re-run).
