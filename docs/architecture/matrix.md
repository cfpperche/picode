# Matrix (ADR-0108, ADR-0113, ADR-0116)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

A matrix is a named board of agent and terminal panels — the Matrix app
(plan: `docs/plans/matrix-app.md`; phase 0 study:
`docs/benchmarks/2026-09-09-matrix-live-grid.md`). Persistence, the API
family and the feed events are phase 2 (ADR-0108); the desktop surface —
react-grid-layout, chunk loading, the picker — is phase 3 and is the
**Surface** section at the end. A matrix has a **layout mode** (ADR-0113,
amending ADR-0108): the 12-column grid it shipped with, or a canvas plane
— the **Modes** section below. The canvas surface itself is C2 of
`docs/plans/matrix-canvas.md` and is the **Canvas** part of that section.

**Data model (migrations 041, 043 and 044).** `matrices(id, name, compact, mode,
created_at, updated_at)` and `matrix_panels(id, matrix_id → matrices ON DELETE
CASCADE, kind, ref, x, y, w, h, created_at, UNIQUE(matrix_id, kind,
ref))`. One row per panel, never one blob: a drag rewrites the rows that
moved and the unique index refuses a duplicate binding. A panel id is a
random slot (`panel-` + 12 hex), never derived from `ref`, so the same
terminal on two matrices is two panels. `kind ∈ {agent, terminal, note, file, diff}`
(the **Kinds** table below says what each `ref` is), with **no foreign key
on purpose** — the store is ignorant of the binding, deleting an agent, a
terminal, a pin or a file leaves the panel, and the UI renders the target as gone.
`compact ∈ {vertical,
none}` (default `vertical`) and keeps meaning only in grid mode; `mode ∈
{grid, canvas}` (migration 043, default `grid`, so every row written before
it reads the meaning its rectangles already had). `cols` is 12 in grid mode
and not stored. `updated_at` (RFC 3339 UTC, nanoseconds) bumps on every
mutation and is the optimistic-concurrency key. The store
(`internal/store/matrix.go`) is the only writer; every mutation is one
transaction with its event appended inside it (ADR-0048), one row per
mutator in `TestEveryMutationAppendsAnEvent`. A third table, `matrix_edges`
(migration 044), links two panels of one matrix — the **Edges** section
below, and ADR-0116 for what it grants.

**Kinds** (ADR-0108; the bodies past a live pane are C3 of
`docs/plans/matrix-canvas.md` §4.2). `kind` is an open text column, so a
kind is a validator edit on both sides — `internal/store/matrix.go` and
`web/shared/domain/matrix.js` — and never a migration. The store validates
the **shape** of a `ref` and never what it points at: whether that pin
exists is the UI's question, the same rule that lets a deleted terminal
leave its panel behind.

| Kind | `ref` | Body | Gone when |
|---|---|---|---|
| `terminal` | terminal id | the live pane | the terminal is not in the fleet |
| `agent` | agent id | the live TUI, or the row its mode asks for | the agent is not in the fleet |
| `note` | pin id | the pin's markdown, read-only, with **Open in Pin Studio** | the pin is not in `GET /api/pins` |
| `file` | `<owner>:<id>:<path>` | `FilePane` in its embedded layout | the **owner** is not in the fleet. A file missing on disk is *not* a binding state — the body reports the read failure and the panel keeps its actions |
| `diff` | `<owner>:<id>:<path>` | `WorkingDiff` — that path's working-tree patch, with its own **Open file** | the same rule: the owner. A path with no changes is the body's *No changes in this file.* |

The owner of a `file` or `diff` ref is one letter — `t` terminal, `a` agent, `w`
workspace — the desktop's own convention for the routes and tab ids that
already name a file (ADR-0030, `web/desktop/src/lib/routes.js`). A path may
hold colons of its own, so only the first two separate; the shape is refused
with **"ref must be `<owner>:<id>:<path>` with owner t, a or w"** on both
sides, and `parseRef` / `buildRef` (`web/shared/domain/matrix.js`) are the
only code that takes one apart or puts one together.

**Routes** (`internal/server/matrix.go`; the auth gate applies as for
every `/api/*`). Every mutation answers with the payload its event
carries, so the client has one reducer path for a response and a frame.

