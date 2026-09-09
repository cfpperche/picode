# 2026-09-09 — feat/matrix-store: Matrix phase 2 — persistence, API, feed events

Shipped: ADR-0108 (boundary: persistence). Migration `041_matrix.sql` (040 is
`feat/unified-messages`'; the two are independent and land in either order):
`matrices` + `matrix_panels`, one row per panel, `UNIQUE(matrix_id, kind, ref)`,
cascade from the matrix, no FK to agents or terminals. `internal/store/matrix.go`:
Create/Get/List/Update/PatchLayout/AddPanel/RemovePanel/Delete, one transaction
each with its `matrix.*` event inside; six rows in `TestEveryMutationAppendsAnEvent`.
`internal/server/matrix.go`: the `/api/matrices` family (8 routes) behind the auth
gate; `pinStatus` → `storeStatus`; OpenAPI regenerated. `web/shared/domain/matrix.js`:
limits, normalizers, validators in the server's words, `applyMatrixEvent`. Docs:
`docs/architecture/matrix.md` (+ index row), fragment `docs/changelog.d/matrix-store.md`.

Verified: `make ci-scoped` PASS (fmt, vet, hooks, go[18], test-js, build, docs). Store
tests cover every decision-table row (limits, rules, duplicate, 409, all-or-nothing
layout, cascade by query, a panel surviving its agent's/terminal's deletion); handler
tests per route family (httptest, no t.Parallel); 8 node tests for `matrix.js`.

visual-review: n/a (no UI in this phase)

Not done / debts: plan §9 row 5 says a minimum panel of 3×6 cells, §4.2 (implemented)
says 4×8 — the owner's pick before phase 3; the API is exercised only by tests until
the surface lands; `nextSlot` / `layoutDiff` belong to phase 3's `matrix.js`.

Merge: fast-forward ready (main d2be9bd7 merged into the branch).
