# 2026-09-11 — feat/cli-hook-timeout: bound the Grok reporter hook at 10 s

Shipped: adversarial review of feat/cli-approval-state found Grok defaults its
gate-class hook events (PostToolUse, PostToolUseFailure) to a 600 s timeout and
the generated grok.json set none — a hung reporter could stall a turn for ten
minutes before failing open. The builder now stamps `timeout: 10` on those two
handlers; SessionStart/UserPromptSubmit/PermissionRequest/Notification keep
Grok's 5 s default.
Verified: live probes recorded in the review — grok `-p` in a GROK_HOME
sandbox fired `post_tool_use` with the payload the map expects (state
`working`, seq from the event timestamp), claude `-p` accepted the injected
settings (PostToolUseFailure is in Claude's own event enum) and ran
UserPromptSubmit → PostToolUse → Stop, and the installer upgraded a
production-shaped 4-event receipt to the 6-event file, idempotently, still
refusing tampered files. `make ci-scoped` green.
visual-review: n/a.
Not done / debts: Codex PostToolUse still has no live fire (usage limit
resets Sep 16); Claude/Codex one reporter hook per tool call stands.
Merge: fast-forward ready.
