# 2026-09-22 — feat/inbox-reply-any-cli: Inbox replies reach every CLI agent, not only Pi

Bug: the owner could never answer an agent's `ask_human` from the Inbox — Reply toasted "The terminal session could not be identified safely". ADR-0184 made every CLI launch an agent, so questions from non-Pi CLIs arrive `sourceKind: agent` with no session path, and the agent reply path (DeliverReply, ADR-0060) assumed Pi and demanded the Pi session file. Every non-Pi agent was affected (Claude Code, Codex, Grok, Hermes, OpenCode, Muse, Antigravity, Omp); the `/api/inbox` respond route (mobile) refused too.

Shipped: `Deps.AnswerAgentQuestion` in `internal/server/agent_answer.go`, used by the Inbox app and the respond route. Owner's call: Omp gets Pi's receiver, the rest the recommended rule. First match wins:
- asker polled with `?wait=1` in the last 30 s → record only (no typing, no double answer)
- Pi, or Omp with a session and running → ADR-0060 receiver/paste with JSONL proof
- Pi not in a terminal → follow_up queue
- other CLI running → record + bracketed paste with verified Enter
- not running, or paste failed → record + a note on the item
POST /api/inbox stamps an agent question with the receiver's last reported session; Omp's launch loads `pi-inbox-reply.ts` via `-e`.

Verified: every row has a test (`internal/server/agent_answer_test.go`, `internal/apps/inbox_test.go`); `make ci-scoped` and `make close` green. Omp 18.2.9 probed live in an isolated tmux: the receiver loads, its hello carries the session file, and it drains/acks an addressed reply file ("different session" refusal). Blind spot: no model call was made, so `sendUserMessage` inside Omp was not exercised live.
visual-review: n/a
Merge: fast-forward ready.

## Next up

- Owner confirms live after deploy: answer an `ask_human` from Claude Code in the Inbox. **Done 2026-09-22:** on 0.5.0+3ca56d4 a restarted Claude Code agent asked, the owner replied from the Inbox, and the answer came back as the `ask_human` result (waiting-asker row).

## Debts

- An `ask_human` MCP process started before the deploy polls without `?wait=1`: until that CLI restarts its MCP server, a reply is both read and typed (double answer). Transitional.
