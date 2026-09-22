# ADR-0185: Unpriced tokens are estimated from LiteLLM's price table, and say so

- **Status**: accepted (owner, 2026-09-22)
- **Date**: 2026-09-22
- **Boundary**: process + persistence — PiCode now fetches a public file from
  the network on its own schedule (`raw.githubusercontent.com`, no
  credential) and keeps a copy in the data dir (`var/litellm-prices.json`).
- **Supersedes**: the "A model price table (ccusage's `calculate` mode)" row of
  "What PiCode explicitly does not copy" in
  [the ADR-0097 study](../benchmarks/2026-09-07-cross-cli-agent-telemetry.md).
  ADR-0097 itself stands: silence is still a value, `Billing` is still never
  inferred.
- **Study**: the 2026-09-22 addendum of that study (t3code's usage reader).

## Context

The dashboard prices only what a CLI prices itself. On this machine, over 30
days (2026-09-22):

| CLI | What it writes | What the dashboard showed |
|---|---|---|
| Codex | 1.4B tokens, never a price | `—`, left out of Spend |
| Claude Code | a cost snapshot on 67 of 87 sessions; a live session has none | a floor labelled `partial` |

The study refused a price table because "it ages silently: the day a vendor
changes a price, every number stays plausible and wrong". t3code (at
`d7819c18`) and ccusage both price the same stores with LiteLLM's
`model_prices_and_context_window.json`. It is a community-maintained file of
per-token rates, refreshed upstream as vendors change prices. The table lists
every Claude Code and Codex model this machine used in the window. Both tools
answer the aging risk by fetching the table rather than shipping a copy, and
by keeping the CLI's own figure whenever one exists.

## Decision

A turn the CLI left unpriced is priced at list rate from LiteLLM's table:
uncached input, cache read, cache write and output, each at its own rate. A
figure the CLI wrote always wins. Today "unpriced" means every Codex turn and
every turn of a Claude Code session with no cost snapshot in any of its
files.

The estimate is never blended silently. It travels as `estimated` beside
`cost` on the window totals, on each CLI row and on each model row, and the
surface labels it as an estimate. A model the table does not list stays
unpriced and is counted in the coverage note. A bare family name
(`opus`, `<synthetic>`) is never guessed.

`internal/pricing` fetches the file at boot and every 24 hours, off the
request path. It keeps the last good copy in `var/litellm-prices.json` and
serves that copy when the network fails. With no copy at all, nothing is
estimated. `PICODE_PRICE_TABLE_URL` points the fetch elsewhere; `off`
disables it. The table's content hash is part of the stats cache key, so a
new table recomputes the window.

## Consequences

- Codex and live Claude Code sessions count in Spend. What a CLI stated
  and what PiCode estimated stay separable on every row.
- An estimate is list price. Under a subscription (`Billing:
  subscription`) it reads as "what this would have cost on the API", the
  same meaning Claude Code's own snapshot already has. It is not a bill.
- The table is base tier only. Long-context, priority and batch tiers are
  not in the transcripts, so a 1M-context session is under-estimated. This
  is the same limitation t3code and ccusage accept.
- PiCode now makes one unauthenticated outbound request a day. A machine
  that must not can set `PICODE_PRICE_TABLE_URL=off` and loses only the
  estimates.
- **If the table is wrong**, estimates are wrong while the CLI's own
  figures are untouched. The `estimated` split names exactly which dollars
  to distrust.

## Alternatives considered

- **Keep refusing.** Codex keeps showing `—` and Claude Code's spend stays
  a floor. The owner asked for the opposite, having seen t3code's usage
  page price all three stores.
- **Ship a price table in the binary.** That is exactly what the study
  refused: it ages the day a vendor changes a price, and nobody notices.
- **Fetch on each dashboard request.** It puts the network on the request
  path and makes a page load depend on GitHub.
- **Per-model overrides in the UI (t3code's settings).** Deferred. There is
  no surface to edit them, and a hand-edited file would be a setting a
  terminal-averse user cannot reach. Tracked in
  `docs/handoff/open/dashboard.md`.
