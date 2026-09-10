# Matrix (ADR-0108)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

A matrix is a named 12-column grid of agent and terminal panels — the
Matrix app (plan: `docs/plans/matrix-app.md`; phase 0 study:
`docs/benchmarks/2026-09-09-matrix-live-grid.md`). Persistence, the API
family and the feed events are phase 2 (ADR-0108); the desktop surface —
react-grid-layout, chunk loading, the picker — is phase 3 and is the
**Surface** section at the end.

**Data model (migration 041).** `matrices(id, name, compact, created_at,
updated_at)` and `matrix_panels(id, matrix_id → matrices ON DELETE
CASCADE, kind, ref, x, y, w, h, created_at, UNIQUE(matrix_id, kind,
ref))`. One row per panel, never one blob: a drag rewrites the rows that
moved and the unique index refuses a duplicate binding. A panel id is a
random slot (`panel-` + 12 hex), never derived from `ref`, so the same
terminal on two matrices is two panels. `kind ∈ {agent, terminal}`; `ref`
is the agent or terminal id, with **no foreign key on purpose** — the
store is ignorant of the binding, deleting an agent or terminal leaves the
panel, and the UI renders the target as gone. `compact ∈ {vertical,
none}` (default `vertical`; `none` is the v2 free canvas). `cols` is 12
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
| `GET /api/matrices` | `{matrices: [summary]}` — `id, name, compact, createdAt, updatedAt, panelCount`, by name (ASCII case-folded), then creation |
| `POST /api/matrices {name}` | 201 summary; 400 with the limit named |
| `GET /api/matrices/{id}` | 200 summary + `panels: [{id, kind, ref, x, y, w, h, createdAt}]` (one read on open; 500 panels ≈ 50 KB) |
| `PATCH /api/matrices/{id} {name?, compact?, ifUpdatedAt}` | 200 summary; 400 on an empty patch or a broken rule; 409 when the row moved on |
| `PATCH /api/matrices/{id}/layout {ifUpdatedAt, panels: [{id, x, y, w, h}]}` | the changed subset, one transaction, all or nothing; 200 `{id, updatedAt, panels}`; 409 when stale |
| `POST /api/matrices/{id}/panels {kind, ref, x, y, w, h}` | the client places (`nextSlot`, phase 3), the server validates; 201 `{id, updatedAt, panel}`; 409 on a duplicate binding |
| `DELETE /api/matrices/{id}/panels/{panelId}` | 204; the feed carries the new `updatedAt` |
| `DELETE /api/matrices/{id}` | 204; the panels cascade |

`ifUpdatedAt` may also travel as an `If-Match` header (the pins
convention); empty means no precondition. Adding or removing a panel takes
none — a picker must not fail because someone dragged.

**Limits refuse and never truncate** (400, the limit named; runes, not
bytes): 64 matrices, 500 panels per matrix, 80 characters of name (empty
or only spaces is "name is required"), `x ≥ 0`, `y ≥ 0`, `w ≥ 4`
columns, `h ≥ 8` rows, `x + w ≤ 12`. The messages are the contract:
`web/shared/domain/matrix.js` repeats them (`validateName`,
`validateCompact`, `validatePlacement`, `validatePanel`) so the UI can
refuse before asking.

**Events** (durable, appended in the mutation's transaction; `touches(ev,
["matrix"])` keys on the prefix, so `matrix.panel.*` reaches the surface
with the rest):

| Event | Data |
|---|---|
| `matrix.created`, `matrix.updated` | the summary |
| `matrix.layout` | `{id, updatedAt, panels: [the subset that moved]}` |
| `matrix.panel.added` | `{id, updatedAt, panel}` |
| `matrix.panel.removed` | `{id, updatedAt, panelId}` |
| `matrix.deleted` | `{id}` — the panels go with it, without an event each |

**Client contract** (`web/shared/domain/matrix.js`, pure): `MATRIX_LIMITS`,
`MATRIX_KINDS`, `MATRIX_COMPACT`, `MATRIX_EVENTS`; `normalizeMatrix`,
`normalizeMatrixList`, `normalizePanel`, `normalizeMatrixDetail` drop junk
the way `contracts/appPrimitives.js` does (a panel without a whole
rectangle or a known binding is dropped); `applyMatrixEvent(state, ev)`
reduces the six events over `{ list: [summaries], byId: { id: { matrix,
panels } } }` — an event for a matrix that is not loaded only touches the
summary list, a `created`/`updated` for a matrix the list never saw
inserts it (the summary is complete), the list stays sorted by name, and
junk or an unknown type returns the same object.

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
| `MatrixGrid.jsx` | react-grid-layout 2.2.4, v2 API: `useContainerWidth` on the canvas, `gridConfig {cols 12, rowHeight 24, margin 8}`, drag by `.mx-head` with `.mx-actions` as the cancel zone, `se`/`s`/`e` handles, `fastVerticalCompactor` from `react-grid-layout/extras`, children keyed by panel id, `minW 4 / minH 8` |
| `Panel.jsx` | the wrapper every panel keeps (and `PanelHead`, which the maximize layer reuses): face, name, hint, the sidebar's chip (`agentRowStatus` / `terminalStatus`), Open · Maximize · Remove from matrix; it observes its own visibility; two memo layers so a grid re-render never touches a body |
| `PanelBody.jsx`, `TerminalPanel.jsx` | loaded → `TermSurface` (the tab's engine; a terminal first `POST /api/terminals/{id}/open`s for its live record); unloaded → one muted line with the feed's last state; the §4.4 rows below as one line + one action |
| `PanelPicker.jsx`, `NameDialog.jsx` | cmdk list of the agents and terminals not yet on this matrix, with the sidebar's faces and words; the name form (`matrixNameSchema`, Zod, the store's messages, `noValidate`) |
| `chunkLoader.js`, `paneOwnership.js` | the `IntersectionObserver` glue over the pure `loadPolicy`; what unloading does to an attach |
| `web/shared/domain/matrix.js` | `nextSlot`, `layoutDiff`, `bindingState`, `loadPolicy`, `suspendedToDispose`, `panelOrder`, `neighborPanel` (+ `PANEL_DEFAULT` 4×14, `PANEL_DIRECTIONS`) — one test per row below in `matrix.test.js` |

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
scroll container, `rootMargin: 100% 0px`.

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
| the body moves host (maximize, restore) | the release waits a tick, so the body that mounts in the other host claims the pane first and the socket never bounces | the same xterm is re-parented (`ShellTerm`) and refits |
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

**Re-measured on a scratch (2026-09-09, phase 3 session 2)** against
[`docs/benchmarks/2026-09-09-matrix-live-grid.md`](../benchmarks/2026-09-09-matrix-live-grid.md):
nine live TUI bodies at 60.0 fps in-page, zero long tasks and 4.2 % of one
core in the page's renderer (0.1 % idle); dragging with those nine live at
60.3–60.9 fps; a pass at the study's speed (16 000 px/s, each panel 0.17 s
in the band) mounted 8 bodies. The dwell is a *speed* rule, not a distance
one: the same 4 s pass over a short 46-panel matrix leaves every panel
0.70 s in the band, so all of them legitimately load, and the 5 s
hysteresis returns to the steady 12–21 within 8 s.
