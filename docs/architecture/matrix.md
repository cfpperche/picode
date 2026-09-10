# Matrix (ADR-0108, ADR-0113)

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

**Data model (migrations 041 and 043).** `matrices(id, name, compact, mode,
created_at, updated_at)` and `matrix_panels(id, matrix_id → matrices ON DELETE
CASCADE, kind, ref, x, y, w, h, created_at, UNIQUE(matrix_id, kind,
ref))`. One row per panel, never one blob: a drag rewrites the rows that
moved and the unique index refuses a duplicate binding. A panel id is a
random slot (`panel-` + 12 hex), never derived from `ref`, so the same
terminal on two matrices is two panels. `kind ∈ {agent, terminal}`; `ref`
is the agent or terminal id, with **no foreign key on purpose** — the
store is ignorant of the binding, deleting an agent or terminal leaves the
panel, and the UI renders the target as gone. `compact ∈ {vertical,
none}` (default `vertical`) and keeps meaning only in grid mode; `mode ∈
{grid, canvas}` (migration 043, default `grid`, so every row written before
it reads the meaning its rectangles already had). `cols` is 12 in grid mode
and not stored. `updated_at` (RFC 3339 UTC, nanoseconds) bumps on every
mutation and is the optimistic-concurrency key. The store
(`internal/store/matrix.go`) is the only writer; every mutation is one
transaction with its event appended inside it (ADR-0048), one row per
mutator in `TestEveryMutationAppendsAnEvent`.

**Routes** (`internal/server/matrix.go`; the auth gate applies as for
every `/api/*`). Every mutation answers with the payload its event
carries, so the client has one reducer path for a response and a frame.

| Route | Answers |
|---|---|
| `GET /api/matrices` | `{matrices: [summary]}` — `id, name, compact, mode, createdAt, updatedAt, panelCount`, by name (ASCII case-folded), then creation |
| `POST /api/matrices {name}` | 201 summary; 400 with the limit named |
| `GET /api/matrices/{id}` | 200 summary + `panels: [{id, kind, ref, x, y, w, h, createdAt}]` (one read on open; 500 panels ≈ 50 KB) |
| `PATCH /api/matrices/{id} {name?, compact?, mode?, ifUpdatedAt}` | 200 summary; 400 on an empty patch or a broken rule; 409 when the row moved on |
| …the same route when `mode` changes it | 200 summary **plus** `panels: [{id, x, y, w, h}]` — every panel the switch moved, the shape the layout patch answers with, so the client has one reducer path. Same mode: the untouched summary, `panels: []`, no event, `updatedAt` unchanged. Unknown mode: 400 "mode must be grid or canvas". A `name`/`compact` in the same body is applied first (its own `matrix.updated`), so a broken one refuses the patch before the switch |
| `PATCH /api/matrices/{id}/layout {ifUpdatedAt, panels: [{id, x, y, w, h}]}` | the changed subset, one transaction, all or nothing, every rectangle judged by the matrix's current mode; 200 `{id, updatedAt, panels}`; 409 when stale |
| `POST /api/matrices/{id}/panels {kind, ref, x, y, w, h}` | the client places (`nextSlot`, phase 3), the server validates against the matrix's current mode; 201 `{id, updatedAt, panel}`; 409 on a duplicate binding |
| `DELETE /api/matrices/{id}/panels/{panelId}` | 204; the feed carries the new `updatedAt` |
| `DELETE /api/matrices/{id}` | 204; the panels cascade |

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
| `matrix.panel.removed` | `{id, updatedAt, panelId}` |
| `matrix.deleted` | `{id}` — the panels go with it, without an event each |

