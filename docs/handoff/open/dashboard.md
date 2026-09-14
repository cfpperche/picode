# Dashboard and per-CLI usage

## Next

- **`Today` draws one full-width bar.** With a single day in the series the daily chart is a solid block under a range picker that promises a chart — the owner's 2026-09-13 screenshot. Either bucket by hour for a one-day range (climetrics has the message timestamps) or replace the chart with a 24 h sparkline plus "one day — 7 days for a shape".
- **Promote `needs you` above the KPI row** when more than one thing is blocked on the reader (the Fleet tile already sorts and accents them). Blocked on a decision, not on code: naming the Inbox app in that line would add a door ADR-0109's closed list does not have.
- Dashboard throughput (tokens/s): definition (generation vs turn), per-CLI coverage, UI gates. Codex's `duration_ms`/`time_to_first_token_ms` still unread; Grok's timings shipped.

## Debts

- Grok (2026-09-11): tokens/cost `partial` by design (`usage.json` is new); Agent CLIs rows show model/title but not its cost; dashboard values not screenshot-verified.
