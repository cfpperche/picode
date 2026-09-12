# ADR-0108: Matrix persistence: one row per panel, six feed events, a subset layout patch under ifUpdatedAt

- **Status**: accepted
- **Date**: 2026-09-09
- **Boundary**: persistence — two tables (`matrices`, `matrix_panels`; migration 041), six durable change-feed event types (`matrix.*`), the limits the store enforces, and a subset layout patch guarded by `ifUpdatedAt`. The protocol side of the Matrix app (the manifest's `surface` field) is phase 1's own ADR.

## Context

The Matrix plan (`docs/plans/matrix-app.md`, decisions taken by the owner
on 2026-09-09, §9) makes Matrix an app: a named 12-column grid of agent
and terminal panels — up to 64 matrices per machine and 500 panels each,
shared across browsers and announced on the change feed (§9 row 2: the
store, not `localStorage`; only "last matrix opened" stays per browser).
The phase-0 study (`docs/benchmarks/2026-09-09-matrix-live-grid.md`)
measured react-grid-layout 2.2.4 with 300 wrappers: a drag moves a few
panels; a compaction after removing a top panel can shift 200; the read
on open is the whole matrix. The write pattern is "a small subset changes,
often", with rare large subsets.

Precedents: pins (`docs/architecture/pins.md`) refuse limits with 400 and
the limit named, answer a stale `ifUpdatedAt` with 409, append their
events in the mutation's transaction and have one row per mutator in
`TestEveryMutationAppendsAnEvent` (ADR-0048). The pins v2 review found a
byte-sliced title that left invalid UTF-8 in the database: limits count
runes and refuse, never cut.

Two facts shape the schema. The same terminal twice on one matrix would be
two live attaches of one pane (§2.2's invariant), so one `(kind, ref)` per
matrix must be refused — by the database, not by a loop. And deleting an
agent or a terminal must not reach into matrices: the store is ignorant of
what a `ref` means, and the panel renders the target as gone (§4.4).

The plan named the migration `040_matrix.sql`; `feat/unified-messages`
already holds `040_peer_attention.sql`, and two files parsing to version
40 would make `migrate()` skip whichever lands second.

## Decision

Migration `041_matrix.sql` adds `matrices(id, name, compact DEFAULT
'vertical', created_at, updated_at)` and `matrix_panels(id, matrix_id →
matrices ON DELETE CASCADE, kind, ref, x, y, w, h, created_at,
UNIQUE(matrix_id, kind, ref))` — one row per panel, the panel id a random
slot (`panel-` + 48 bits) never derived from `ref`, no foreign key to
agents or terminals. `internal/store/matrix.go` is the only writer:
`CreateMatrix`, `UpdateMatrix` (name and/or compact, `ifUpdatedAt`),
`PatchMatrixLayout` (a subset of `{id, x, y, w, h}` under `ifUpdatedAt`,
every id on this matrix and every rectangle inside the grid or nothing is
written), `AddMatrixPanel`, `RemoveMatrixPanel`, `DeleteMatrix`. Each runs
in one transaction, bumps `updated_at` (RFC 3339 UTC, nanoseconds) and
appends its event inside that transaction: `matrix.created` and
`matrix.updated` carry the summary (`id, name, compact, createdAt,
updatedAt, panelCount`), `matrix.layout {id, updatedAt, panels: the
subset}`, `matrix.panel.added {id, updatedAt, panel}`,
`matrix.panel.removed {id, updatedAt, panelId}`, `matrix.deleted {id}`.
Limits refuse with the limit named: 64 matrices, 500 panels per matrix,
80 runes of name, `x ≥ 0`, `y ≥ 0`, `w ≥ 4`, `h ≥ 8`, `x + w ≤ 12`,
`kind ∈ {agent, terminal}`, `compact ∈ {vertical, none}`. A duplicate
`(kind, ref)` is refused by the unique index and read back as a conflict
("already on this matrix"); a stale `ifUpdatedAt` is a conflict with
nothing written; an add or a remove takes no precondition (a picker must
not fail because someone dragged). The handlers under `/api/matrices` map
`ErrInvalid` → 400, `ErrNotFound` → 404, `ErrConflict` → 409 and answer
every mutation with the payload its event carries, so the surface has one
reducer path for a response and a feed frame. Deleting an agent or a
terminal leaves matrices untouched.

Decision table (every row has a store or handler test):

| Operation | Condition | Result |
|---|---|---|
| create | 64 matrices exist | 400 "limit: 64 matrices" |
| create | name empty / > 80 runes / only spaces | 400 naming the limit; never truncated |
| create | ok | 201 summary; `matrix.created` |
| add panel | same (kind, ref) already on this matrix | 409 "already on this matrix" |
| add panel | 500 panels exist | 400 "limit: 500 panels per matrix" |
| add panel | w < 4, h < 8, x < 0, y < 0, x + w > 12, kind not agent/terminal, ref empty | 400 naming the rule |
| add panel | ok | 201 `{id, updatedAt, panel}`; `matrix.panel.added`; `updatedAt` bumped |
| layout patch | `ifUpdatedAt` stale | 409, nothing written |
| layout patch | a panel id not in this matrix, or a position out of bounds | 400, nothing written (all or nothing) |
| layout patch | ok subset | 200 `{id, updatedAt, panels}`; one `matrix.layout` event carrying exactly the subset |
| update | rename ok / compact ∈ {vertical, none} | 200 summary; `matrix.updated` |
| update | compact other / name over limit / stale `ifUpdatedAt` | 400 / 400 / 409 |
| remove panel | unknown panel | 404 |
| remove panel | ok | 204; `matrix.panel.removed` |
| delete matrix | ok | 204; panels gone (cascade proven by a query); `matrix.deleted` |
| any | unknown matrix id | 404 |
| agent or terminal deleted elsewhere | — | matrices and panels untouched (the panel row survives) |

## Consequences

- A drag in a 300-panel matrix writes and broadcasts the rows that moved
  — a few hundred bytes — instead of a 30 KB blob. The largest event is a
  compaction after removing a top panel: bounded by the cap (500 rows ≈
  50 KB) and rarer than a drag.
- Two browsers can edit one matrix. The writer holding a stale
  `ifUpdatedAt` gets a 409 and the surface refetches (plan §4.7); the
  feed carries the other writer's rows to everyone else.
- The store stays ignorant of the binding: a panel whose agent or
  terminal is gone stays until a person removes it, and the UI — not the
  schema — says so. The cost is a matrix that can list dead refs; the
  benefit is that deleting a terminal never fans out into N panel events
  nobody asked for.
- `cols` is not stored (12, fixed in v1); a v2 canvas adds the column
  then. The `compact` column already admits `none` so the v2 free canvas
  is a value, not a migration.
- Who breaks if this is wrong: a client that wants to save the whole
  layout can still send every row through the subset patch; a cap that
  proves too low is one constant and one test in each of the store, the
  handler and `web/shared/domain/matrix.js`.
- Nothing here is UI. Phase 3 builds the surface on `GET /api/matrices`,
  `GET /api/matrices/{id}` and `applyMatrixEvent`.

## Alternatives considered

- **One JSON blob per matrix** (`layout TEXT`). A drag rewrites and
  re-broadcasts the whole matrix; a duplicate binding is a loop in code,
  not a unique index; a subset patch becomes read-modify-write under a
  lock. Refused.
- **`localStorage` layouts.** Not shared across browsers, not backed up,
  not on the feed; the repo keeps only ephemeral chrome there (open tabs,
  rail width). Refused — "last matrix opened" is the one thing that stays
  in the chrome.
- **A store-side cascade when an agent or terminal is deleted.** The
  store would have to know what a `ref` means (agents and terminals live
  in different tables, and terminals have no foreign keys by design —
  ADR-0026), and a delete would announce panel removals nobody asked for.
  Refused: the panel stays and the UI shows it gone.
- **A whole-layout PUT instead of a subset PATCH.** A simpler client and
  30 KB per drag at 300 panels. Refused; the subset patch accepts every
  row when a client wants that.

## Amendment 2026-09-12 — a text panel keeps its words on the panel row

Owner, listing what a canvas should hold: "agents, terminais, pins, texto,
desenhos". The first four already exist. **Text** is different in kind from
all of them.

Every panel so far is a **reference**. An agent, a terminal, a pin, a file,
the changes to a file — each names something that lives elsewhere in PiCode
and would still be there if the canvas were deleted. The panel row says
where it sits and nothing more, which is why `ref` is `NOT NULL` and why
`UNIQUE(canvas_id, kind, ref)` means "this thing is on this canvas once".

A text panel names nothing. The words **are** the panel: written on the
plane, read there, and gone when it goes.

### What that changes

- **`content` is a column on `canvas_panels`** (migration 046), empty by
  default so every existing row reads back unchanged. Not a table of its
  own: a text block has no life outside its panel, nothing else in PiCode
  can hold a reference to one, and a second table would have added a key, a
  cascade and a join to say exactly that. It would also have suggested that
  a text block can be shared — which is what a **pin** already is, and what
  a pin panel already shows.
- **A text panel binds to its own id.** `ref` stays `NOT NULL` and the
  store mints it, so two empty text panels on one canvas do not collide the
  way two empty refs would. A caller that sends a ref is refused ("a text
  panel takes no ref"): it is describing something that does not exist.
- **`canvas.panel.content`** carries the whole string, not a patch. The
  store keeps what a reader typed and every other browser draws that, rather
  than replaying keystrokes; last writer wins, which is what a shared label
  on a plane should do.
- **2 000 characters** (`MaxCanvasText`), refused rather than truncated, the
  rule every other limit here follows. A label beside a terminal, not a
  document — a pin is what holds prose.
- **Saved on blur and after an 800 ms pause**, never per keystroke: each
  save is a transaction and an event to every open browser. The draft lives
  in the component until then, which is also what lets a panel be dragged,
  zoomed or unloaded mid-sentence without losing it.

### Consequences

- The canvas gains a kind that needs no fleet, no binding state that can go
  gone, and no read of its own: `text-ready` is unconditional and the words
  arrive with the panel row.
- A tool that places one skips the picker entirely — there is nothing to
  choose (`PICKS_A_TARGET`, `CanvasSurface.jsx`).
- **If wrong**: the column is additive with a default. Dropping the kind
  means one migration and deleting the rows; nothing else in PiCode points
  at them.
