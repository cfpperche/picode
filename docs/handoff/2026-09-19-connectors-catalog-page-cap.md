# 2026-09-19 — feat/connectors-catalog-page-cap: publish the partial catalog when the registry outgrows the page budget

Shipped: The sync-fix logging did its job — after deploy the service log
showed "mcpcatalog: background refresh failed: registry pagination exceeded
120 pages": the registry (now past 12k servers) had outgrown the 120-page
paginator, so every background refresh aborted and discarded all fetched
pages and the pane stayed seed-only. Fix: hitting the page budget is now a
normal stop, not a failure — the partial catalog publishes and persists
(cap raised 120 → 200 pages).

Verified: mcpcatalog tests green — the new test drives a cursor chain
longer than the budget: the refresh succeeds, keeps exactly the
budget-worth of rows, stops at exactly the budget, and a restarted store
serves the persisted partial catalog. make ci-scoped PASS (2 paths).

## Debts
- After deploy: wait for the first sync (~2-3 min), then owner acceptance —
  /api/connectors/gallery?q=excel must return registry rows and
  ~/.picode/connectors-catalog.json must exist.
