# 2026-09-13 — feat/thinking-format

Custom endpoints learned how thinking is requested on the wire. The Advanced
section of Add/Edit endpoint now offers `compat.thinkingFormat` — default
(pi's own choice), `reasoning_effort`, `deepseek`, `qwen`, `openrouter`,
`together`, `zai` — and the empty choice removes the key instead of writing an
empty string. `chat-template` / `qwen-chat-template` stay out: both need
`chatTemplateKwargs`/`Args` objects the form does not edit, so the switch
would do nothing.

The formats were never reachable from the form, and a hand-edited one was
worse than invisible: `compat` was decoded as `map[string]bool`, so a string
value failed the whole entry and the provider vanished from the roster, Edit
and the catalog while pi kept using it. `LoadCustomDefinitions` now decodes
compat key by key — the two managed bools, the format string, everything else
untouched — pinned by `TestLoadCustomDefinitionsWithStringCompat`.

Verified on scratch (`qa-scratch thinkfmt`, :8474): create with
`thinkingFormat: "deepseek"` wrote it beside the bools, Edit reopened with the
same select value and chips, `pi auth check` + `--list-models` read the file
green, and the row stayed in the roster after reload (the bug, end to end).
Desktop at 633/1280 and mobile at 480×720/480×520: select visible, sheet caps
and scrolls internally (90dvh), `__picodeOverlayAudit()` ok, nothing clipped.

## Next up

- Same select for the other CLIs' custom endpoints when they get one.
- `chatTemplateKwargs` / `chatTemplateArgs` as an Advanced field pair (P3) —
  the two formats left out above.

## Debts

- Context window, max output and thinking settings still apply to every model
  of a provider instance (per-model editing is P2).
- `supportsReasoningEffort` and the format are independent controls; picking
  `deepseek` does not clear the checkbox, and nothing warns about a
  combination the gateway ignores. Cheap request verification remains P4.
