# 2026-09-25 — feat/mobile-archived-check: phone "Include archived" is a real tap row
Shipped: owner asked to fix the phone missions list's "Include archived"
checkbox (a bare ~13px box beside 12px grey text, small tap target). The label
gets class `mission-archived` (web/mobile/src/screens/Missions.jsx);
web/shared/styles/missions.css scopes `.m-missions .mission-archived`
(phone-only) to a flex row, min-height 44px, 13px text, and a 20×20
accent-colour checkbox. Desktop untouched.
Verified: `make ci-scoped` PASS; scratch 390×844: row 362×44 at the 14px
gutter, box 20×20, a real pointer tap on the text toggled it on.
visual-review: PASS (off and on states).
Not done / debts: none. Merge: landed on main c859624e2; deployed by the owner 2026-09-25.
