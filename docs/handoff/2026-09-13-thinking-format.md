# 2026-09-13 — feat/thinking-format

Custom endpoints learned how thinking is requested on the wire: Add/Edit's
Advanced section offers `compat.thinkingFormat` (pi's default, `reasoning_effort`,
`deepseek`, `qwen`, `openrouter`, `together`, `zai`) and the empty choice
removes the key. `chat-template` / `qwen-chat-template` stay out: both need
`chatTemplateKwargs`/`Args` objects this form does not edit. The same change
fixed a string `compat` value hiding the provider from the GUI (changelog).

Two `main`-owned test races were fixed here on the test side, both blocking a
green gate: `TestPaneRootSurvivesSIGHUP` (signalled before the pane's trap
existed; 5/5 failures here, 4/5 on `main` at load ~5) and
`TestCLITerminalResumeDecisionTable` (`waitCLIFile` returned the creation
launch's bytes; 2/2 sharded). Measurements: `docs/handoff/open/terminal.md`.
`TestPeerStopStubbornChildStaysPending` remains red and deterministic.

Verified on scratch: `thinkingFormat: "deepseek"` written beside the compat
bools, Edit reopened on it, `pi auth check` + `--list-models` green, row
survived reload; desktop and mobile screenshots read, overlay audit ok.

## Next up

- `chatTemplateKwargs` / `chatTemplateArgs` as an Advanced field pair (P3).

## Debts

- Thinking settings apply to every model of an instance (per-model editing is
  P2); `supportsReasoningEffort` and the format are independent controls, and
  nothing warns about a combination the gateway ignores (verify is P4).
