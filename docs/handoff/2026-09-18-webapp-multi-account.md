# 2026-09-18 — webapp-multi-account: two accounts of one service, discoverable

Shipped: owner feedback — multi-account hidden behind a hand-typed
fragment was undiscoverable. Migration 057 rebuilds `webapps` without
the UNIQUE(url) constraint (plain index keeps the by-url lookup); the
install request gains `allowDuplicate`; the tile menu gains **Add
another account** (pre-filled `<name> (2)`, pre-consented — no error
round trip) and the duplicate conflict dialog gains **Add as new
account** beside Open existing. Each row is one partition (ADR-0153):
own logins, own data, remove-one-touches-nothing.
Verified: go suites (same-URL coexistence · allowDuplicate second
account · 409 default kept · invariant rows), 9 JS tests incl.
allowDuplicate pass-through; scratch QA — two Excalidraw accounts
installed via the menu, both `partitioned:true`, dialog screenshot read.
visual-review: PASS (dialog layout; overlayAudit flow unchanged).
Merge: fast-forward ready.

## Next up

- Owner on Windows: the two-tile isolation proof (separate logins, restart, clear-data) — same checklist as ADR-0153.

## Debts

- docs/handoff/open/installed-webapps.md: standalone window stays refused (unchanged).
