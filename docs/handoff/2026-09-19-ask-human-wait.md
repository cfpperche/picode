# 2026-09-19 — feat/ask-human-wait (ADR-0154, N1 follow-up)

Owner: `ask_human` must not depend on the agent re-calling with the item
id. Facts (Claude Code MCP docs): a tool call's wall-clock limit is ~28 h
by default, and a separate idle window aborts a call that sends neither a
response nor a progress notification. So `internal/mcptool` now runs each
`tools/call` on its own goroutine (ping, list and cancellation get
through), sends `notifications/progress` with the client's
`_meta.progressToken` every 15 s while a call waits, honours
`notifications/cancelled` (the wait ends, the item stays open), and
`ask_human` waits 8 h by default (`PICODE_ASK_WAIT`). Tests: concurrency,
progress with the token, cancellation.

Proof: `go test ./internal/mcptool`; live after `make deploy` and a Claude
Code session restart: ask, leave it for minutes, answer from the mobile
Inbox, the answer returns without any re-call.

## Next up
- Owner: `make deploy` + restart the Claude Code session, then the live wait above.

## Debts
- Codex/OpenCode idle windows unmeasured; if one aborts sooner, the cancel path names the item.
