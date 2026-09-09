# 2026-09-09 — checklist-align: plan line text on the card's metadata column

Shipped: dropped the ghost 12px slot from the sidebar checklist disclosure
(`web/desktop/src/components/WorkspaceRows.jsx`, `web/desktop/src/styles/app.css`).
The collapsed plan text now starts on the card's 23px metadata column — the same
left edge as the title, subtitle and the folder/branch icons. The expanded list
keeps its idiom: glyphs at 23px, step text at 39px (folder-label column).

Verified: `make web`; `make ci-scoped` PASS (fmt, vet, hooks, test-js, build).
Scratch instance `chk-align` with a seeded 7-step checklist: measured
`.ws-check-text` = `.ws-title` = 50px; step glyph 50 / step text 66. Expand on a
real click, collapse stays closed, `__picodeOverlayAudit()` ok.

visual-review: PASS (chk-expanded.png + chk-collapsed-expand-click.png read, plus
3x zoom crops of the card).

Not done / debts: none.

Merge: fast-forward ready.
