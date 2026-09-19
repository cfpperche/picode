# 2026-09-19 — feat/picode-mcp-n1 (ADR-0154, N1)

`picode mcp` gains two families, mirroring the pi packages:
`inbox` (`notify_human`, `ask_human`) and `checklist`, in
`internal/mcptool/{inbox,checklist}.go` with the package tests ported as
goldens. Cards "PiCode · Inbox" and "PiCode · Checklist" in Connectors;
Inbox and Checklist switches under PiCode tools in Launch settings.

The one semantic difference, by necessity: a guest CLI has no pi receiver
for the human's reply, so `ask_human` waits on the item — polling
`GET /api/inbox/{id}` (new route) every 2 s up to `PICODE_ASK_WAIT` s (90)
— and returns the answer; at the deadline it names the item to resume with.
Daemon: a question from a terminal not running pi is answered by recording
the response on the item (`handleRespondInbox`), where delivery used to
refuse. pi's checklist gate and reminder have no MCP equivalent (said so).

Proof: Go tests in mcptool, mcp and server (Inbox), `make ci-scoped`. Live
proof pending: Claude Code with Inbox ticked → `ask_human` → answer in the
Inbox → the answer returns to the CLI; `checklist` → the card shows the step.

## Next up
- Owner: `make deploy`, then the two live checks above from a Claude Code terminal.

## Debts
- N2: launch injection for Grok, Hermes, Muse, Antigravity, Omp where a mechanism exists.
