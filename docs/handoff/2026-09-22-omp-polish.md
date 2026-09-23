# 2026-09-22 — feat/omp-polish: omp model rows polish (owner asked)
Shipped: picker chevron sits at its right edge like the selects; 32px between a role's help line and its controls, and every picker on one column (the DEFAULT row's long help had squeezed its picker narrower; column now fixed `14rem 11rem 7.5rem`, stacked below 860px); Settings rows without a help line no longer touch (6px gap). Files: `web/{browser,mobile}/src/components/agent-clis.css`.
Verified: measured on a qa-scratch (help→picker 32px, chevron 11px from the right edge, picker lefts all 953px, Interface row gap 6px, phone pickers all 320px, no overflow); overlayAudit rows aligned; `make ci-scoped` PASS.
visual-review: PASS (desktop 1600×1000 and phone 390×844, dark, read in subagents).
Merge: fast-forward ready after `make close`; the owner asked for deploy with force.
