# Dashboard and per-CLI usage

## Next

- Dashboard throughput (tokens/s): definition (generation vs turn), per-CLI coverage, UI gates. Codex's `duration_ms`/`time_to_first_token_ms` still unread; Grok's timings shipped.
- **Only Codex reports quota windows.** Claude Code, Grok and the rest have plan limits PiCode never reads, so `Limits` is a one-CLI panel with a coverage footnote. Reading a second CLI's windows is the next real improvement there, not more UI.
- Per-model price overrides (2026-09-22, ADR-0185 deferred them): t3code lets the user set a model's rates; PiCode has no surface for it yet, so a model LiteLLM omits (Grok's `-build` ids, Codex's `gpt-5.3-codex-spark`) stays unpriced.

## Debts

- Grok (2026-09-11): tokens/cost `partial` by design (`usage.json` is new); Agent CLIs rows show model/title but not its cost; dashboard values not screenshot-verified.
- Limits card (2026-09-22): the failed-plan Providers link lands at the top of Agent CLIs; the Providers pane is below the fold.
- Limits card (2026-09-22): "ChatGPT (Codex)" ellipsizes in the card sub at 1280px (full name only on hover).
- Limits card (2026-09-22): prepaid Credits windows (no percentage) are not shown on the card at all.
- Estimates (ADR-0185, 2026-09-22): a resumed Claude Code session keeps its old snapshot as its whole cost, so turns after the last `cost-state` are neither priced nor estimated (one session here: 176 turns, ~$33 at list price).
- Estimates (ADR-0185, 2026-09-22): `costSplit` (Efficiency's "Spent on cache") carries only CLI-reported cost; estimated cache reads are not in it.
- Estimates (ADR-0185, 2026-09-22): long-context premiums are not priced — the table has no `_above_200k` rates for the Claude models in use; $48.6 of $107.7 estimated Claude Code spend here came from turns over 200k context.
