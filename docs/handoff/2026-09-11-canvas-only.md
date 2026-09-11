# 2026-09-11 — Canvas replaces Matrix, session 1 (`feat/canvas-only`)

**Shipped.** ADR-0118's model half. Migration **045** rewrites every panel of a
`grid` board in canvas units once (`x·8`, `y·3`, `w·8`, `h·3`, clamped to
`w ≥ 32`, `h ≥ 28`), renames `matrices`/`matrix_panels`/`matrix_edges` with
their `canvas_id` column and both endpoint indexes, and drops `mode` — in the
one transaction the runner gives a migration. The store keeps one rectangle
rule, so `SetMatrixMode`, the grid validation, the `x + w ≤ 12` cap and the
switch transform go with the `matrix.mode` event. The rest is the rename, all
the way down: `/api/canvases…`, `canvas.*`, `internal/store/canvas.go`,
`internal/apps/canvas.go` (id, name and icon `canvas`),
`web/shared/domain/canvas.js` + `canvasGrants.js`, `docs/architecture/canvas.md`.

**Proved, not trusted.** `TestMigration045ConvertsGridPanelsOnce` rebuilds the
pre-045 shape on a real database and runs the shipped SQL: grid rectangles
convert (an 8-row panel lands 28 units tall, not 24), canvas ones do not move,
a second `migrate()` converts nothing again. `TestCanvasRenameKeepsKeysAndIndexes`
checks the foreign keys point at the renamed tables, both cascades still fire,
the unique binding index still refuses a duplicate and `canvas_edge_a`/`_b`
exist — `ALTER TABLE … RENAME TO` did carry them.

**Left for session 2.** The desktop surface: file names, its *matrix* copy, the
`#/app/matrix[/<id>]` redirect, the `canvas` icon in `AppIcon`,
`docs-site/guide/matrix.md`, the react-grid-layout / react-resizable
dependencies. This branch touched the desktop only where the build demanded it,
and deleted `MatrixGrid.jsx` with the mode switch: the domain no longer offers
the transform and the API no longer takes a mode. **Verified:** `ci-scoped` PASS.
