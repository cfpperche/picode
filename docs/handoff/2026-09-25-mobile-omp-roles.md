# 2026-09-25 — feat/mobile-omp-roles: Omp Models role controls fill the row on the phone
Shipped: owner asked to fix the phone Omp › Models role selects ending short. At 390px the
model picker (`.role-pick`) stopped at x=366 (desktop `max-width: 22rem`) and the thinking-level
select (`.role-level`) sat at its text width (14–179) on the line below. The phone media query in
web/mobile/src/components/agent-clis.css, scoped to `#cli-models-view .set-row-role`, sets the
picker `max-width: none` and the level `flex: 1 1 auto; min-width: 0; max-width: none` (a button
after it would share its line).
Verified: `make ci-scoped` PASS; all 15 roles, picker and level 14–376 at 390px and 14–346 at
360px; no overflow. visual-review: PASS at 390 and 360. Noted, not fixed: the two stacked
controls have no caption (told apart by value and chevron); the picker's chevron is faint vs the
native select's; control text 16px vs 14px role name. No scratch role had a button after the
level, so that shared-line case is unseen. Not done / debts: none. Merge: fast-forward ready.
