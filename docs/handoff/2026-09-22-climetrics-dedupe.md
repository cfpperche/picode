# 2026-09-22 — feat/climetrics-dedupe: Claude Code, Codex and Grok meters count each billed token once

Shipped (`internal/climetrics/`, `docs/architecture/climetrics.md`):
- Claude Code: dedupe by `message.id:requestId`, keeping the record with the largest output; reads the whole `<session>/subagents/` tree, workflow agents included.
- Codex: fork-copy suppression (`forked_from_id` only); the first `session_meta` names the session; repeated `token_count` dropped; cached subtracted from input; `custom_tool_call` counted as a tool.
- Grok: usage from `updates.jsonl` `turn_completed` (fallback `usage.json`); turns before `summary.json` `forked_at` dropped; cached subtracted from input.

Measured on the owner's machine, 30-day window: Claude Code 12.6B -> 8.1B tokens (cost unchanged at $5,046 — it is snapshot-spread); Codex 2.8B -> 1.4B tokens (sessions 60 -> 88, forks are their own sessions now); Grok spend $273 -> $290.
Method compared against t3code's usage reader (pingdotgg/t3code d7819c18, `apps/server/src/usage/usageTranscripts.ts`). An adversarial review found 4 bugs in b425932b; all fixed in 8ccdb1cf.

Verified: go tests + `make ci-scoped` green. Blind spot: numbers checked against transcripts on one machine, not on the dashboard.
visual-review: n/a — no UI change in this branch; dashboard values not screenshot-verified.
Not done / debts: below and in `docs/handoff/open/dashboard.md`. Not deployed.
Merge: fast-forward ready.

## Next up

- The Limits card reads only Codex rollouts; feat/dashboard-limits (in flight) wires the Providers usage cache into it.
- Price table for unpriced tokens (Codex, live Claude Code sessions) is an owner decision that reopens the ADR-0097 study's refusal (docs/handoff/open/dashboard.md).

## Debts

- Codex `duration_ms`/`time_to_first_token_ms` still unread (docs/handoff/open/dashboard.md).
- Subagent cost attribution by day/model follows the parent's messages; the session total is exact (docs/handoff/open/dashboard.md).
