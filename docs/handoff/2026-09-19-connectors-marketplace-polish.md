# 2026-09-19 — feat/connectors-marketplace-polish: marketplace toolbar, zero-hit and card-density polish

Shipped: Custom server… moved from the floating footer into the toolbar
beside search (owner's ask); zero-hit state points at that button —
"No connectors match — try another word, or use Custom server… above." —
no duplicate (a self-caught double-button defect was fixed in-session).
Card density: letter-tile column 88px (was 132px, scoped ≥641px so the
shared mobile stack wins); Add pinned to the card foot (margin-top:auto)
so feet align across a row (measured equal in the browser); badge
"Local command" → "Local"; group labels "PiCode" / "Catalog" split pinned
picks from synced registry rows.

visual-review: PASS on scratch connmkt2 — desktop marketplace with toolbar
button + PiCode group + aligned feet (audit ok:true, rows 36px), zero-hit
one-liner, mobile 390px single column clean. Screenshots
var/screenshots/connectors-marketplace-polish/ (not committed).

Honest gap: "Catalog" label not seen live — scratch registry sync didn't finish
in-session; render path matches the PiCode group, covered by mcpcatalog tests.

Gates: web npm test 425 pass 0 fail; npm run build ok; make ci-scoped PASS
(fmt,vet,hooks,test-js,build; 4 paths); main merged in once, ff-ready.

## Debts
- Catalog curation filter tuning after first real sync — tracked in docs/handoff/open/connectors-parity.md (first registry sync takes minutes; expected).
