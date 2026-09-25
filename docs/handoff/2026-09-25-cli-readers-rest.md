# 2026-09-25 — cli-readers-rest: Grok's Model picker reads its own cache

Shipped (88fe2947b): a Grok climodels reader that runs nothing — it reads
`$GROK_HOME/models_cache.json` (Grok's own per-account list) and fills the Model
field's Choose… (`models.default`). Hidden rows are dropped; `reasoning_efforts`
become thinking levels; a missing file answers "sign in and open Grok once".
`docs/architecture/cli-settings.md` updated.

- Verified on a scratch instance: the API returned 4 models, the picker listed
  them, choosing Grok 4.6 wrote the field, the no-file state shows the message;
  visual review PASS (overlay audit ok).
- Seen, not fixed (pre-existing shared layout): "Use inherited" makes an
  overridden row wider than its neighbours; the picker's error state has one
  line but no action button.

## Next up

- Claude Code and Antigravity still have no reader: listing needs running the
  CLI (claude rewrites `~/.claude.json`; agy refreshes its OAuth token).
  Benchmark 2026-09-25: t3code and paseo ship a hard-coded versioned Claude
  manifest gated by `claude --version` + user settings.json model/env
  overrides; orca spawns claude with stream-json control_request `list_models`
  (falls back to haiku/sonnet/opus) and `agy models`; t3code gets Antigravity
  models from ACP session/new. Decision pending with the owner.
