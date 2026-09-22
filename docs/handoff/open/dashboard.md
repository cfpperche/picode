# Dashboard and per-CLI usage

## Next

- Dashboard Fase 2 (2026-09-17, perf Fase 1 landed): per-meter fingerprint cache + stale-while-revalidate, so one dirty CLI stops invalidating all six and polls stay warm.
- Dashboard throughput (tokens/s): definition (generation vs turn), per-CLI coverage, UI gates. Codex's `duration_ms`/`time_to_first_token_ms` still unread; Grok's timings shipped.
- **Only Codex reports quota windows.** Claude Code, Grok and the rest have plan limits PiCode never reads, so `Limits` is a one-CLI panel with a coverage footnote. Reading a second CLI's windows is the next real improvement there, not more UI.
- Per-model price overrides (2026-09-22, ADR-0185 deferred them): t3code lets the user set a model's rates; PiCode has no surface for it yet, so a model LiteLLM omits (Grok's `-build` ids, Codex's `gpt-5.3-codex-spark`) stays unpriced.

## Debts

- Grok (2026-09-11): tokens/cost `partial` by design (`usage.json` is new); Agent CLIs rows show model/title but not its cost; dashboard values not screenshot-verified.
- Limits card (2026-09-22): the failed-plan Providers link lands at the top of Agent CLIs; the Providers pane is below the fold.
- Limits card (2026-09-22): "ChatGPT (Codex)" ellipsizes in the card sub at 1280px (full name only on hover).
- Limits card (2026-09-22): prepaid Credits windows (no percentage) are not shown on the card at all.
