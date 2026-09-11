# 2026-09-11 — feat/cli-approval-state: the row stops saying "Needs you" while the agent works

Shipped: tool lifecycle as the permission-resume signal. `hookMapPy` maps
`PostToolUse`/`PostToolUseFailure`/`PreToolUse` to `working`; Grok
(`nativeGrokHookEvents`, `native_integration.go`), Claude (`claudeHookEvents`)
and Codex (`codexHookSpecs`) register the tool events each one supports.
Grok's non-permission notifications (`task_complete`, …) no longer set a
false `needs-you`. Docs: `docs/architecture/terminal-bridge.md` (CLI resume
table), `docs-site/guide/terminal-status.md`; fragment
`docs/changelog.d/cli-approval-state.md`.
Verified: `make close` green (ci-scoped: fmt, vet, hooks, go, docs).
`TestToolActivityResumesAfterPermissionPrompt` drives the real handler
(`needs-you` seq N → `working` seq N+1); map rows cover the three CLIs;
Codex's `post_tool_use` trust hash was captured from `codex app-server
hooks/list` and is asserted in `TestCodexHookHashMatchesCodexFingerprint`.
Root cause: Grok logged `permission_requested exit_plan_mode` at 15:19:20 —
exactly the state timestamp — and no report after the approval; every
approval prompt in the three hook-driven CLIs had no "resolved" event.
visual-review: n/a (no UI change).
Not done / debts: no live Codex `PostToolUse` fire — the account hit its usage
limit, so config acceptance and the trust hash were verified through the
app-server schema and `hooks/list` instead. A running Grok session keeps its
old hook file until relaunch; the installed `~/.grok/hooks/picode-native.json`
updates at the next wrapper launch. Claude/Codex now pay one reporter hook per
tool call (PostToolUse only; `PreToolUse` is mapped but deliberately not
registered — it fires before the permission gate, so it cannot report the
answer).
Merge: fast-forward ready.
