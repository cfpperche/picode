# 2026-09-23 — agent-fork: Fork agent… for CLI agents

Study: native fork per CLI. Command line: Claude Code (`--fork-session`), Codex (`codex fork`), Grok (`--fork-session`),
OpenCode (`--fork`), Pi (`--fork`), Omp (`--fork`, parsed but absent from `--help`, found in its source). Muse has
`session/fork` over `muse serve` (MSP). Hermes (`/branch`) and Antigravity (`/fork`) fork only inside their TUI.
Shipped: `clisession.Forker` + the `sessions.fork` capability; `POST /api/agents/{id}/fork-agent` (the old `/fork` route is
Pi's managed in-place fork, untouched); `session_handoffs.mode = "fork"`; git action `create-worktree-branch`;
`AttachComposer` extracted from `TermAttachBar` and reused by `ForkAgentDialog`. Claude/Grok forks pin the pre-assigned
id at once, so a restart resumes the copy. Pi left out on purpose: a Pi agent owns its session file (`--session` reserved).
Verified: go tests (clisession, store, server fork tests with fake CLIs), node tests, `make ci-scoped` PASS. Live on a
scratch instance with real Claude Code 2.1.280: fork created in `.worktrees/calc-fork` on branch `calc-fork`; the copied
JSONL held the source's message plus the task and `@.picode/drop/…-bug.png`; source session untouched. Model replies
were blocked by the account's usage limit. Earlier probe: `claude -p --resume A --fork-session --session-id B` answered
from A's context. Codex, Grok, OpenCode, Omp: verified against `--help`/source and fake CLIs, not a real run.
visual-review: PASS after one FAIL round (missing menu icon, error far from the Name field, composer focus/lock, cramped
at 390px — all fixed); light, dark and 390px captured; `__picodeOverlayAudit()` ok.
Not done / debts: durable Next and Debts live in `docs/handoff/open/agent-fork.md` (Muse, Hermes/Antigravity, Pi, phone,
restart before first pin, one-line task, lineage display, typed worktree command, live checks beyond Claude Code).
Merge: fast-forward ready.
