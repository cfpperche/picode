# 2026-09-13 — feat/thinking-format

Custom endpoints learned how thinking is requested on the wire. The Advanced
section of Add/Edit endpoint offers `compat.thinkingFormat` — default (pi's own
choice), `reasoning_effort`, `deepseek`, `qwen`, `openrouter`, `together`,
`zai` — and the empty choice removes the key instead of writing an empty
string. `chat-template` / `qwen-chat-template` stay out: both need
`chatTemplateKwargs`/`Args` objects the form does not edit.

The formats were never reachable from the form, and a hand-edited one was
worse than invisible: `compat` was decoded as `map[string]bool`, so a string
value failed the whole entry and the provider vanished from the roster, Edit
and the catalog while pi kept using it. `LoadCustomDefinitions` now decodes
compat key by key, pinned by `TestLoadCustomDefinitionsWithStringCompat`.

Two main-owned test races were diagnosed and fixed on the test side, both
unrelated to this feature but in the way of a green `make ci` — see
`docs/handoff/open/terminal.md` for the measurements:
`TestPaneRootSurvivesSIGHUP` signalled the pane root before its `trap` existed
(5/5 failures here, 4/5 on `main` at load ~5); `TestCLITerminalResumeDecisionTable`
read the creation launch's `--default` bytes back because `waitCLIFile` returns
as soon as the path exists (2/2 in sharded runs, always shard 3).

Verified on scratch (`qa-scratch thinkfmt`, :8474): create with
`thinkingFormat: "deepseek"` wrote it beside the bools, Edit reopened on the
same select value and chips, `pi auth check` + `--list-models` read the file
green, and the row survived reload (the bug, end to end). Desktop and mobile
at 480×720 / 480×520: select visible, sheet scrolls internally, overlay audit
ok, nothing clipped. Still red after this branch: `TestPeerStopStubbornChildStaysPending`
(deterministic, `main`-owned, two live sessions in that code).

## Next up

- `chatTemplateKwargs` / `chatTemplateArgs` as an Advanced field pair (P3) —
  the two formats left out above.

## Debts

- Context window, max output and thinking settings still apply to every model
  of a provider instance (per-model editing is P2).
- `supportsReasoningEffort` and the format are independent controls; nothing
  warns about a combination the gateway ignores. Cheap request verify is P4.