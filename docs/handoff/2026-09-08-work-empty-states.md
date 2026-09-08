# 2026-09-08 — work-empty-states: center mobile Work empty wells

Shipped: Work's Agents / Terminals / Workspaces empty well is a centered
one line + primary create (muted icon, no essay). A search miss stays at
the top with Clear search. Copy drops "free".
Verified: `make close`; scratch `work-empty` at :8471 — empty, search miss,
create sheet, light, 320 px, filled workspaces after seed. `overlayAudit` ok.
visual-review: PASS (agents-empty-dark.png + terminals-empty-dark.png +
agents-search-miss.png + agents-create-sheet.png; card 5/5)
Not done: desktop sidebar still says "No free agents yet."; Now first-run
empty unchanged.
Merge: fast-forward ready.
