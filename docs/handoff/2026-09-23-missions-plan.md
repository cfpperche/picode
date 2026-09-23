# 2026-09-23 — missions-plan: persistent work proposal

Prepared: `docs/plans/missions.md`, indexed from `docs/README.md`.
The proposal defines M0–M5, an M1–M3 MVP, ownership, context transfer,
evidence-bound acceptance and 22 planned lifecycle/recovery cases.
Review tightened atomic assignment reservations, busy targets, cancellation
races, session rollover and owner-reported manual acknowledgements.
Verified: `make close` scoped gates (fmt, vet, hooks, living docs) passed;
the generated handoff rendered with its existing size warning.
visual-review: n/a — planning documentation only; no UI changed.
Not implemented: Missions, its runtime contract, ADRs or D01–D22 tests.
The proposal does not accept architectural decisions or authorize rollout.
Integration: branch checks only; full main CI belongs to landing.
Follow-up: `docs/handoff/open/missions.md` tracks owner review and M0.
