# ADR-0113: Matrix canvas mode: a layout mode per matrix, mode-dependent rectangle units, one transactional switch

- **Status**: accepted
- **Date**: 2026-09-10
- **Boundary**: persistence — `matrices` gains a `mode` column (migration 043), the meaning of `matrix_panels.x/y/w/h` becomes mode-dependent and is validated per mode, a seventh durable change-feed event (`matrix.mode`) appears, and switching modes rewrites every panel's rectangle in one transaction. Amends ADR-0108, which fixed those four columns to 12-column grid cells. No UI: the canvas surface is phase C2.

## Context

The Matrix canvas plan (`docs/plans/matrix-canvas.md`, every question in
§8 taken by the owner on 2026-09-10) gives a matrix a second layout mode:
the same panels, freed from the 12-column grid, placed anywhere on a plane
the viewer pans and zooms. The C0 spike
(`docs/benchmarks/2026-09-10-node-canvas.md`) is a GO on `@xyflow/react`
and found no blocker that forces two engines.

ADR-0108 stores one row per panel with `x, y, w, h` as grid cells, and the
store refuses anything outside `x + w ≤ 12`, `w ≥ 4`, `h ≥ 8`. **A canvas
cannot live inside that contract**: its coordinates are absolute, they run
negative in both axes, and they have no column cap — a panel dragged left
of the origin is legal on a plane and refused by the grid. Something has
to give, and only three things can: the columns (make them large enough to
fake a plane), the rows (a second table), or the meaning of the four
columns.

C0 also settled the unit. The grid's own margin is 8 px, `snapToGrid` with
an 8 px `snapGrid` places exactly (a drag lands on multiples of 8; with it
off the same drag lands at 78), and 8 px keeps every coordinate an
integer — so the layout PATCH, the events, the diff and the optimistic
wrapper stay exactly as they are.

The plan's rule for every mutation still holds: the answer carries the
payload its event carries, so the surface has one reducer path for a
response and a feed frame.

## Decision

A matrix has a **layout mode**: migration 043 adds `matrices.mode TEXT NOT
NULL DEFAULT 'grid'`, one column and no second table, and every existing
row reads `grid` — which is exactly what its rectangles already mean, so
there is no backfill and no rewrite. `mode ∈ {grid, canvas}` decides what
`x, y, w, h` mean and which rules the store enforces:

| Mode | Unit | Rules |
|---|---|---|
| `grid` | grid cell (12 columns, `rowHeight` 24 px) | `x ≥ 0`, `y ≥ 0`, `w ≥ 4`, `h ≥ 8`, `x + w ≤ 12` — ADR-0108's rules, unchanged |
| `canvas` | 8 px canvas unit | `w ≥ 32`, `h ≥ 28`, `x` and `y` may be negative, no column cap; `|x|, |y| ≤ 100000` and `w, h ≤ 4096` so a panel cannot be lost |

Every rectangle is judged by **the matrix's current mode**, read inside the
same transaction that writes it (the store has one SQLite connection, so
the read goes through the tx), so a canvas rectangle cannot be written into
a grid matrix or the reverse, and one bad row still refuses the whole
layout batch. `compact` stays and keeps meaning only in grid mode.

`SetMatrixMode(id, mode, ifUpdatedAt)` is the switch: one transaction that
changes the column, rewrites every panel by the documented transform, bumps
`updated_at` and appends **one** `matrix.mode` event carrying the summary
(with the new mode) plus exactly the panels it moved — the shape
`PATCH …/layout` answers with. `PATCH /api/matrices/{id}` accepts `mode`
beside `name` and `compact` under the same `ifUpdatedAt`, and answers that
same payload when the mode changes. Switching to the mode a matrix already
has writes nothing, announces nothing and leaves `updated_at` alone.

The transform, one direction each, repeated as pure functions in
`web/shared/domain/matrix.js` (`gridToCanvas`, `canvasToGrid`) so the UI can
preview a switch before asking for it:

- **grid → canvas**: a cell is 8 canvas units wide (`colWidth / 8`) and 3
  tall (`rowHeight` 24 px / 8) — `x' = x·8`, `w' = w·8`, `y' = y·3`,
  `h' = h·3` — then clamped to the canvas minimums and the plane bounds.
- **canvas → grid**: divide by the same factors, round half away from zero,
  clamp into `w ≥ 4`, `h ≥ 8`, `x ≥ 0`, `y ≥ 0`, `x + w ≤ 12`, then **pack**
  — reading order (top-down, then left-to-right, ties by panel id), each
  panel at the first free slot, the same arithmetic as `nextSlot`.

Decision table (every row has a store or handler test):

