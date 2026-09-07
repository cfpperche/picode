# 2026-09-06 — feat/session-handoff: continue a session in another Agent CLI

Shipped (ADR-0088): `internal/transcript` (timeline, window, repair,
prepare, note, deterministic brief); `clisession` Reader for Claude Code,
Codex, Grok (per-session dirs now listed with title/model/size), pi and
Hermes; Writer for Claude Code, Codex, pi (create-only, atomic, round-trip
verified); Prompter for all but Hermes; `POST /api/clis/{cli}/sessions/
handoff[/preview]` on one plan; `session_handoffs` + `session.handoff`;
`GET /api/clis` advertises `sessions` capabilities; Sessions rows get a
••• menu, a dialog and lineage badges; setup check runs on demand.

Verified: `make ci-scoped` green after `make openapi`; scratch instance
with a real Claude Code 2.1.263 session → Codex 0.153.4 native: rollout
indexed by Codex's startup backfill, `codex resume <id>` loaded it and
answered "what were we doing" with the source's last command; the TUI
shows no prior turns for a `legacy` rollout but context loads (4%) →
Claude Code native (from the Codex-written file): history and tool calls
render; "Session model unknown" when the source model is unknown → pi:
stopped agent, chat renders 5 tools ok → Grok 1.0.13 brief: read it and
resumed the last ask.

Not done: Grok native spike; Hermes as target (owner decision); Codex
writer emits no `turn_context` (model lost Codex→Claude); Codex's picker
lists a handed-off rollout only after a Codex restart.

Merge: pending `make close` and fast-forward of main.
