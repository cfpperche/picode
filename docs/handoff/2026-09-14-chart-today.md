# 2026-09-14 — chart-today: a one-day window is bucketed by hour
Shipped: `climetrics.Request.Hourly` (+ `SeriesKey`, `LocOf`) and `fillSeries`
stepping by real hours for `range=today` only; the server sets it from the
range (`internal/server/session_stats.go`). `SeriesKey` writes `2026-09-14` or
`2026-09-14T14`, so the key states its own granularity and older clients print
the raw key rather than a wrong label. Web: `dayLabel` → `bucketLabel`
(`web/shared/domain/dashboardStats.js`), `DailyChart.jsx` titles the panel
`Hourly` and its aria-label `per hour` from the same shape test. ADR-0042's
2026-09-14 amendment, index row, routes.md, changelog fragment.
Verified: `make ci-scoped` PASS; Go tests pin the decision table (day window
fills days, hourly fills 24, keys carry the zone, a repeated DST hour merges —
`TestSeriesBucketsByDayOrByHour`, `TestSeriesKeysCarryTheirOwnGranularity`,
`TestHourlySeriesMergesARepeatedDSTHour`, `TestHandleSessionStatsBucketsTodayByHour`
with a pi fixture at a non-current hour, so mtime bucketing cannot pass by
luck); 39 JS tests. Scratch instance `qa-scratch.sh chart1` with a seeded pi
session: `range=today` 24 buckets with the six spending hours in place,
`range=7d`/`30d` day buckets unchanged; `__picodeOverlayAudit()` `ok: true`,
aria read back as "Spend per hour".
visual-review: PASS (var/screenshots/chart-today-{hourly,week,empty}.png; card 5/5)

## Next up

- Promote `needs you` above the KPI row when more than one thing is blocked
  (Fleet rows already sort and accent those) — needs the ADR-0109 door
  decision before naming the Inbox app there.
