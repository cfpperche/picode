# 2026-09-13 — feat/thinking-format

Custom endpoints learned how thinking is requested on the wire: the Advanced
section of Add/Edit now offers `compat.thinkingFormat` (default, the pi's own
choice, plus `reasoning_effort`, `deepseek`, `qwen`, `openrouter`, `together`,
`zai`), and the empty choice removes the key instead of writing `""`.
`chat-template` / `qwen-chat-template` stay out — both need
`chatTemplateKwargs`/`Args` objects this form does not edit.

The formats were never reachable from the form, and a hand-edited one was worse
than invisible: `compat` was decoded as `map[string]bool`, so a string value
failed the whole entry and the provider vanished from the roster, Edit and the
catalog while pi kept using it. `LoadCustomDefinitions` now decodes compat key
by key (`TestLoadCustomDefinitionsWithStringCompat`).

Also fixed here, both `main`-owned test races in the server package's way of a
green gate: `TestPaneRootSurvivesSIGHUP` (signalled before the pane's trap
existed; 5/5 failures here, 4/5 on `main` at load ~5) and
`TestCLITerminalResumeDecisionTable` (`waitCLIFile` returned the creation
launch's bytes; 2/2 in sharded runs). Measurements and the fix rationale:
`docs/handoff/open/terminal.md`. Still red after this branch:
`TestPeerStopStubbornChildStaysPending` — deterministic, and two live sessions
are in that code.

Verified on scratch (`thinkfmt`, :8474): create wrote
`"thinkingFormat": "deepseek"` beside the compat bools, Edit reopened on it,
`pi auth check` + `--list-models` read the file green, the row survived reload
(the bug, end to end). Desktop and mobile (480×720, 480×520): select visible,
sheet scrolls inside, overlay audit ok, nothing clipped.

## Next up

- `chatTemplateKwargs` / `chatTemplateArgs` as an Advanced field pair (P3).

## Debts

- Context window, max output and thinking settings still apply to every model
  of a provider instance (per-model editing is P2).
- `supportsReasoningEffort` and the format are independent controls; nothing
  warns about a combination the gateway ignores. Cheap request verify is P4.