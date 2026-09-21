# 2026-09-20 — feat/inspector-sheet: Inspector rises as the standard bottom sheet
Shipped: owner asked for the standard bottom-sheet idiom plus a header-wrap fix. The right drawer stacked its header — the sheet portals to body, outside #m-app, so --m-tap/--m-pad never resolved and the m-head grid collapsed to one column. The Inspector now rises as the standard bottom sheet (default Vaul side) with the m-* metrics re-declared on the sheet scope; MobileSheet keeps its generic side prop.
Verified: make ci-scoped PASS; visual-review PASS (sheet-open.png: one-row header, full-width sheet). Blind spot: sheet drag/swipe feel on a physical device untested (headless QA only).
visual-review: PASS
Merge: fast-forward ready.

## Debts

- Sheet drag/swipe feel unverified on a physical device — headless QA only
