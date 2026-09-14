# 2026-09-13 — feat/thinking-format

Custom endpoints learned how thinking is requested on the wire: Add/Edit's
Advanced section offers `compat.thinkingFormat` (pi's default, `reasoning_effort`,
`deepseek`, `qwen`, `openrouter`, `together`, `zai`) and the empty choice
removes the key. `chat-template` / `qwen-chat-template` stay out: both need
`chatTemplateKwargs`/`Args` objects this form does not edit.

A hand-edited format used to be worse than unreachable: `compat` was decoded as
`map[string]bool`, so a string value failed the entry and the provider vanished
from the roster, Edit and the catalog while pi kept using it. It is now decoded
key by key (`TestLoadCustomDefinitionsWithStringCompat`).

Two `main`-owned test races were fixed here on the test side, both blocking a
green gate: `TestPaneRootSurvivesSIGHUP` (signalled before the pane's trap
existed; 5/5 failures here, 4/5 on `main` at load ~5) and
`TestCLITerminalResumeDecisionTable` (`waitCLIFile` returned the creation
launch's bytes; 2/2 sharded) — measurements in `docs/handoff/open/terminal.md`.
`TestPeerStopStubbornChildStaysPending` remains red and deterministic.

Verified on scratch (`thinkfmt`, :8474): create wrote `"thinkingFormat":"deepseek"`
beside the compat bools, Edit reopened on it, `pi auth check` + `--list-models`
read green, the row survived reload. Desktop and mobile (480×720, 480×520):
select visible, sheet scrolls inside, overlay audit ok, nothing clipped.

## Next up

- `chatTemplateKwargs` / `chatTemplateArgs` as an Advanced field pair (P3).

## Debts

- Context window, max output and thinking settings apply to every model of an
  instance (per-model editing is P2); `supportsReasoningEffort` and the format
  are independent controls, and nothing warns about a combination the gateway
  ignores (cheap request verify is P4).