| Operation | Condition | Result |
|---|---|---|
| read | a database written before 043 | every matrix reads `grid`, no rewrite |
| patch mode | `grid → canvas` | 200 summary + every panel moved; one `matrix.mode`; rectangles multiplied by 8 and 3 |
| patch mode | `canvas → grid` | rectangles divided, rounded, clamped and packed; no two panels overlap; nothing lost |
| patch mode | same mode | no-op: no event, `updated_at` unchanged |
| patch mode | unknown mode | 400 "mode must be grid or canvas" |
| patch mode | stale `ifUpdatedAt` | 409, nothing written |
| add panel | canvas matrix, `w < 32` or `h < 28` | 400 naming the limit in canvas units |
| add panel | canvas matrix, negative `x`/`y` | accepted — the plane's, not a mistake |
| add panel | canvas matrix, `|x| > 100000` | 400 naming the bound |
| add panel | grid matrix, `x + w > 12` | 400 (ADR-0108's rule, unchanged) |
| add panel | grid matrix, a canvas-sized rectangle | 400 — judged by the matrix's mode, not the payload's shape |
| layout patch | mixed: one rectangle legal, one not | 400, nothing written (all or nothing, as today) |
| round trip | `grid → canvas → grid` | every panel inside the grid rules and no two overlap — **not** the layout it started with |

## Consequences

- The canvas is a value, not a schema: a 500-panel matrix switches in one
  transaction, and a rollback is one column. Grid mode is untouched — the
  same rules, the same messages, the same tests.
- **A round trip is lossy by design.** A grid panel at the 8-row minimum is
  24 canvas units tall, under the canvas minimum of 28, so the switch grows
  it — which can leave it overlapping the panel below by up to 4 units.
  Canvas mode is free placement and allows overlap; the switch back packs
  it out and the panel comes back 9 rows tall, not 8. Nothing is ever lost
  and nothing ever overlaps in grid mode; positions are not preserved to
  the cell. Overlap after a switch to the grid would be a bug, not a
  tolerance, and has its own test with panels that collide after rounding.
- The plane is bounded (`|x|, |y| ≤ 100000` units = ±800 000 px, `w, h ≤
  4096` units = 32 768 px). A panel cannot be dragged where no viewport can
  find it, and the bound is named in the refusal like every other limit.
- The messages are the contract twice over: `internal/store/matrix.go` and
  `web/shared/domain/matrix.js` carry the same words per mode, and the
  transform is implemented twice — once in Go for the write, once in JS for
  the preview — with the same fixtures on both sides. Two implementations
  can drift; the fixtures are what catches it.
- The client keeps one reducer path: `matrix.mode` is a summary plus a
  subset of rectangles, which is `matrix.updated` and `matrix.layout` in
  one frame, and `applyMatrixEvent` judges each rectangle in the mode its
  frame carries.
- Who breaks if this is wrong: a viewer whose browser is one release behind
  reads an unknown `mode` and falls back to `grid`, so it draws a canvas
  matrix as a grid — wrong, but not destructive, and any drag it saves is
  refused by the server's mode rule instead of corrupting the layout.
- Nothing here is UI. C2 builds the canvas surface on `GET
  /api/matrices/{id}`, the mode patch and `applyMatrixEvent`.

## Alternatives considered

- **A second table for canvas rectangles** (`matrix_canvas_panels`). Two
  sources of truth for one panel's position: every switch, every drag, every
  event and every read would have to keep both in step forever, and a panel
  present in one table and missing from the other is a state nothing can
  repair. Refused — a panel has exactly one position, and which units it is
  in is a property of the matrix.
- **Float coordinates** (store pixels as REAL). C0's 8 px unit keeps every
  coordinate an integer, so the layout PATCH, the events, `layoutDiff` and
  the optimistic wrapper are unchanged, and `snapGrid` falls out of the
  coordinate system for free. Floats would add rounding drift to every save
  and buy sub-8 px placement nobody asked for. Refused.
- **Storing the viewport server-side** (`camera_x, camera_y, zoom`). A
  camera is per viewer: two browsers on one matrix would yank each other,
  and a pan is not an edit. It follows the inspector width and the open-tabs
  list into `localStorage` (`picode-matrix-view:<id>`). Refused.
- **Widening the grid instead of adding a mode** (`cols` stored, e.g. 1000
  columns). One rule set, no transform — but a column is a relative unit
  that reflows to the container width, so "1000 columns" still has no fixed
  size, panels still cannot go negative, and every grid behaviour
  (compaction, collision resolution) would have to be disabled by a flag
  that is a mode in all but name. Refused.
- **A separate canvas app.** The owner's §8 row 2: a canvas is a mode of a
  matrix, one store, one API, one surface. Refused.
