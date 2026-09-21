# 2026-09-21 — delivery-observation: observe integration evidence

Shipped: D1b desktop/mobile Git Delivery views and common CLI/MCP observations,
with owner/root-scoped reads, branch ancestry, current checkout associations,
explicit unknown/stale coverage, and best-effort CI/land receipts (ADR-0170).
Verified: full scoped gates, focused Go race and JavaScript contract tests;
revision/dirty/deleted/divergent inputs, confinement, receipt failure, refresh
lifecycle and moved-root cases. Coverage: `docs/plans/delivery-observation.md`.
visual-review: PASS — independent screenshot review of desktop/mobile populated,
detail, empty, blocked, errors, partial and moved-root states; overlays passed.
Scratch used isolated Git/data fixtures and explicit browser fault injection;
no authenticated vendor, physical mobile or Windows-shell acceptance is claimed.
Not done: D2 publication observation or D3/D4 execution queues. Bounded history
and checkout expansion remain in `docs/handoff/open/delivery-flow.md`.
Merge: fast-forward ready after closure; main integration runs its own full CI.
No production deployment or restart was performed.
