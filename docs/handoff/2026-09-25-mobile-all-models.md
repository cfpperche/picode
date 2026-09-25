# 2026-09-25 — feat/mobile-all-models: Scoped models "All models" line aligns with the field
Shipped: owner asked to fix the misaligned "All models" line in Pi › Settings ›
Scoped models. It is a `p.side-empty` (the sidebar's empty-line class) inside
`.set-pats`; its `margin: 4px 4px 8px` put it at x=18 vs the input at 14 (phone
and desktop alike). Both apps' styles/app.css add
`.set-pats > .side-empty { margin-left: 0; margin-right: 0; }`.
Context: the earlier "Choose…" request was measured and is not truncated (the
label is literally "Choose…"), so nothing changed there.
Verified: `make ci-scoped` PASS; measured phone L14 = input L14, desktop L539 =
L539. visual-review: PASS on both. Not done / debts: none. Merge: fast-forward ready.
