# 2026-09-09 — checklist-line-no-chevron: plan line loses its chevron, gains italics

Shipped: `ChecklistDisclosure` (web/desktop/src/components/WorkspaceRows.jsx)
renders no chevron; an empty 12px slot keeps the text on the card's 39px
text column (folder/branch labels, expanded step text). The line is italic
(`font-style: italic` on `button.ws-check-line`, app.css). Click/keyboard
expand behavior, focus ring, is-done dimming and hover brightening are
unchanged.

Verified: `make ci-scoped` PASS (fmt, vet, hooks, test-js, build); scratch
instance (qa-scratch, seeded checklist via POST /api/agents/{id}/checklist):
collapsed and expanded screenshots read; aria-expanded toggles true/false
across open/close cycles; `__picodeOverlayAudit()` ok.

visual-review: PASS (chkline-card-collapsed.png + chkline-card-expanded.png; card 5/5)

Not done / debts: the expanded panel lines remain upright (JetBrains Mono
upright) — the owner reads them as italic; if they want true italics there
too, it is a one-line follow-up on `.ws-check-list li`.

Merge: fast-forward ready.
