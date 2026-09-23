# 2026-09-23 — feat/instr-observed: what an agent's session actually read

Step 2 of the AGENTS.md plan (owner: "siga com a execução").

**Record, not prediction.** `internal/cliinstructions/observed.go` reads what a session loaded from the record its own CLI wrote, for the session PiCode pinned on each of the workspace's terminals (ADR-0084): Claude Code's transcript (`Contents of … instructions` entries before the first answer), Codex's rollout (`# AGENTS.md instructions for <folder>`), Grok's `prompt_context.json`. The first version scanned the whole Claude transcript and picked up paths the conversation merely quoted, plus the auto-memory index. Running it against this machine's real records caught both; it now stops at the first answer and keeps instruction entries only. `GET /api/workspaces/{id}/instructions` returns them as `agents`; the tab lists them under "What agents here read".

Slip: `observed.go` was first written to the root checkout by a wrong relative path, and moved into the worktree at once. The root was clean again before any commit.

Verified: `make ci-scoped` and `make close`. Visual review on scratch `instrobs`, which was seeded with a pinned Claude session (no model call). Nothing deployed.
