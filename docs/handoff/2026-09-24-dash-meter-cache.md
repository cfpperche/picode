# 2026-09-24 — dash-meter-cache

Dashboard Fase 2: `climetrics.WindowCache` + `AggregateCached` keep each
meter's window per `root|range`; `statsForRange` uses it and serialises
concurrent misses per range (`statsFlight`).

- Measured on the owner's data: warm 7-day poll 275 ms → 14 ms (190 ms when
  a large transcript moved), identical totals. OpenCode's database was
  ~270 ms of every old poll.
- Stale-while-revalidate dropped on purpose (reason in
  `docs/architecture/climetrics.md`).
