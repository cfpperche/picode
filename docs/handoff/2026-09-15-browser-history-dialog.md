# 2026-09-15 — browser-history-dialog

Owner review: keep Browsing history in the dialog (no sub-route) but the
data must match or beat the reference.

## Done

- Store: `DeleteBrowserVisits(ids)` (one transaction, stale ids ignored)
  + test; server: `POST /api/browser/history/delete` + test.
- Dialog: search (server-side, 250ms debounce), day groups with collapse
  and counts, favicon from the site's own /favicon.ico over https with a
  letter fallback, visit time, per-row ⋯ menu (Copy link / Remove), bulk
  selection with "N selected → Remove", "Clear browsing data" on the
  section header, and an empty state per case (no history vs no match).
- Visual: read three states (populated/collapsed days, selected, empty
  search); overlay audit ok.
## Notes

- Visits older than the loaded page (200) are not shown; the server has
  no cursor yet — debt only if history grows past that in practice.
- Next: Downloads section, Browser permissions (camera/mic), Developer
  mode.
