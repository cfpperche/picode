# 2026-09-25 — claude-models: Claude Code's Model picker lists its own models

Shipped (c9461e840): a Claude Code climodels reader (`internal/climodels/claude.go`,
adapted from orca) that sends one `list_models` control request over
`claude -p --input-format stream-json --output-format stream-json --verbose`
with no API turn. It runs with `CLAUDE_CONFIG_DIR` and cwd in a throwaway temp
dir (removed after) and an env stripped of `CLAUDECODE`/`CLAUDE_CODE_*`, so the
owner's `~/.claude.json` is never written and no project hooks run; that file
is only read for `additionalModelOptionsCache` extras. `default` and `disabled`
rows are dropped. Fills the `model` field's Choose…. Owner asked for the
throwaway dir on 2026-09-25, after Grok's reader. cli-settings.md updated.

- Measured signed out: 0.6–0.7 s, 4 models (opus[1m], claude-fable-5-1,
  sonnet, haiku) with effort levels. Scratch QA: the picker listed them,
  choosing Sonnet wrote `"model": "sonnet"` to the scratch home settings.json;
  visual review PASS, overlay audit ok.
- Not measured: whether a signed-in account's catalog differs from the
  signed-out one beyond the cached extras. The fingerprint keys only on the
  binary (`~/.claude.json` changes constantly), so account extras refresh on
  the 10-minute age bound or Refresh.

## Next up

- Antigravity has no reader (`agy models` hits the network, refreshed the OAuth
  token), Hermes neither: try the throwaway dir for agy only if it lists signed out.
