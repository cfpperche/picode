# 2026-09-15 — browser-wipe-dialog

Owner review, item by item: Clear browsing data must open a dialog for
choosing what is cleared.

## Done

- Shell: `btab_clear_data(kinds, since)` — one mask per kind
  (history / cookies+site data / cache / downloads / autofill /
  site settings, passwords never), all-time through ClearBrowsingData,
  a range through ClearBrowsingDataInTimeRange.
- Go: `POST /api/browser/history/clear?since=<RFC3339>` +
  `Store.ClearBrowserHistorySince`, table-tested.
- UI: `.dlg-wipe` dialog — range pills, six checked rows with icons and
  live history count, Cancel / Delete data (primary).
- Visual: audit ok after centring the tall dialog (`top: 50%` + a
  86dvh cap) — the stock 42% anchor clipped it off the top.
- Fixed the shipped mask bug: the old one-click clear included
  general autofill and missed WebView2 browsing history.
