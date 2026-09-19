# 2026-09-19 — fix/inbox-app-answer

First live `ask_human` from Claude Code (N1): the question reached the
Inbox, but "Send reply" in the Inbox app answered "computer-use is not
running pi, so the reply cannot be delivered". The app does not go through
the inbox route; it calls the terminal delivery directly (`apps.go`), and
only the route had N1's recorded-answer branch. `Deps.AnswerTerminalQuestion`
is now the one rule for both: pi terminals get delivery, any other terminal
gets the answer recorded and the item closed. Test rows for both callers.

## Next up
- Owner: `make deploy`, then answer the open question from the Inbox; the Claude Code terminal reads it back.
