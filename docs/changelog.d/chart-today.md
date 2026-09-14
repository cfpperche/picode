### Fixed
- **The `Today` range drew one bar stretched across the whole chart.** A
  one-day window was one calendar day, so the panel showed a single
  full-width block under a range picker that promised a chart. `Today` is now
  bucketed by hour (24 bars, labelled `midnight` … `11pm`, panel titled
  `Hourly`), and every longer range keeps one bar per day.