**Client contract** (`web/shared/domain/matrix.js`, pure): `MATRIX_LIMITS`,
`MATRIX_KINDS`, `MATRIX_COMPACT`, `MATRIX_MODES`, `MATRIX_EVENTS`,
`UNIT_PX` (8); `normalizeMatrix`,
`normalizeMatrixList`, `normalizePanel`, `normalizeMatrixDetail` drop junk
the way `contracts/appPrimitives.js` does (a panel without a whole
rectangle or a known binding is dropped); `applyMatrixEvent(state, ev)`
reduces the seven events over `{ list: [summaries], byId: { id: { matrix,
panels } } }` — an event for a matrix that is not loaded only touches the
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
| add panel | same (kind, ref) already on this matrix | 409 "already on this matrix" |
| add panel | 500 panels exist | 400 "limit: 500 panels per matrix" |
| add panel | w < 4, h < 8, x < 0, y < 0, x + w > 12, kind not agent/terminal, ref empty | 400 naming the rule |
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
| `PanelBody.jsx`, `TerminalPanel.jsx` | loaded → `TermSurface` (the tab's engine; a terminal first `POST /api/terminals/{id}/open`s for its live record); unloaded → one muted line with the feed's last state; the §4.4 rows below as one line + one action |
| `PanelPicker.jsx`, `NameDialog.jsx` | cmdk list of the agents and terminals not yet on this matrix, with the sidebar's faces and words; the name form (`matrixNameSchema`, Zod, the store's messages, `noValidate`) |
| `chunkLoader.js`, `paneOwnership.js` | the `IntersectionObserver` glue over the pure `loadPolicy`; what unloading does to an attach |
| `web/shared/domain/matrix.js` | `nextSlot`, `layoutDiff`, `bindingState`, `loadPolicy` (with the zoom), `zoomBody`, `pointerAtZoom`, `unitsToPx` / `pxToUnits`, `tidyCanvas`, `normalizeViewport`, `suspendedToDispose`, `panelOrder`, `neighborPanel` (+ `PANEL_DEFAULT` 4×14, `PANEL_DEFAULT_CANVAS` 32×42, `CANVAS_ZOOM`, `PANEL_DIRECTIONS`) — one test per row below in `matrix.test.js` |

**The fleet decides what a panel is bound to**, so the surface draws the
skeleton until the desktop says it has one: `host.fleet.loaded` is the
App's `bootstrapped`, and until it turns true an empty fleet means "not
read yet", not "deleted". Without it every terminal panel showed *That
terminal is gone.* for the length of the boot fetch — 15 s on a fixture
with 45 terminals (each row carries git state). A host that passes no
`loaded` is taken at its word.

**Reads.** `GET /api/matrices` and `GET /api/matrices/{id}` once on open
and on a reveal older than 10 s; then the feed only: `matrix.*` through
`applyMatrixEvent`, agents and terminals through the host's fleet,
`agent.tui` for the Working chip after one `GET /api/tui-working` for the
agents on the matrix. No timer.

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
`hasPane` (PanelBody.jsx) is the gate on both halves — the `is-inert` /
`is-still` pointer-events rule and the `.mx-snap` layer: the four rows that
answer with one line and one action (a managed agent's *Open*, a gone
terminal's *Remove*, a stopped agent's *Run*, the error row's *Try again*)
keep their own button at every zoom. They have no cell to miss, and a
visible button that zooms instead of doing what it says is worse than the
risk it was protecting against. A name-plate is the other way round: below
0.4 the plate *is* the panel, there is no header left to reach, so it takes
the layer whatever it is bound to.

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

**Re-measured on a scratch (2026-09-09, phase 3 session 2)** against
[`docs/benchmarks/2026-09-09-matrix-live-grid.md`](../benchmarks/2026-09-09-matrix-live-grid.md):
nine live TUI bodies at 60.0 fps in-page, zero long tasks and 4.2 % of one
core in the page's renderer (0.1 % idle); dragging with those nine live at
60.3–60.9 fps; a pass at the study's speed (16 000 px/s, each panel 0.17 s
in the band) mounted 8 bodies. The dwell is a *speed* rule, not a distance
one: the same 4 s pass over a short 46-panel matrix leaves every panel
0.70 s in the band, so all of them legitimately load, and the 5 s
hysteresis returns to the steady 12–21 within 8 s.
