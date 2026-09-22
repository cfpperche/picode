# 2026-09-21 — changes-typography: Changes tab reads in Files' typography
Shipped: renamed the grouped Changes view's classes `.insp-group*` → `.insp-checkout*` (InspectorChanges.jsx, app.css). They had collided with the Servers panel's section-label style (10px, uppercase, .06em tracking, weight 600), which inheritance carried into every branch header and file row. Header now rides the Files tree's scale: 13px, weight-600 branch name, 11px tabular counts; rows are byte-identical to `.ft-row`. Servers labels untouched; mobile uses its own `m-insp-group`.
Verified: computed styles read live on a qa-scratch instance before/after (row fw 600→400, ls 0.6px→normal, header uppercase→none); screenshots read for Changes groups, Files, Servers, and the "This agent" empty scope; `window.__picodeOverlayAudit()` ok; ci-scoped PASS. Blind spot: not checked in the mobile Inspector screen (it does not import the edited component).
visual-review: PASS (ctypo-after-changes.png + cypo-after-files.png read; card 5/5)
Not done / debts: none new.
Merge: fast-forward ready.

## Next up

- Owner may `make deploy` when convenient; production at screenshot time (0.3.1) predates several landings, so the before-state there can differ from this branch's base.
