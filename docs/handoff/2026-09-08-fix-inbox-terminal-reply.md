# 2026-09-08 — fix-inbox-terminal-reply: Inbox replies reach terminal-sourced ask_human

Shipped: pi-inbox 0.2.0 stamps `sourceKind: "terminal"` from `PICODE_TERM_ID`;
`Deps.DeliverTerminalReply` (internal/server/terminal_ask.go) answers those
items through the terminal's receiver with the item's exact session
(ADR-0060 contract, no task row; reopen via new `store.ReopenInboxItem`).
`RespondAndForward` refuses channelless blocking replies (`ErrNoReplyChannel`)
— replying can no longer close an item while nothing was sent (ADR-0037
amendment 2026-09-09). Raw route + app action both route terminals; Host
gains `DeliverTerminalReply`; delivery hint + named terminal label in the
Inbox view.

Verified: make ci-scoped PASS (fmt/vet/hooks/go/js); full store+apps+server
suites green; new receiver-delivery, raw-route, refusal-matrix, row-timeout
reopen and app-routing tests. Root cause confirmed against production data
(three `system`-sourced questions, state done, never delivered).

visual-review: n/a (no JSX/CSS; server-rendered copy only)
Not done / debts: pi-inbox 0.2.0 must be installed in each pi session
(`pi install -l …/packages/pi-inbox`) before new questions carry the
terminal identity; 0.1.x items (`pi (unmanaged)`) are refused with honest
copy, answered by hand. Daemon death between park and JSONL row leaves a
done item with an unproven reply (same accepted gap as the terminal ask).
Merge: fast-forward ready.
