# 2026-09-11 — feat/grok-usage-meter: Grok's dashboard row stops lying

Shipped: `internal/climetrics/grok.go` reads the four files Grok writes per
session — `prompt_history.jsonl` (prompts), `summary.json` (cwd, model,
title), `events.jsonl` (turns, tools, `tool_completed.duration_ms`,
`first_token`) and `usage.json` (per-turn tokens, `costUsdTicks` at 10¹⁰/USD).
Coverage moves from "prompt history only" to turns/tools/errors/timing
`reported` and tokens/cost `partial` with both counts. `cache.go` gains
`byPresence` proration (Grok weighs turns, not tokens) and `statKey`/
`cachedParseKeyed` for a parse spanning three files. Docs: dated addendum in
`docs/benchmarks/2026-09-07-cross-cli-agent-telemetry.md` and
`docs/plans/dashboard-cross-cli.md` §2/§3/§9 corrected.

Verified: `make ci-scoped` + `make close` green (fmt, vet, hooks, go).
Five Grok tests; the meter run on this machine's real store — 199 sessions,
785 turns, 478 tool errors, 51 aborted turns, all matching the raw event
counts, and apiMs + toolMs ≈ sessionMs.

visual-review: n/a (no JSX/CSS touched, values only); not screenshot-verified.

Not done / debts: Codex's `task_complete.duration_ms` +
`time_to_first_token_ms` still unread by its meter; the Agent CLIs session
row shows Grok's model and title but not the per-turn cost `usage.json`
holds.

Merge: fast-forward ready (main can ff to this branch).
