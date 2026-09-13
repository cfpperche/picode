# 2026-09-10 — feat/matrix-canvas-model (Matrix v2 phase C1)

Base 547c4001 · head after `main` merge · store + server + client contract + docs. **No UI** — the canvas surface is C2.

## What landed

**ADR-0113** (amends ADR-0108) and **migration 043**: `matrices.mode TEXT NOT NULL DEFAULT 'grid'`. One column, not a second table; every row written before it reads `grid`, which is what its rectangles already meant, so there is no backfill.

**Store** (`internal/store/matrix.go`). `Matrix.Mode` on the summary; validation is mode-aware — grid keeps ADR-0108's rules verbatim, canvas is 8 px units with `w ≥ 32`, `h ≥ 28`, negative `x`/`y` allowed, `|x|, |y| ≤ 100000` and `w, h ≤ 4096`, each named in its refusal. Every rectangle is judged by the matrix's mode, read through the tx by `matrixGuard` (one SQLite connection). `SetMatrixMode` is the switch: one transaction, the transform (`x·8`, `w·8`, `y·3`, `h·3` clamped; back = divide, round half away from zero, clamp, then pack in reading order), `updated_at`, and one `matrix.mode` event carrying the summary plus exactly the panels it moved. Same mode = no write, no event, no bump. A 500-panel switch measured 79 ms (canvas → grid, the packing direction) and 4.8 ms (grid → canvas) in a throwaway timing test — the pack is not a cost worth optimising.

**Server**: `PATCH /api/matrices/{id}` takes `mode` under the same `ifUpdatedAt` and answers summary + moved panels; a `name`/`compact` in the same body is applied first, so a broken one refuses the patch before anything switches, and the switch then runs under the `updatedAt` the rename wrote (two events, both complete). `GET`s carry `mode`. OpenAPI is unchanged — no route was added.

**Client** (`web/shared/domain/matrix.js`): `MATRIX_MODES`, `UNIT_PX`, the canvas limits, `validateMode`, mode-aware `validatePlacement`/`validatePanel`/`nextSlot`/`layoutDiff` (all defaulting to `grid`, so v1 callers are unchanged), `gridToCanvas`/`canvasToGrid` with the server's rounding and pack, `PANEL_DEFAULT_CANVAS` 32×42, and `applyMatrixEvent` reducing `matrix.mode`.

Every row of the plan's decision table has a test: `TestMatrixModeReadsGridOnADatabaseWrittenBefore043`, `TestMatrixModeSwitchGridToCanvas` (with the same-mode, stale and unknown-mode rows), `TestMatrixModeSwitchCanvasToGridPacks`, `TestMatrixModeRoundTrip`, `TestMatrixCanvasPanelRules`, `TestMatrixModeStatuses`, `TestMatrixCanvasPanelStatuses`, and four JS tests in `matrix.test.js`.

## Debts

- The transform lives twice (Go for the write, JS for the preview). Only the shared fixtures catch drift — no cross-language check.
- `grid → canvas` clamps an 8-row panel from 24 to 28 units, so it can overlap the panel below by 4. The switch back packs it out; the round trip is legal and disjoint, never identical. Deliberate, in the ADR.
- Nothing exercises the mode from a browser yet, and no surface reads it: C2 (`docs/plans/matrix-canvas.md`) owns the React Flow host, the mode switch in the header and the QA.
