# 2026-09-13 — dashboard-fleet: the Fleet tile measures the machine
Shipped: `fleetStats()` (`web/shared/domain/dashboardStats.js`) counts managed
agents **and** agent-CLI terminals, shells apart; buckets `working` /
`need you` / `idle` / `no signal` fold each kind's own vocabulary
(`agentStatus.js`, `terminalCli.js`). `DashboardView.jsx` renders `N live` +
`live now`, a non-zero-only strip with a hint behind each word, four rows that
open the agent/terminal tab, `+N more` revealing the rest in place; App passes
`terminals` + `onOpen`. ADR-0042 amended (index row too), routes.md updated.
Verified: 38 JS unit tests, `make ci-scoped` PASS, `make close` fast-forward
ready; scratch instance (`qa-scratch.sh dashfleet2`, embedded UI) driven with
agent_browser — rich (5 live, four buckets, expander, row → `#/term/…` and
→ `#/agent/…`), empty (`0 live`), `__picodeOverlayAudit()` `ok: true`.
visual-review: PASS (var/screenshots/dashfleet-{rich,expanded,empty}.png; card 5/5)

## Next up

- Promote `needs you` above the KPI row when more than one thing is blocked
  (Fleet rows already sort and accent those) — needs the ADR-0109 door
  decision before naming the Inbox app there.

## Debts

- `Today` draws one full-width bar under a range picker that promises a chart
  (owner's 2026-09-13 screenshot): hourly buckets for a one-day range, or a
  24 h sparkline + "one day — 7 days for a shape".