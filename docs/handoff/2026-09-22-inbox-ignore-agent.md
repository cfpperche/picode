# 2026-09-22 — feat/inbox-ignore-agent: Ignore closes an interactive agent's question

Bug (owner, live on 0.5.0+3ca56d4): the stale `ask_human` item from before the reply fix could not even be ignored — Ignore toasted "The terminal session could not be identified safely". The Inbox app still sent Ignore on an interactive agent's item through `DeliverReply` (ADR-0060), whose session check refuses every non-Pi agent; `feat/inbox-reply-any-cli` had excluded Ignore from its new door but not from this older branch.

Shipped: `internal/apps/inbox.go` skips the terminal delivery for Ignore, so it falls through to the local close (`RespondAndForward` records `ignore`, queues nothing). Test: `TestInboxIgnoreInteractiveAgentClosesLocally`. The owner's stuck item was closed through `POST /api/inbox/{id}/respond {"verb":"ignore"}`, which already took that path.

Verified: `make close` green.
visual-review: n/a
Merge: fast-forward ready.
