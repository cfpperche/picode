# 2026-09-23 — missions-delivery-boundary: define the integration seam

Changed: `docs/architecture/missions.md` names `assignment.delivery` as the Mission prompt receipt and reserves Delivery for the artifact and integration feature. It documents the one-way evidence link, separate review authority, and ADR-0186 queue boundary.
Changed: `docs/handoff/open/missions.md` links the Delivery flow and records the owner decision required before M5 integration work.
Verified: `make ci-scoped` and `make close` passed on the documentation branch; no runtime behavior or UI was changed or exercised.
visual-review: n/a (documentation only).
Not done: the five-mission pilot and any M5 design or implementation remain separate work in the Missions topic.
Merge: fast-forward ready at `f15ae2180` before this closing note.
