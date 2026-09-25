# 2026-09-25 — feat/mobile-mission-band: phone New mission loses its empty top band
Shipped: owner asked to fix the empty band on the phone's New mission screen.
The form is a `.mission-section`, whose border-top + 18px padding-top separate
stacked sections on the desktop; first in `.m-missions-body` on the phone it
drew ~34px, a rule, then ~19px before "Workspace" (first field at y=105 under a
52px header). web/shared/styles/missions.css: phone-only
`.m-missions-body > .mission-section:first-child` drops both; first field now
at y=86, no rule. Desktop untouched (the class is phone-only).
Verified: `make ci-scoped` PASS; scratch 390×844 probe; New mission and the
missions list captured (no regression). visual-review: PASS
Noted, not fixed: "Include archived" on the missions list is a small tap target.
Not done / debts: none. Merge: fast-forward ready.