| Route | Answers |
|---|---|
| `GET /api/matrices` | `{matrices: [summary]}` — `id, name, compact, mode, createdAt, updatedAt, panelCount`, by name (ASCII case-folded), then creation |
| `POST /api/matrices {name}` | 201 summary; 400 with the limit named |
| `GET /api/matrices/{id}` | 200 summary + `panels: [{id, kind, ref, x, y, w, h, createdAt}]` + `edges: [{id, aPanel, bPanel, createdAt}]` (one read on open; 500 panels ≈ 50 KB, an edge is four short strings) |
| `PATCH /api/matrices/{id} {name?, compact?, mode?, ifUpdatedAt}` | 200 summary; 400 on an empty patch or a broken rule; 409 when the row moved on |
| …the same route when `mode` changes it | 200 summary **plus** `panels: [{id, x, y, w, h}]` — every panel the switch moved, the shape the layout patch answers with, so the client has one reducer path. Same mode: the untouched summary, `panels: []`, no event, `updatedAt` unchanged. Unknown mode: 400 "mode must be grid or canvas". A `name`/`compact` in the same body is applied first (its own `matrix.updated`), so a broken one refuses the patch before the switch |
| `PATCH /api/matrices/{id}/layout {ifUpdatedAt, panels: [{id, x, y, w, h}]}` | the changed subset, one transaction, all or nothing, every rectangle judged by the matrix's current mode; 200 `{id, updatedAt, panels}`; 409 when stale |
| `POST /api/matrices/{id}/panels {kind, ref, x, y, w, h}` | the client places (`nextSlot`, phase 3), the server validates against the matrix's current mode; 201 `{id, updatedAt, panel}`; 409 on a duplicate binding |
| `DELETE /api/matrices/{id}/panels/{panelId}` | 204; the feed carries the new `updatedAt`, and the panel's edges go with it |
| `GET /api/matrices/{id}/edges` | `{edges: [{id, aPanel, bPanel, createdAt}]}`, oldest first — the audit list, which wants the edges without the panels |
| `POST /api/matrices/{id}/edges {aPanel, bPanel}` | 201 `{id, updatedAt, edge}`; the store orders the pair, so either direction is the same row; 400 names the rule (another matrix's panel, the same panel twice, a kind with no mailbox, the cap); 409 "These panels are already linked" |
| `DELETE /api/matrices/{id}/edges/{edgeId}` | 204, and the grant goes with the row; 404 when it is already gone |
| `DELETE /api/matrices/{id}` | 204; the panels and their edges cascade |

`ifUpdatedAt` may also travel as an `If-Match` header (the pins
convention); empty means no precondition. Adding or removing a panel takes
none — a picker must not fail because someone dragged.

**Limits refuse and never truncate** (400, the limit named; runes, not
bytes): 64 matrices, 500 panels per matrix, 80 characters of name (empty
or only spaces is "name is required"). A rectangle is judged by **the
matrix's mode**, read inside the same transaction that writes it — so a
canvas rectangle cannot be written into a grid matrix or the reverse:

| Mode | Unit | Rules |
|---|---|---|
| `grid` | grid cell (12 columns, `rowHeight` 24 px) | `x ≥ 0`, `y ≥ 0`, `w ≥ 4` columns, `h ≥ 8` rows, `x + w ≤ 12` |
| `canvas` | 8 px canvas unit | `w ≥ 32` units (256 px), `h ≥ 28` units (224 px), `x` and `y` may be negative, no column cap |

The canvas plane is bounded so a panel cannot be dragged out of reach of
every viewport: `|x| ≤ 100000` and `|y| ≤ 100000` units (±800 000 px),
`w ≤ 4096` and `h ≤ 4096` units (32 768 px). The messages are the
contract: `web/shared/domain/matrix.js` repeats them per mode
(`validateName`, `validateCompact`, `validateMode`, `validatePlacement`,
`validatePanel`) so the UI can refuse before asking.

**Modes** (ADR-0113). `mode` decides what `x, y, w, h` mean — grid cells or
8 px canvas units — and `compact` keeps meaning only in grid mode.
Switching is `SetMatrixMode`: one transaction that changes the column,
rewrites every panel by the transform below, bumps `updated_at` and
appends one `matrix.mode` event carrying the summary plus exactly the
panels it moved. Switching to the mode a matrix already has writes and
announces nothing. A grid cell is **8 canvas units wide** (`colWidth / 8`)
and **3 tall** (`rowHeight` 24 px / 8), so `grid → canvas` is `x·8`, `w·8`,
`y·3`, `h·3` clamped to the canvas minimums, and `canvas → grid` divides by
the same factors, rounds half away from zero, clamps into the 12-column
rules and **packs** — reading order, first free slot (`nextSlot`'s
arithmetic), because rounding alone leaves panels overlapping. The trip is
not identity: an 8-row panel is 24 units tall, grows to the 28-unit
minimum (and may overlap the panel below, which a canvas allows), and comes
back 9 rows tall. Nothing is ever lost, and two panels never overlap in
grid mode. `web/shared/domain/matrix.js` repeats the transform as
`gridToCanvas` / `canvasToGrid` so the UI can preview a switch before
asking for it. **The viewport is per viewer and is not stored** — a camera
follows the inspector width into `localStorage`, not the store.

**Events** (durable, appended in the mutation's transaction; `touches(ev,
["matrix"])` keys on the prefix, so `matrix.panel.*` reaches the surface
with the rest):

| Event | Data |
|---|---|
| `matrix.created`, `matrix.updated` | the summary |
| `matrix.mode` | the summary (with the new mode) + `panels: [the subset the switch moved]` |
| `matrix.layout` | `{id, updatedAt, panels: [the subset that moved]}` |
| `matrix.panel.added` | `{id, updatedAt, panel}` |
| `matrix.panel.removed` | `{id, updatedAt, panelId}` — the panel's edges go with it, without an event each |
| `matrix.edge.added` | `{id, updatedAt, edge}` |
| `matrix.edge.removed` | `{id, updatedAt, edgeId}` |
| `matrix.deleted` | `{id}` — the panels and edges go with it, without an event each |

**Client contract** (`web/shared/domain/matrix.js`, pure): `MATRIX_LIMITS`,
`MATRIX_KINDS`, `PANE_STATES` / `hasPane`, `MATRIX_COMPACT`, `MATRIX_MODES`, `MATRIX_EVENTS`,
`UNIT_PX` (8), `EDGE_KINDS`; `normalizeMatrix`,
`normalizeMatrixList`, `normalizePanel`, `normalizeEdge`,
`normalizeEdgeList`, `normalizeMatrixDetail` drop junk
the way `contracts/appPrimitives.js` does (a panel without a whole
rectangle or a known binding is dropped); `validateEdge(edge, panels,
edges)` and `edgeEndpoints(edge, panels)` are the **Edges** section below;
`applyMatrixEvent(state, ev)`
reduces the nine events over `{ list: [summaries], byId: { id: { matrix,
panels, edges } } }` — an event for a matrix that is not loaded only touches the
summary list, a `created`/`updated` for a matrix the list never saw
inserts it (the summary is complete), the list stays sorted by name, and
junk or an unknown type returns the same object. Every rectangle is judged
in the mode it belongs to (`normalizeMatrix` falls back to `grid` for an
unknown one), and `validatePlacement`, `validatePanel`, `nextSlot` and
`layoutDiff` take the mode with `grid` as the default, so every v1 caller
is unchanged.

Decision table (every row has a store or handler test —
`internal/store/matrix_test.go`, `internal/server/matrix_test.go`):

| Operation | Condition | Result |
|---|---|---|
| create | 64 matrices exist | 400 "limit: 64 matrices" |
| create | name empty / > 80 runes / only spaces | 400 naming the limit; never truncated |
| create | ok | 201 summary; `matrix.created` |
| add panel | same (kind, ref) already on this matrix | 409 "already on this matrix" — per kind, so one pin can be a `note` on two matrices and one id may be two kinds |
| add panel | `note` whose pin does not exist, `file`/`diff` whose owner or file does not exist | 201 — the store never asks; `bindingState` and the body are what say so |
| add panel | 500 panels exist | 400 "limit: 500 panels per matrix" |
| add panel | w < 4, h < 8, x < 0, y < 0, x + w > 12, kind not agent/terminal/note/file/diff ("kind must be agent, terminal, note, file or diff"), ref empty | 400 naming the rule |
| add panel | a `file` or `diff` ref that is not `<owner>:<id>:<path>`, or whose owner letter is not t/a/w | 400 naming the shape; a path with colons of its own is fine |
| add panel | the same path as a `file` and as a `diff` | 201 twice — the unique index is per (kind, ref), and reading a file and reading its changes are two things to have open at once |
| add panel | ok | 201 `{id, updatedAt, panel}`; `matrix.panel.added`; `updatedAt` bumped |
| layout patch | `ifUpdatedAt` stale | 409, nothing written |
| layout patch | a panel id not in this matrix, or a position out of bounds | 400, nothing written (all or nothing) |
| layout patch | ok subset | 200 `{id, updatedAt, panels}`; one `matrix.layout` event carrying exactly the subset |
| update | rename ok / compact ∈ {vertical, none} | 200 summary; `matrix.updated` |
| update | compact other / name over limit / stale `ifUpdatedAt` | 400 / 400 / 409 |
| read | a database written before 043 | every matrix reads `grid`, no rewrite |
| patch mode | `grid → canvas` | 200 summary + every panel moved; one `matrix.mode`; rectangles multiplied by 8 and 3 |
| patch mode | `canvas → grid` | divided, rounded, clamped and packed; no two panels overlap; nothing lost |
| patch mode | same mode | no-op: no event, `updatedAt` unchanged |
| patch mode | unknown mode / stale `ifUpdatedAt` | 400 naming the two / 409, nothing written |
| add panel | canvas matrix: `w < 32`, `h < 28`, `|x| > 100000` | 400 naming the limit in canvas units |
| add panel | canvas matrix, negative `x`/`y` | accepted — the plane's, not a mistake |
| add panel | a rectangle sized for the other mode | 400 — judged by the matrix's mode, not the payload's shape |
| layout patch | canvas matrix, mixed: one rectangle legal, one not | 400, nothing written (all or nothing) |
| round trip | `grid → canvas → grid` | inside the grid rules and disjoint, never identical (the 8-row minimum returns 9 rows tall) |
| remove panel | unknown panel | 404 |
| remove panel | ok | 204; `matrix.panel.removed` |
| delete matrix | ok | 204; panels gone (cascade proven by a query); `matrix.deleted` |
| any | unknown matrix id | 404 |
| agent or terminal deleted elsewhere | — | matrices and panels untouched (the panel row survives) |

## Edges (ADR-0116)

An edge is the owner's recorded intent that two sessions may exchange
messages. **It grants exactly ADR-0104's mailbox contact, and nothing
else**: an edge never lets one session read another's session file,
scrollback, buffer or history — **no transcript, ever**. The supported way
to get context out of another session is to ask it, and let it answer under
its own judgment.

**Table** (migration 044):

```
matrix_edges(id, matrix_id → matrices ON DELETE CASCADE,
             a_panel → matrix_panels ON DELETE CASCADE,
             b_panel → matrix_panels ON DELETE CASCADE,
             created_at, UNIQUE(matrix_id, a_panel, b_panel))
```

The pair is stored **ordered** (`a_panel < b_panel`, sorted before the
write), so the same edge drawn in either direction collides on the unique
index: an undirected edge is one row, and there is no direction column —
an arrow would promise a one-way restriction the mailbox cannot keep.
Endpoints are **panels**, not sessions: a panel already carries its own
`(kind, ref)`, and re-pointing a panel is a new binding that must be drawn
again. Both foreign keys cascade, so an edge cannot outlive the matrix or
either panel — the grant follows the line you can see. The unique index
leads with `matrix_id` (the per-matrix read and the cascade from
`matrices`); each endpoint has its own index, because the cascade from
`matrix_panels` and the contact union both look a panel up as an *endpoint*.
Cap: **1000 edges per matrix**, refused with the limit named.

**Who may draw one.** Only the owner, in the browser, through the
authenticated owner API. **The MCP surface gains no verb** — no tool in
`internal/communication` creates, lists or implies an edge, so an agent can
neither draw one nor discover that one could exist
(`TestMCPSurfaceGainsNoEdgeVerb`). A capability whose beneficiary can grant
it to itself is not a capability.

**The contact union.** `PeerContacts` is the union of two sources, deduped,
each row still filtered by `peerCurrent` (ADR-0104 invalidates a connection
whose recorded session moved, and this does not soften it):

1. the caller's own **workspace**, exactly as ADR-0104 scoped it, and
2. the connections a **live edge** links to the caller — resolved through
   the far panel's `(kind, ref)`: `agent` matches `peer_connections.agent_id`,
   `terminal` matches `terminal_id`.

The grant is **derived on every contact read** and is never cached: nothing
about an edge is written into `peer_connections`, no column, no flag. The
same union decides `SendPeerMessage`, so a contact the caller cannot write
to is impossible — a grant that is only drawn would be a lie in the UI.

| Situation | Contact? |
|---|---|
| same workspace, no edge | yes, both ways — unchanged |
| different workspaces, no edge | no |
| different workspaces, live edge, both enrolled and current | yes, each side sees the other **exactly once** |
| the edge is removed | no |
| one connection revoked | no |
| one session changed (`peerCurrent` false) | no |
| one panel removed | no (the edge cascaded away with it) |
| the matrix is deleted | no; the edges are gone |
| two edges in two matrices for the same pair | yes, listed once |
| the panel's target was deleted from the fleet | no — the panel survives (ADR-0108), the connection cascades with its agent or terminal |
| an edge between two panels bound to my own session | no — the caller is never in its own list |

Dropping edges is dropping one table and one union clause: the mailbox
returns to workspace scope with no migration of anything else, because
nothing was ever copied while an edge existed.

**Client contract** (`web/shared/domain/matrix.js`): `normalizeEdge` /
`normalizeEdgeList` (an id and two different panel ids, junk dropped);
`validateEdge(edge, panels, edges)` repeats the server's refusals in the
server's order and words — the two ids, two different panels, both on this
matrix, both a kind with a mailbox (`EDGE_KINDS`), then the cap and "These
panels are already linked" in either direction; `edgeEndpoints(edge,
panels)` answers the two panel rows the canvas draws between, or `null`
when an end is not on the board — the one case it must not draw.
`applyMatrixEvent` reduces `matrix.edge.added` / `matrix.edge.removed`, and
`matrix.panel.removed` drops the edges that touched the panel, mirroring the
store's cascade (which announces no event per edge).

**Drawing one** (`MatrixCanvas.jsx`, `MatrixLink.jsx`, `MatrixSurface.jsx`).
React Flow's `Handle` plus `onConnect`: a connector knob in the panel
header, in the action row that already carries `nodrag`, so a drag from it
starts a connection and never moves the panel; the body keeps `nodrag
nowheel` and `.mx-head` stays the drag handle. It is hidden until the panel
is hovered, focused or selected — 200 panels must not wear 200 knobs — and
**only `agent` and `terminal` panels have one**: a note has no mailbox, the
store refuses that edge, and a UI that offers what the store refuses teaches
a lie. `connectionMode` is **loose**, React Flow's word for "any handle may
meet any handle", which is the honest model for an undirected edge: one
connector per panel, and `source`/`target` in `onConnect` are only the
order the pointer travelled — the store sorts the pair. Two refusals are
live during the drag (a panel to itself, a pair already linked) so the drop
reads invalid instead of arriving as a 409; every other refusal is
`validateEdge`'s, in the server's words, before the request.

**The two questions** (`matrixGrants.js`; ADR-0116 §5, *an edge never enrols
silently*). The connection list is read **fresh** on every draw — enrolment
is exactly the thing that may have changed a second ago — and nothing is
written until the owner has answered:

1. **Across two folders**, first and on its own, naming both: *"Atlas works
   in ~/a and Delta in ~/b. Linking them lets these two sessions message
   each other across folders — nothing else is shared, and neither can read
   the other's history."* It is asked separately because pairing across
   workspaces is the only thing an edge can do that the workspace rule could
   not (§3).
2. **An end that is not connected**, offering the existing owner enrolment
   (`POST /api/communication`, `automatic` when the CLI supports launch —
   the same call the Messages view makes): *"Cleo is not connected yet —
   connecting it and drawing this link lets Atlas and Cleo message each
   other. Neither can read the other's history."*

Cancel at either leaves no edge **and no connection**. Two ends the store
would refuse to enrol are said plainly instead of offered a failing action:
a session that is gone, and one with no recorded conversation (`EnablePeer`
needs a session, a workspace and a CLI). Both dialogs are `askConfirm` —
the app's `ResponsiveDialog` primitive (ADR-0046) — one line and one action
at `--ctl-h`. Removing asks the same way, and **only while the link still
grants**: taking away a link that grants nothing takes nothing away.

**Reading broken** (§7). `matrixGrants.js` joins a matrix's edges and panels
with `GET /api/communication` — whose `connections[].active` *is* the
store's `peerCurrent` — and answers `grants`, the reason, and whether the
pair crosses folders. It is the only place that decides, so the plane and
the audit list can never disagree. The four reasons are kept apart because
each has a different next action: `gone` (the session left the fleet),
`off` (never enrolled), `revoked` (the owner revoked it), `stale` (the
recorded session moved). A link that grants nothing is drawn in the warn
colour with an unlink mark, the word **Broken**, and the one line naming
which end and why — never a dotted line, which is a thing a viewer learns
to ignore. The connection list is read only when the matrix holds an edge,
refreshed on `peer.*` **and** on `agent.updated` / `terminal.updated` /
`*.deleted` (a grant can stop granting because the *owner* changed session,
which no `peer.*` row announces), debounced 400 ms — the feed, never a
timer. A failed read keeps the last answer: flipping every live link to
broken is the one direction a wrong answer is dangerous in.

**Where the chip sits**, and why it is the way it is (the browser pass,
below, is the evidence). React Flow draws edge labels *under* the nodes and
the midpoint of a line between two headers lands on a panel more often than
not, so the chip is lifted above the nodes and pushed off the line by half
its own size plus ten screen pixels along the line's normal — forced to one
side, because the arithmetic's side follows the stored pair order, which is
the panel ids sorted and nothing a viewer can see. A link that **grants**
shows its chip only while it is asked for (hover or selection, with a grace
period so the pointer can cross the gap to it); a link that **grants
nothing** shows it always. Hovering the chip — not only selecting the line —
is what reveals the reason, because two panels close together hide their
whole link behind themselves and then the line has no hittable pixel.

**Edges in grid mode.** An edge belongs to the matrix, not to a mode, but a
grid has no plane to draw one on. The header wears a **link chip with the
count** instead (`linkCounts` / `linkChipTitle`), amber when any of them
grants nothing, and it leads to the audit list. The alternative — drawing
edges over the grid — was refused: react-grid-layout reflows panels on
every compaction, so a line would chase rows that move for reasons that
have nothing to do with the link, and the count is the whole of what a
non-spatial layout can honestly say.

**The audit list** (`web/desktop/src/components/MatrixLinks.jsx`), the
non-spatial place ADR-0116's Consequences ask for by name. A section in the
Messages view (`#/clis/messages`) listing **every** live edge: both ends by
name and kind, the matrix it lives on, whether it grants right now, and a
Remove that calls the same `DELETE` the canvas does. It reuses the
participants list's row shape — "who may reach whom" should read the same
whether the pairing came from a workspace or from a line someone drew — and
it is deliberately **not** scoped to the workspace picker above it, because
an edge is per pair and half of what it is for is pairing two folders.
Reads: the matrix list plus one detail per matrix (which already carries
`edges` and `panels`) and the connection list, refreshed by the same feed
rows, debounced.

Decision table (every row has a store test in
`internal/store/matrix_edges_test.go` and, where it is an HTTP answer, a
handler test in `internal/server/matrix_test.go`):

| Operation | Condition | Result |
|---|---|---|
| add edge | two `agent`/`terminal` panels of this matrix | 201 `{id, updatedAt, edge}`; `matrix.edge.added`; the pair stored ordered |
| add edge | the same pair drawn backwards | 409 "These panels are already linked"; still one row, one event |
| add edge | `aPanel == bPanel` | 400 "an edge needs two different panels" |
| add edge | either id empty | 400 "aPanel and bPanel are required" |
| add edge | a panel of another matrix, or one that does not exist | 400 "panel ⟨id⟩ is not on this matrix" |
| add edge | a `note`, `file` or `diff` panel | 400 "panel ⟨id⟩ is a ⟨kind⟩ panel and has no mailbox: an edge links agent or terminal panels" |
| add edge | 1000 edges exist | 400 "limit: 1000 edges per matrix" |
| add edge | unknown matrix | 404 |
| remove edge | ok | 204; `matrix.edge.removed`; the grant goes with the row |
| remove edge | already gone | 404 "edge not found" |
| remove panel | the panel had edges | they cascade; the feed carries only `matrix.panel.removed` |
| delete matrix | — | the edges go with it, without an event each |
| MCP | any tool | no verb creates, lists or implies an edge |

## Surface (phase 3)

The app (`internal/apps/matrix.go`: id `matrix`, icon `matrix`, `surface:
"native"`, no badge — ADR-0109) is registered by the desktop as `matrix`
in `web/desktop/src/lib/nativeApps.js` and mounts
`web/desktop/src/components/matrix/MatrixSurface.jsx` with `host` plus
`initialPath` / `onPathChange`, so `#/app/matrix/<matrixId>` opens that
matrix and switching updates the hash. The phone lists the tile as
*Desktop only*.

| File | Holds |
|---|---|
| `MatrixSurface.jsx` | header (native `<select>` switcher, **Add panel**, **New matrix**, a menu with Rename / Delete), the scroll container chunk loading observes, the empty states, the store `{ list, byId }` reduced by `applyMatrixEvent`, the save flow, "last matrix opened" in `localStorage` `picode-matrix-last` |
| `MatrixCanvas.jsx` | canvas mode's host: `@xyflow/react` 12.11.6, lazy-imported, one custom node type rendering the same `Panel`; the camera, the zoom rule and the minimap. **Canvas** below |
| `PanelStill.jsx`, `stills.js` | the two bodies a canvas panel has instead of a live pane — the text still and the name-plate — and the memory-only map of captured screens |
| `MatrixGrid.jsx` | react-grid-layout 2.2.4, v2 API: `useContainerWidth` on the canvas, `gridConfig {cols 12, rowHeight 24, margin 8}`, drag by `.mx-head` with `.mx-actions` as the cancel zone, `se`/`s`/`e` handles, `fastVerticalCompactor` from `react-grid-layout/extras`, children keyed by panel id, `minW 4 / minH 8` |
| `Panel.jsx` | the wrapper every panel keeps (and `PanelHead`, which the maximize layer reuses): face, name, hint, the sidebar's chip (`agentRowStatus` / `terminalStatus`), Open · Maximize · Remove from matrix; it observes its own visibility; two memo layers so a grid re-render never touches a body |
| `PanelBody.jsx`, `TerminalPanel.jsx`, `NotePanel.jsx`, `FilePanel.jsx`, `DiffPanel.jsx` | `PanelBody` routes by **kind**: a terminal or an agent is `TermSurface` (the tab's engine; a terminal first `POST /api/terminals/{id}/open`s for its live record), a note is `NotePanel` — one `GET /api/pins/{id}` per mount, `react-markdown` + `remarkGfm` over the app's `.md` styles, refetched on that pin's `pin.updated` (it subscribes to the feed itself, as the Inspector does for `git.updated`) — a file is `FilePanel`, which is `FilePane` with `variant="embedded"` plus the two things the matrix adds (`docKey`, `onDirty`) — a diff is `DiffPanel`, `WorkingDiff` under a nonce the feed bumps. Unloaded → one muted line with the feed's last state; the rows below as one line + one action |
| `PanelFace.jsx` | the mark that says what a panel is bound to, in the header and on the name-plate: a provider face, a CLI badge, or the pin mark |
| `PanelPicker.jsx`, `NameDialog.jsx` | cmdk list **grouped by kind** — Agents, Terminals, Pins, Open files, Changes to an open file — of what is not yet on this matrix, with the sidebar's faces and words; a group with nothing in it says so under the list with its one action (*No pins yet.* — New pin); the name form (`matrixNameSchema`, Zod, the store's messages, `noValidate`) |
| `chunkLoader.js`, `paneOwnership.js` | the `IntersectionObserver` glue over the pure `loadPolicy`; what unloading does to an attach |
| `web/desktop/src/lib/fileDocs.js` | the open documents of `file` panels, keyed by ref and held outside React — `paneOwnership.js` for an editor, so maximizing a panel or switching layout mode keeps unsaved text |
| `web/shared/domain/matrix.js` | `nextSlot`, `layoutDiff`, `parseRef` / `buildRef` / `validateRef`, `refOwner`, `gitTouches`, `bindingState`, `hasPane`, `loadPolicy` (with the zoom and the per-row `pane`), `zoomBody`, `pointerAtZoom`, `unitsToPx` / `pxToUnits`, `tidyCanvas`, `normalizeViewport`, `suspendedToDispose`, `panelOrder`, `neighborPanel` (+ `PANEL_DEFAULT` 4×14, `PANEL_DEFAULT_CANVAS` 32×42, `CANVAS_ZOOM`, `PANEL_DIRECTIONS`) — one test per row below in `matrix.test.js` |

**The fleet decides what a panel is bound to**, and a note's pin follows the
same rule from the surface's own list: `pins` is `null` until
`GET /api/pins` answers, and `bindingState` reads a missing list as "not
read yet", never as gone. That list is read **only when something needs
it** — the matrix holds a note, or the picker is open — then kept current by
`pin.*` on the feed and a stale reveal; a matrix of terminals costs no pin
request at all. It carries summaries, never bodies (`docs/architecture/pins.md`),
which is why the body reads its own pin. The fleet's own rule: the surface
draws the skeleton until the desktop says it has one: `host.fleet.loaded` is the
App's `bootstrapped`, and until it turns true an empty fleet means "not
read yet", not "deleted". Without it every terminal panel showed *That
terminal is gone.* for the length of the boot fetch — 15 s on a fixture
with 45 terminals (each row carries git state). A host that passes no
`loaded` is taken at its word.

**Reads.** `GET /api/matrices` and `GET /api/matrices/{id}` once on open
and on a reveal older than 10 s; then the feed only: `matrix.*` through
`applyMatrixEvent`, agents and terminals through the host's fleet,
`agent.tui` for the Working chip after one `GET /api/tui-working` for the
agents on the matrix, `GET /api/pins` when a note or the picker needs it and
again on `pin.*`, and one `gitdiff` read per `diff` panel, again on a
`git.updated` for its owner's folder. No timer.

**Binding × fleet** (`bindingState`, plan §4.4):

| Row | Body when loaded | Actions |
|---|---|---|
| `terminal-running` (also a plain shell whose tmux died: opening revives it) | live xterm | Open · Maximize · Remove |
| `terminal-stopped` (a CLI terminal, `launchCli` + `running: false`) | `TermSurface`'s own stopped state — Resume last session / Start from Agent CLIs | same |
| `terminal-gone` | "That terminal is gone." | Remove |
| `agent-interactive` | live TUI | Open · Maximize · Remove |
| `agent-managed` | "Managed agent — open to read." (phase 4 brings the conversation) | Open · Remove |
| `agent-stopped` | "Agent is stopped." — **Run** (`POST /api/agents/{id}/open`; the feed flips the mode and the body swaps in place) | Open · Remove |
| `agent-gone` | "That agent is gone." | Remove |
| `note-ready` | the pin's markdown, read-only | Open in Pin Studio · Maximize · Remove |
| `note-gone` | "That pin is gone." | Remove |
| `file-ready` | `FilePane`, embedded: the editor, its Preview/Raw toggle and Save. The chip says **Unsaved** while it is dirty | Open in its own tab · Maximize · Remove |
| `file-gone` | "Where this file was read from is gone." | Remove |
| `diff-ready` | `WorkingDiff` for that path, refetched on `git.updated` for the owner's folder (`gitTouches`) — its own header has **Open file** | View this diff in a tab · Maximize · Remove |
| `diff-gone` | "Where this file was read from is gone." | Remove |

**Chunk loading** (`loadPolicy`, plan §4.5). Observer root: the surface's
scroll container in grid mode (`rootMargin: 100% 0px` — it only scrolls
vertically) and the React Flow pane in canvas mode (`100%`, both axes: a
plane pans sideways too, and C0 measured 49 panels in the band against 28).
A wrapper that unmounts stays loaded for one tick, because switching layout
mode unmounts every panel in one host and mounts it in the other inside the
same commit — dropping the flag at once would make the switch a socket
suspend and kick per panel.

| Situation | Behaviour |
|---|---|
| wrapper near the viewport for 300 ms | body mounts |
| leaves before 300 ms | the pending load is dropped |
| loaded, then leaves | body unmounts after 5 s; coming back sooner cancels |
| dragged, resized, focused or maximized | pinned: never unloads |
| matrix tab hidden | every panel reads as far and unpinned: all unload after 5 s, the focused one too |

**Unsaved work is never unmounted silently.** A `file` body reports its
dirty bit up (`FilePane`'s `onDirty`) and the surface `keep`s that panel in
the loader: `keep` is the `pinned` set a drag and the focus already use, with
one difference — it survives a hidden matrix, because a hidden tab throwing
away a draft is exactly the silent unmount the rule exists to stop. The chip
says **Unsaved** while it holds. The text itself outlives a remount either
way: the document lives in `lib/fileDocs.js` keyed by the panel's ref, so
maximizing a panel and switching layout mode move the body between hosts
without losing it, the same way `paneOwnership.js` keeps one xterm across
hosts. Removing a dirty panel asks first (Undo restores the panel, never the
text) and is the one thing that forgets the document.

**Where a file or a diff panel comes from.** The picker must not become a
file manager, so it offers **only what is already open as a file tab** — a
path this browser already has, read from `host.openTabs` and turned into a
ref by `buildRef` — once as *Open files* and once as *Changes to an open
file*, because the same path as a `file` and as a `diff` is two panels. Everything else stays a debt on purpose: an *Add to matrix* row
on an Inspector change or an entry in a file tab's own menu would need the
desktop to know which matrix is open and where a panel would land, and that
state lives inside the Matrix app's surface — apps read the desktop through
`host`, never the other way round (ADR-0109). Adding it means giving the
desktop a matrix client of its own, which is a decision, not a line.

**Ownership** (`paneOwnership.js`, plan §4.6):

| Situation | Unload | Load |
|---|---|---|
| the terminal's tab (`t:<id>`) or the agent's tab is in `host.openTabs` | park the pane only; the tab keeps the attach | `TermSurface` re-claims the pane |
| only on matrices | `suspendTermSocket` — socket closed, xterm and scrollback kept | `ShellTerm` kicks the suspended socket |
| more than 24 suspended, or one for 10 min | `closeShellTerm`, only while still suspended (a kicked entry belongs to someone again) | a fresh xterm; tmux holds the screen |
| its tab closes while the panel shows it | — | the body remounts with a fresh xterm (the tab disposed the old one) |
| the body moves host (maximize, restore) | **counted, not guessed**: `mounted` holds how many bodies show that pane, so only the last one out parks it, and even then on the next tick. The grid unmounts before it mounts (one commit); the canvas mounts the maximize layer's body a commit *before* its React Flow node drops the wrapper's, because a node's data is applied in an effect — with a tick alone that order suspended the socket of the pane the viewer was looking at (measured: `readyState` 3 and keystrokes lost, C2 session 2) | the same xterm is re-parented (`ShellTerm`) and refits |
| `terminal.deleted` / `agent.deleted` on the feed | `forgetPane`: a pane the Matrix holds suspended is disposed at once; one still mounted is disposed when its body unmounts (the gone row replaces it); one a tab owns is the tab's | — |
| focus | one focused panel per matrix (click or arrows); only the focused **and engaged** panel gets `autoFocus`, so only it calls `term.focus()`; every visible panel re-claims its pane and fits on reveal |
| the same terminal twice on one matrix | the picker does not offer it; a second browser that has not heard yet gets the server's 409 and its message, and the optimistic wrapper goes |

**Saving** (plan §4.7). A drag or resize stop moves the local rows at
once; `layoutDiff` against the last saved rows accumulates the subset;
`PATCH …/layout {ifUpdatedAt, panels}` goes 500 ms later, and at once on
hide, on a switch and on unmount. The 200 is applied like a
`matrix.layout` frame; a 409 refetches the matrix and toasts "Matrix
changed elsewhere — reloaded." `onLayoutChange` is ignored while hidden,
at width 0 and during a gesture; feed frames that arrive during a gesture
are applied after it. The store is kept compacted with the grid's own
compactor (`compactPanels`): react-grid-layout compacts a controlled
layout silently and fires no `onLayoutChange` when only the prop moved,
so the surface runs the same compaction on every panels change and saves
that difference through the same path — a matrix stored with gaps (a
remove from another browser) is saved compacted once, on the next
reveal. A `matrix.panel.added` frame that beats its own POST answer
settles the pending wrapper of that binding at once. Adding a panel places it at `nextSlot` (4×14), shows
the wrapper at once and settles on the 201 — a refusal shows its message
and removes the wrapper; Remove and Delete apply locally on the 204;
Delete confirms through the app's alert dialog and the tab moves to the
last-opened or first matrix, else the empty state.

**Copy** (plan §4.8): "No matrix yet. A matrix shows many agents and
terminals side by side, live." — New matrix; "Add your first panel." —
Add panel; picker: "Everything is already on this matrix." / "No agents or
terminals yet — create one from the sidebar." — Close.

**Keyboard** (plan §4.6 "Focus"). The surface holds two bits: which panel
is focused (`focusedId`) and whether the keyboard is inside its terminal
(`engaged`). The wrapper is the roving tab stop of its matrix — `tabIndex`
0 on the focused panel, or the first in reading order until one is
focused, -1 on the rest — and hands every key to the surface, which owns
the model. `panelOrder` and `neighborPanel` (`web/shared/domain/matrix.js`)
are the arithmetic; `paneLeaveKey` (`termKeys.js`) is the chord xterm hands
back.

| On the panel chrome | Does |
|---|---|
| click, or arrow from a neighbour | focus this panel; the wrapper takes DOM focus and scrolls into view, which loads it |
| `←` `→` | the nearest panel that shares rows; ends of the row stop |
| `↑` `↓` | the nearest panel that shares columns, else the nearest in that half — a row holding one panel is never a dead end |
| `Home` / `End` | first / last panel in reading order |
| `Enter` | engage: the pane takes the keyboard (`autoFocus`); on a placeholder row with one action, run it |
| `Delete` / `Backspace` | remove the panel, with an **Undo** notice that re-adds the same binding at its old rectangle; focus moves to the next panel in reading order, else the previous |
| `Esc` while a panel is maximized | restore |

| Inside the pane | Does |
|---|---|
| `Shift`+`Esc` | leave: the wrapper (or the maximize layer) takes focus back, `engaged` off |
| every other key, `Esc` included | the guest's — xterm writes it to the shell; `Shift`+`Esc` is the only chord the app takes, and xterm encodes it as the same `\x1b`, so no TUI loses a key it had |

**Maximize** is host-level state on the surface, not a transform on the
grid item: the maximized panel's body renders in a layer over the grid
(`.mx-max`, absolute inside `.mx-stage` — no transformed ancestor to
break), the wrapper keeps its slot and says *Shown maximized* with a
Restore button, `.mx-body` goes `inert`, and the layout is untouched. The
pane follows the visible host, so it is the same xterm and it refits to
the layer (measured: a shell went 58×29 → 120×47 and back). The header
button toggles; `Esc` on any chrome restores — never from inside the pane
(the TUI may need it) and never out of a dialog, menu or listbox, which
own their own `Esc`.

A binding whose target is deleted while the page is up keeps the name and
hint it last showed, so the gone row reads *grid20 · Shell · Gone* instead
of the raw id; a page opened after the deletion has nothing to remember
and falls back to the ref.

### Canvas (phase C2)

`MatrixCanvas.jsx` is the peer of `MatrixGrid.jsx`: `@xyflow/react` 12.11.6,
**lazy-imported** (eager it is +63.3 KB gzip on the desktop's main chunk;
split it costs 205 B and lands in a 49 KB gzip chunk only a viewer who opens
a canvas fetches). It is a host and nothing more — the surface keeps the
store, the save flow, the focus model, the keyboard, the picker and the
maximize layer, and every panel is the **same `Panel` wrapper** rendered as
one custom node type, so a panel has the same header, chip, actions and keys
in both modes. Configuration is C0's
([`docs/benchmarks/2026-09-10-node-canvas.md`](../benchmarks/2026-09-10-node-canvas.md)):
`minZoom` 0.2, `maxZoom` 1.5, `snapToGrid` on an 8 px `snapGrid` (the canvas
unit, so a drag lands on whole units), `onlyRenderVisibleElements` **off**
(on, it unmounts a node with no dwell — 54 socket suspends and kicks over
three fast pans), `NodeResizer` for resize (`onResizeEnd` fires once; it
does not honour `snapGrid`, so `pxToUnits` rounds), the body marked `nodrag
nowheel` with the header as the drag handle, marquee selection on a left
drag, pan on the middle or right button or Space, and a `<MiniMap />` and a
zoom cluster themed from our tokens (React Flow's defaults are a near-white
panel over the app's dark surface). Node position is `x · 8, y · 8` px and
size `w · 8, h · 8`; `unitsToPx` / `pxToUnits` are the only place that
multiplies.

**Zoom decides the body** (plan §4.3, `loadPolicy` with `{zoom, band}`,
`zoomBody`, `pointerAtZoom` — one test per row in `matrix.test.js`). The
reason is not performance, it is correctness: **xterm divides the pointer's
offset inside the transformed rect by the untransformed cell size**, so the
cell it reports is `cell × zoom` — at 0.8 a click aimed at column 50 row 15
arrives as column 40 row 12 — and PiCode ships tmux `mouse on`, so that
wrong coordinate reaches copy mode and every mouse-aware TUI. The DOM
renderer stays crisp, so nothing on screen would say so; the surface has to.

| Condition | Body |
|---|---|
| in the band, `zoom = 1.0` (± 0.005) | live pane, attached — **the only state that takes a pointer** |
| in the band, `0.8 ≤ zoom < 1.0`, or `zoom > 1.0` | live pane, readable, taking keys; `pointer-events: none` on the body and a layer over it whose click **snaps the plane to 1** and hands the pointer over |
| in the band, `zoom < 0.75` | **still**: `term.buffer.active` as text, monospace, inert; the header stays live from the feed |
| `zoom < 0.4` | **name-plate**: face, name and status colour sized by `1 / zoom`, because a still down there is grey texture and the header does not resolve either |
| outside the band, any zoom | today's placeholder; the socket suspends after the 5 s hysteresis |
| focused and engaged | the plane animates to `zoom = 1` before the pane takes keys — Enter engages through the same snap |

The rule is about a **cell**, so it only binds a body that has one.
`hasPane` (`web/shared/domain/matrix.js`, the allow-list `PANE_STATES`) is
the gate on all three halves — the `is-inert` / `is-still` pointer-events
rule, the `.mx-snap` layer, and **the still row of `loadPolicy` itself**:
the wrapper tells the loader `setPane(id, hasPane(model))` and the pure
module decides the row, so no component holds an `if` about it. The rows
that answer with one line and one action (a managed agent's *Open*, a gone
terminal's *Remove*, a stopped agent's *Run*, the error row's *Try again*)
keep their own button at every zoom. They have no cell to miss, and a
visible button that zooms instead of doing what it says is worse than the
risk it was protecting against. A name-plate is the other way round: below
0.4 the plate *is* the panel, there is no header left to reach, so it takes
the layer whatever it is bound to.

**A body that is not a pane** — a note, a file, a diff — has no `term.buffer` to
capture, so the still row never applies to it. It follows the other three:

| Condition | A body with no pane |
|---|---|
| in the band, `zoom ≥ 0.4` | renders normally, pointer and all: DOM text scales, and the pointer bug is xterm's alone |
| `zoom < 0.4` | the same **name-plate** as everything else — down there nothing textual is legible |
| outside the band, any zoom | unmounted, like every other body: there is no socket to suspend, and mounting again is one fetch |

The band it reports back is still the pane band, because the band is what
bounds the *attaches*; `zoomBody(zoom, band, pane)` answers both rows from
one table and `loadPolicy` picks per entry (`matrix.test.js`, three rows).

The band is **hysteretic** — live at 0.8 and above, still below 0.75 — so a
viewer parked on the boundary does not thrash the attaches, and it flips
when the gesture ends (`onMoveEnd`) rather than once per wheel frame — one
crossing per gesture instead of sixty, and the panes stay live *during* a
zoom-out. The band itself is
**screen-space**, so zooming out puts everything in it (C0: 500 of 500 at
0.2); what bounds the attaches down there is the still rule, not the band.

**Stills** (`stills.js`) are memory-only, never persisted, and cost 0.02 ms
for a full screen. They are captured **before the flip, never at unmount** —
React renders the still body before it runs the live body's cleanup, so a
capture in the cleanup is one crossing late — and refreshed while the panes
are live. A camera that *arrives* below the band (a stored viewport, or the
switch out of grid mode, where the panes went live in the other host) fills
the gaps from the xterm instances a suspend keeps. A panel with no capture
yet shows the unloaded body's one muted line, and a still whose panel has
moved on the feed since carries its age in the header.

**The camera is per viewer**: `localStorage` `picode-matrix-view:<id>` holds
`{x, y, zoom}`, restored on open, debounced on `onMoveEnd`, with a fit to
the panels the first time a viewer opens that matrix. It never reaches the
store — a pan is not an edit (ADR-0113).

**Tidy** (`tidyCanvas`, the header menu) is canvas mode's answer to
react-grid-layout's automatic compaction: explicit instead of silent. It
lays the panels out in reading order, sizes kept, three default-sized panels
across (98 units, the grid's 12 columns and their gutters), each row as tall
as its tallest panel, and saves once through the layout PATCH. Grid mode
keeps RGL's compactor untouched this phase (plan §3's two-step).

**Tidy arranges; it never resizes.** It is standing in for RGL's compactor,
which never resized either — it only ever moved items up into free space —
so a Tidy that also normalised sizes would silently throw away a resize the
owner of the panel made on purpose. Rows are as tall as their tallest panel
for the same reason: the row gives way to the panel, not the other way
round.

**The switch** is the `Grid | Canvas` segmented control in the header. It
PATCHes `mode` under `ifUpdatedAt` (after flushing the moves made under the
old mode, whose answer carries the precondition the switch then uses) and
applies the answer through `applyMatrixEvent` like every other mutation; a
409 refetches and says *Matrix changed elsewhere — reloaded.* Because the
transform is implemented twice against the same fixtures — Go writes it,
`gridToCanvas` / `canvasToGrid` preview it — the preview **is** the answer,
so the panels move at once and the 200 reconciles them. Canvas → grid asks
first in one line (*Grid mode packs the N panels into 12 columns; where they
sit on the plane is not kept*) because that direction is lossy; grid →
canvas loses nothing and does not ask.

**What canvas mode adds to the keyboard** (everything phase 3 shipped keeps
working — `neighborPanel` was already a 2D spatial search, so the arrows
need no change on a free plane): `+` / `-` zoom, `0` fits, Space-drag pans,
Enter engages through the snap to 1, and moving focus to an off-screen panel
**pans it into view**, which is what loads it. A panel already on screen is
left where it is: an arrow key must not shove the plane about.

**Accepted in a browser (2026-09-10, C2 session 2)** on a scratch instance
with eight panels — an interactive agent's TUI, a managed agent and six
shells — plus a 24-panel matrix, driven at 1600 × 1200:

| Row | Measured |
|---|---|
| the pointer rule, end to end | at **1.0** a real click aimed at cell (25, 13) of a 29 × 16 pane came back from the terminal as `^[[<0;25;13M` — the cell aimed at, through tmux `mouse on`. At **0.83** xterm's own mapping for the same point answered **(21, 11)** — 4 columns and 2 rows out — the body read `pointer-events: none`, the click produced **no** report at all and moved the plane to 100 %, and the click after it reported (25, 13) again |
| bodies per zoom | 1.0: 7 live, 7 attached. 0.83: 7 live, every panel `is-inert`. 0.69: stills, **0 attached, 7 suspended, 7 instances kept**. 0.30: name-plates, 0 attached |
| chunk loading under the transform | at rest 7 attaches and **7 tmux clients**, one per session. A 3 883 px pan at 4 678 px/s (61.4 fps) and +10 s: 0 attached, 7 suspended, the 7 instances still there. Back at 4 781 px/s (62.8 fps): the same 7 instances reattached, 7 tmux clients. Zoom 0.2 → 0 attaches; back to 1.0 → the same 7 |
| a marquee of many | 24 nodes selected at **61.9 fps**, dragging all 24 at **62.7 fps** (worst frame 16.8 ms), **one** layout PATCH of 24 rows, every rectangle moved by exactly (25, 19) units. C0's 31 fps was 20 nodes among 500 with nine live bodies; this is a lighter scene and not a refutation of it |
| mode switch with live panes | grid → canvas → grid with six live panels: **zero** socket transitions, zero disposals, zero xterm replacements over both switches (polled at 40 ms) |
| maximize on a canvas | the pane refits 29 × 16 → **120 × 58**, tmux follows, the wrapper says *Shown maximized*, and the socket stays open in **both** directions |
| a real 409, from a second browser | a layout PATCH answered 409, refetched (200), toasted *Matrix changed elsewhere — reloaded.*, rolled the optimistic move back and took the other browser's; the mode PATCH did the same and the mode stayed `canvas`. No lingering wrapper either time |
| the camera is per viewer | two browsers on one matrix at (−562, −623.4, 1.0) and (565, 288, 0.712); neither moved the other, and a reload restored each its own |
| fullscreen | the canvas fills the window inside `picode-focus`, the left and top reveals float over it, and leaving the mode leaves the camera exactly where it was |

Two things this pass found and fixed: **maximize on a canvas suspended the
pane's socket** (the deferred node data — the ownership table above), and a
body that is not a terminal was made pointer-inert with it (the `hasPane`
gate in §Canvas). One thing it did not fix: the scratch daemon leaves
`tmux: client` zombies, so three of seven sessions still listed a client
25 s after their socket was suspended. It never exceeded one client per
session and production shows the same pattern, so it is not the canvas's.

**Accepted in a browser (2026-09-10, C3)** on a scratch instance at
1600 × 1100 with five panels — a note, a file, a diff, a shell and a second
note — in both themes:

| Row | Measured |
|---|---|
| the picker | five groups (Agents · Terminals · Pins · Open files · Changes to an open file) from `GET /api/pins` and `host.openTabs`; a binding already on the matrix is not offered, and with no file tab open the Files line reads *No file is open — open one in a tab…*. `__picodeOverlayAudit()` `ok` with it open |
| the bodies | the note renders the pin's markdown (headings, task list, table, quote, fenced code); the file is the editor with Preview/Raw/Save; the diff shows `+2 −0` with **Open file**; the shell is a live xterm |
| the feed | a `PATCH /api/pins/{id}` from another client changed the note's body in place, and `matrix.panel.added` from a second client drew a fifth panel — no timer either side |
| the zoom rows | 100 %: all four live, none inert. 78 %: only the terminal is `is-inert` with the snap layer. 64 % and 52 %: the terminal is a still, the note, file and diff still render. 42 %: same. 35 %: every panel a name-plate. Back at 100 %: all four live again |
| the gone rows | deleting the pin left *That pin is gone.* — Remove (name kept until the page reloads, then the ref); deleting the terminal left the file and the diff at *Where this file was read from is gone.* — Remove, chip **Gone**, with only Remove offered |
| unsaved work | typing in the file panel turned the chip to **Unsaved**; with the matrix tab hidden for 9 s every other body unloaded (`data-loaded="0"`) and that one did not; maximizing it and switching canvas → grid both kept the text and the chip |
| the store's answers | 409 *This note is already on this matrix* and *This diff is already on this matrix*; 400 *ref must be `<owner>:<id>:<path>` with owner t, a or w* and *kind must be agent, terminal, note, file or diff*; the same path as a `file` and as a `diff` is two panels |
| Open in Pin Studio | the note header's Open lands on `#/pins/<id>` with that pin loaded |

**Accepted in a browser (2026-09-10, C4)** on a scratch instance at
1440 × 900 in both themes, seeded with two workspaces (`Alpha` at the
worktree, `Beta` at `/tmp/picode-qa-beta`), four `pi` agents with recorded
sessions — Atlas, Bravo, Cleo in Alpha, Delta in Beta — and a five-panel
canvas matrix whose fifth panel is a note. Every mailbox claim was **driven
through the MCP endpoint** with a bearer the scratch minted, never inferred
from the UI:

| Row | Observed |
|---|---|
| baseline, before any edge | `list_contacts` — Atlas sees `Bravo` (same workspace, ADR-0104 unchanged); Delta sees **nothing** |
| draw between two enrolled panels, one workspace | a real pointer drag from Atlas's connector to Bravo's: the line appeared, **one** `POST …/edges` → 201, no dialog, and a **second browser session** that never touched the plane drew the same `edge-90d707f3e244` from `matrix.edge.added` alone |
| draw when an end is not enrolled | *Connect Cleo?* — "…lets Atlas and Cleo message each other. Neither can read the other's history." **Cancel**: still 1 edge, Cleo still has no connection, and the only request the whole gesture made was the one `GET /api/communication` it asks with. Confirm: `POST /api/communication` 201 **then** `POST …/edges` 201, and both lines read `is-live` |
| draw across two workspaces | a second confirm, on its own: *"Atlas works in ~/picode/.worktrees/matrix-edges and Delta in /tmp/picode-qa-beta…"*. After it, **`list_contacts` for Delta returned `Atlas`** (workspace `alpha-ed1885`) where it had returned nothing, and `send_message` Delta → Atlas was **accepted** and landed in the Messages history |
| remove the line | the chip on the line → *Remove this link? Delta and Atlas will no longer be able to message each other.* → Delta's contacts **empty again**, Atlas no longer lists Delta, and `send_message` answered *connection is disabled or its recorded conversation changed* |
| revoke one connection in Messages | Advanced → Cleo → **Disable**: the audit row turned amber with *Cleo's connection was revoked; the link grants nothing.* (no reload — the `peer.*` row), the plane drew the edge `is-broken`, and Atlas's contacts dropped Cleo |
| re-connect | drawing Bravo → Cleo offered the enrolment again; confirming it made **all three** edges live at once, including the one that had been broken — the grant is derived, so nothing had to be repaired |
| the session moves | `PATCH /api/agents/{delta}` to a new `sessionPath`: the link read *Delta's session changed; the link grants nothing.*, the grid chips turned amber (`3 links, 1 broken`), and the old bearer stopped answering. Putting the path back made it live again — **both ways, with no reload** |
| delete one panel | removing the Delta panel took its edge with it (3 → 2 edges, no per-edge event) and Delta's contacts went **empty** |
| the two meanings of Delete | selecting the line and pressing <kbd>Delete</kbd> asked the same one-line confirm and removed the edge (3 → 2); with a **panel** focused, <kbd>Delete</kbd> still removed the panel (5 → 4) with its Undo toast and left the edges alone |
| a note panel | no connector on it, and a hand-made `POST …/edges` with the note's id answered **400** *panel … is a note panel and has no mailbox: an edge links agent or terminal panels* |
| the audit list | the same edges as the plane, each with both ends, both kinds, the matrix, and whether it grants; its **Remove** confirmed in one line and revoked for real (Delta's contacts `["Cleo"]` → `(none)`); the list survived a reload; it listed the cross-folder pair with **no** workspace chosen |
| grid mode | the header chips read 2 · 2 · 3 · 1 against four edges, amber with the count of broken ones, and the note panel has none |
| geometry | `__picodeOverlayAudit()` `ok: true` on both dialogs in both themes; dialog buttons 36 px = `--ctl-h`; no misaligned `[data-align-row]` |

Four defects this pass found and fixed, all of them about a line you can
cover: the chip rendered **under** the panels (React Flow draws edge labels
below nodes), then covered the panel name once lifted, then made a
three-link board wear three pills on three headers, and finally a link
between two adjacent panels had **no hittable pixel at all** — its whole run
was behind them. The current rules (elevate, push off the line's normal,
show a live chip only on hover or selection, reveal the reason on the
chip's own hover) are each one of those four. A fifth was not visual: a
`PATCH` of an agent's session path left every chip reading live, because
only `peer.*` was being watched.

**Re-measured on a scratch (2026-09-09, phase 3 session 2)** against
[`docs/benchmarks/2026-09-09-matrix-live-grid.md`](../benchmarks/2026-09-09-matrix-live-grid.md):
nine live TUI bodies at 60.0 fps in-page, zero long tasks and 4.2 % of one
core in the page's renderer (0.1 % idle); dragging with those nine live at
60.3–60.9 fps; a pass at the study's speed (16 000 px/s, each panel 0.17 s
in the band) mounted 8 bodies. The dwell is a *speed* rule, not a distance
one: the same 4 s pass over a short 46-panel matrix leaves every panel
0.70 s in the band, so all of them legitimately load, and the 5 s
hysteresis returns to the steady 12–21 within 8 s.
