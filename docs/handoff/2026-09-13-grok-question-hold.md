# 2026-09-13 — feat/grok-question-hold: a Grok question card keeps Needs you

Shipped: Grok's `ask_user_question` card notifies `elicitation_dialog`, which
the hook map dropped (it mapped only `permission_prompt`), so the row stayed
Working for the whole wait. The notification now reports `needs-you` with
`attention: question`; the state report carries it and the hold is released
only by the question tool's own completion (`attention: answered`) or a
settled lifecycle report — a sibling tool's `PostToolUse` in the same parallel
batch never releases it (`internal/server/term_state.go`, `term_wiring.go`,
`native_session.go`). A held question's event can be older than a sibling
completion, so it is exempt from the native ordering fence (map and server).
The Grok hook set also dropped `PermissionRequest`: Grok 1.0.30 has no such
event (it has `PermissionDenied`) and skipped the group silently.

Verified: live probe of Grok 1.0.30 in tmux with a temporary hook logger — the
card fires `elicitation_dialog` ("User question requested") ~10ms after it
appears, before the batch siblings finish; dismissal fires
`PostToolUse ask_user_question` too. Unit tests: map reports, registry decision
table, and the hold across batch completions through the real endpoint.
Changelog fragment `docs/changelog.d/grok-question-hold.md` present.

visual-review: n/a (no UI change; the row's Needs you chip already exists)
Not done / debts: a *permission* hold still lacks a release identity, so a
sibling tool's `PostToolUse` can clear a permission needs-you (documented in
`docs/architecture/terminal-bridge.md`).
Merge: fast-forward ready (`cd /home/goat/picode && git merge --ff-only feat/grok-question-hold && make ci`). Deploy is the owner's call.
