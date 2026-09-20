# 2026-09-18 — webapp-refresh: Refresh re-resolves a tile's identity

Shipped: the installed-webapp tile menu gains **Refresh**
(`POST /api/webapps/{id}/refresh`). The daemon re-resolves the stored URL
with the same bounded fetch and rewrites icon + `start_url`/`scope`/
`display`/`theme_color`; **name and url are never touched** (the user's
label and chosen identity). A failed icon fetch keeps the old icon; an
unreachable site returns 502 and leaves the row untouched; a site that
dropped its manifest clears the manifest fields and falls back to
favicon. The tile icon cache-busts (`?v=` timestamp bumped per refresh)
so the new icon shows without a reload. Store:
`UpdateWebappMetadata` announcing `webapp.updated` in-transaction
(invariant row added). Plan: `docs/plans/installed-webapps-v2.md`
(feature 1 of 2; per-app WebView2 partitions follow with their own ADR).
Verified: go store+server suites — refresh replaces identity keeping
name/url · invalid display refused · missing row 404 · site-down 502 with
row untouched · invariant row; 8 JS tests; scratch QA with an alternating
manifest fixture — two refreshes flip fields both directions and the tile
icon swaps live (blue → red, screenshots read); build green.
visual-review: PASS (refresh-swap.png read; toast + swapped icon).
Merge: fast-forward ready.

## Next up

- Feature 2 of the plan: per-webapp WebView2 partitions + ADR-0153 (`docs/plans/installed-webapps-v2.md`).
- Owner: Refresh on a real webapp tile (GitHub) from the desktop shell.

## Debts

- docs/handoff/open/installed-webapps.md: standalone window stays refused; per-webapp partitions move from debt to in-progress with this branch.
