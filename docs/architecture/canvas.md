# Canvas (ADR-0108, ADR-0116, ADR-0118)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

A canvas is a named board of agent and terminal panels on one plane you
pan and zoom — the Canvas app (plan: `docs/plans/matrix-app.md`; phase 0
study: `docs/benchmarks/2026-09-09-matrix-live-grid.md`). Persistence, the
API family and the feed events are phase 2 (ADR-0108); the desktop surface
— chunk loading, the picker — is phase 3 and is the **Surface** section at
the end. The plane itself is C2 of `docs/plans/matrix-canvas.md` and is the
**Canvas** part of that section. ADR-0118 is what renamed the whole family
and left one layout engine.

**Data model (migrations 041, 044 and 045).** `canvases(id, name, compact,
created_at, updated_at)` and `canvas_panels(id, canvas_id → canvases ON DELETE
CASCADE, kind, ref, x, y, w, h, created_at, UNIQUE(canvas_id, kind,
ref))`. One row per panel, never one blob: a drag rewrites the rows that
moved and the unique index refuses a duplicate binding. A panel id is a
random slot (`panel-` + 12 hex), never derived from `ref`, so the same
terminal on two canvases is two panels. `kind ∈ {agent, terminal, note, file, diff}`
(the **Kinds** table below says what each `ref` is), with **no foreign key
on purpose** — the store is ignorant of the binding, deleting an agent, a
terminal, a pin or a file leaves the panel, and the UI renders the target as gone.
`compact ∈ {vertical, none}` (default `vertical`) is still the column
`UpdateCanvas` patches, and nothing compacts a plane, so it decides nothing
today. `updated_at` (RFC 3339 UTC, nanoseconds) bumps on every
mutation and is the optimistic-concurrency key. The store
(`internal/store/canvas.go`) is the only writer; every mutation is one
transaction with its event appended inside it (ADR-0048), one row per
mutator in `TestEveryMutationAppendsAnEvent`. A third table, `canvas_edges`
(migration 044), links two panels of one canvas — the **Edges** section
below, and ADR-0116 for what it grants.

**What migration 045 did** (ADR-0118, superseding ADR-0113). ADR-0113 had
given a board a `mode` column (migration 043, default `grid`), so a
rectangle meant either a cell of a 12-column grid or a unit of this plane.
Migration 045 renamed `matrices`, `matrix_panels` and `matrix_edges` to
`canvases`, `canvas_panels` and `canvas_edges`, renamed the `matrix_id`
column on both children to `canvas_id` and recreated the two endpoint
indexes under their new names; it dropped `mode`; and in the same
transaction it rewrote every panel of a board still in `grid` into canvas
units — `x·8`, `y·3`, `w·8`, `h·3`, clamped to `w ≥ 32` and `h ≥ 28`, so a
panel at the old 8-row minimum comes out 28 units tall instead of 24 and may
overlap the one below it, which free placement allows. A board already on
the plane was untouched. That conversion is the irreversible half: going
back means restoring the column and packing every panel into 12 columns
again, and nothing does. `gridToCanvas` (`web/shared/domain/canvas.js`) is
the same arithmetic kept as the documented record of it, with no caller;
the reverse direction is gone.

**Kinds** (ADR-0108; the bodies past a live pane are C3 of
`docs/plans/matrix-canvas.md` §4.2). `kind` is an open text column, so a
kind is a validator edit on both sides — `internal/store/canvas.go` and
`web/shared/domain/canvas.js` — and never a migration. The store validates
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
sides, and `parseRef` / `buildRef` (`web/shared/domain/canvas.js`) are the
only code that takes one apart or puts one together.

**Routes** (`internal/server/canvas.go`; the auth gate applies as for
every `/api/*`). Every mutation answers with the payload its event
carries, so the client has one reducer path for a response and a frame.

| Route | Answers |
|---|---|
| `GET /api/canvases` | `{canvases: [summary]}` — `id, name, compact, createdAt, updatedAt, panelCount`, by name (ASCII case-folded), then creation |
| `POST /api/canvases {name}` | 201 summary; 400 with the limit named |
| `GET /api/canvases/{id}` | 200 summary + `panels: [{id, kind, ref, x, y, w, h, createdAt}]` + `edges: [{id, aPanel, bPanel, createdAt}]` (one read on open; 500 panels ≈ 50 KB, an edge is four short strings) |
| `PATCH /api/canvases/{id} {name?, compact?, ifUpdatedAt}` | 200 summary; 400 on an empty patch or a broken rule; 409 when the row moved on |
| `PATCH /api/canvases/{id}/layout {ifUpdatedAt, panels: [{id, x, y, w, h}]}` | the changed subset, one transaction, all or nothing, every rectangle judged by the same canvas rules; 200 `{id, updatedAt, panels}`; 409 when stale |
| `POST /api/canvases/{id}/panels {kind, ref, x, y, w, h}` | the client places (`nextSlot`, phase 3), the server validates the rectangle; 201 `{id, updatedAt, panel}`; 409 on a duplicate binding |
| `DELETE /api/canvases/{id}/panels/{panelId}` | 204; the feed carries the new `updatedAt`, and the panel's edges go with it |
| `GET /api/canvases/{id}/edges` | `{edges: [{id, aPanel, bPanel, createdAt}]}`, oldest first — the audit list, which wants the edges without the panels |
| `POST /api/canvases/{id}/edges {aPanel, bPanel}` | 201 `{id, updatedAt, edge}`; the store orders the pair, so either direction is the same row; 400 names the rule (another canvas's panel, the same panel twice, a kind with no mailbox, the cap); 409 "These panels are already linked" |
| `DELETE /api/canvases/{id}/edges/{edgeId}` | 204, and the grant goes with the row; 404 when it is already gone |
| `DELETE /api/canvases/{id}` | 204; the panels and their edges cascade |

`ifUpdatedAt` may also travel as an `If-Match` header (the pins
convention); empty means no precondition. Adding or removing a panel takes
none — a picker must not fail because someone dragged.

**Limits refuse and never truncate** (400, the limit named; runes, not
bytes): 64 canvases, 500 panels per canvas, 80 characters of name (empty
or only spaces is "name is required"). A rectangle is always in **canvas
units**, one unit 8 px, and is read the same way everywhere: `w ≥ 32` units
(256 px), `h ≥ 28` units (224 px), `x` and `y` may be negative — the plane
has no origin corner and no column cap.

The plane is bounded so a panel cannot be dragged out of reach of
every viewport: `|x| ≤ 100000` and `|y| ≤ 100000` units (±800 000 px),
`w ≤ 4096` and `h ≤ 4096` units (32 768 px). The messages are the
contract: `web/shared/domain/canvas.js` repeats them (`validateName`,
`validateCompact`, `validatePlacement`, `validatePanel`)
so the UI can refuse before asking.

**Events** (durable, appended in the mutation's transaction; `touches(ev,
["canvas"])` keys on the prefix, so `canvas.panel.*` reaches the surface
with the rest):

| Event | Data |
|---|---|
| `canvas.created`, `canvas.updated` | the summary |
| `canvas.layout` | `{id, updatedAt, panels: [the subset that moved]}` |
| `canvas.panel.added` | `{id, updatedAt, panel}` |
| `canvas.panel.removed` | `{id, updatedAt, panelId}` — the panel's edges go with it, without an event each |
| `canvas.edge.added` | `{id, updatedAt, edge}` |
| `canvas.edge.removed` | `{id, updatedAt, edgeId}` |
| `canvas.deleted` | `{id}` — the panels and edges go with it, without an event each |

**Client contract** (`web/shared/domain/canvas.js`, pure): `CANVAS_LIMITS`,
`CANVAS_KINDS`, `PANE_STATES` / `hasPane`, `CANVAS_COMPACT`, `CANVAS_EVENTS`,
`UNIT_PX` (8), `EDGE_KINDS`; `normalizeCanvas`,
`normalizeCanvasList`, `normalizePanel`, `normalizeEdge`,
`normalizeEdgeList`, `normalizeCanvasDetail` drop junk
the way `contracts/appPrimitives.js` does (a panel without a whole
rectangle or a known binding is dropped); `validateEdge(edge, panels,
edges)` and `edgeEndpoints(edge, panels)` are the **Edges** section below;
`applyCanvasEvent(state, ev)`
reduces the eight events over `{ list: [summaries], byId: { id: { canvas,
panels, edges } } }` — an event for a canvas that is not loaded only touches the
summary list, a `created`/`updated` for a canvas the list never saw
inserts it (the summary is complete), the list stays sorted by name, and
junk or an unknown type returns the same object. `validatePlacement`,
`validatePanel`, `nextSlot` and `layoutDiff` take a rectangle and nothing
else: there is one unit and one set of rules, so no caller passes a mode.

Decision table (every row has a store or handler test —
`internal/store/canvas_test.go`, `internal/server/canvas_test.go`):

| Operation | Condition | Result |
|---|---|---|
| create | 64 canvases exist | 400 "limit: 64 canvases" |
| create | name empty / > 80 runes / only spaces | 400 naming the limit; never truncated |
| create | ok | 201 summary; `canvas.created` |
| add panel | same (kind, ref) already on this canvas | 409 "already on this canvas" — per kind, so one pin can be a `note` on two canvases and one id may be two kinds |
| add panel | `note` whose pin does not exist, `file`/`diff` whose owner or file does not exist | 201 — the store never asks; `bindingState` and the body are what say so |
| add panel | 500 panels exist | 400 "limit: 500 panels per canvas" |
| add panel | `w` under 32, `h` under 28, `w` or `h` over 4096, `x` or `y` outside ±100000, kind not agent/terminal/note/file/diff ("kind must be agent, terminal, note, file or diff"), ref empty | 400 naming the rule in canvas units |
| add panel | a `file` or `diff` ref that is not `<owner>:<id>:<path>`, or whose owner letter is not t/a/w | 400 naming the shape; a path with colons of its own is fine |
| add panel | the same path as a `file` and as a `diff` | 201 twice — the unique index is per (kind, ref), and reading a file and reading its changes are two things to have open at once |
| add panel | ok | 201 `{id, updatedAt, panel}`; `canvas.panel.added`; `updatedAt` bumped |
| layout patch | `ifUpdatedAt` stale | 409, nothing written |
| layout patch | a panel id not in this canvas, or a position out of bounds | 400, nothing written (all or nothing) |
| layout patch | ok subset | 200 `{id, updatedAt, panels}`; one `canvas.layout` event carrying exactly the subset |
| update | rename ok / compact ∈ {vertical, none} | 200 summary; `canvas.updated` |
| update | compact other / name over limit / stale `ifUpdatedAt` | 400 / 400 / 409 |
| add panel | negative `x`/`y` | accepted — the plane's, not a mistake |
| layout patch | mixed: one rectangle legal, one not | 400, nothing written (all or nothing) |
| remove panel | unknown panel | 404 |
| remove panel | ok | 204; `canvas.panel.removed` |
| delete canvas | ok | 204; panels gone (cascade proven by a query); `canvas.deleted` |
| any | unknown canvas id | 404 |
| agent or terminal deleted elsewhere | — | canvases and panels untouched (the panel row survives) |

## Edges (ADR-0116)

An edge is the owner's recorded intent that two sessions may exchange
messages. **It grants exactly ADR-0104's mailbox contact, and nothing
else**: an edge never lets one session read another's session file,
scrollback, buffer or history — **no transcript, ever**. The supported way
to get context out of another session is to ask it, and let it answer under
its own judgment.

**Table** (migration 044):

```
canvas_edges(id, canvas_id → canvases ON DELETE CASCADE,
             a_panel → canvas_panels ON DELETE CASCADE,
             b_panel → canvas_panels ON DELETE CASCADE,
             created_at, UNIQUE(canvas_id, a_panel, b_panel))
```

The pair is stored **ordered** (`a_panel < b_panel`, sorted before the
write), so the same edge drawn in either direction collides on the unique
index: an undirected edge is one row, and there is no direction column —
an arrow would promise a one-way restriction the mailbox cannot keep.
Endpoints are **panels**, not sessions: a panel already carries its own
`(kind, ref)`, and re-pointing a panel is a new binding that must be drawn
again. Both foreign keys cascade, so an edge cannot outlive the canvas or
either panel — the grant follows the line you can see. The unique index
leads with `canvas_id` (the per-canvas read and the cascade from
`canvases`); each endpoint has its own index, because the cascade from
`canvas_panels` and the contact union both look a panel up as an *endpoint*.
Cap: **1000 edges per canvas**, refused with the limit named.

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
| the canvas is deleted | no; the edges are gone |
| two edges in two canvases for the same pair | yes, listed once |
| the panel's target was deleted from the fleet | no — the panel survives (ADR-0108), the connection cascades with its agent or terminal |
| an edge between two panels bound to my own session | no — the caller is never in its own list |

Dropping edges is dropping one table and one union clause: the mailbox
returns to workspace scope with no migration of anything else, because
nothing was ever copied while an edge existed.

**Client contract** (`web/shared/domain/canvas.js`): `normalizeEdge` /
`normalizeEdgeList` (an id and two different panel ids, junk dropped);
`validateEdge(edge, panels, edges)` repeats the server's refusals in the
server's order and words — the two ids, two different panels, both on this
canvas, both a kind with a mailbox (`EDGE_KINDS`), then the cap and "These
panels are already linked" in either direction; `edgeEndpoints(edge,
panels)` answers the two panel rows the canvas draws between, or `null`
when an end is not on the board — the one case it must not draw.
`applyCanvasEvent` reduces `canvas.edge.added` / `canvas.edge.removed`, and
`canvas.panel.removed` drops the edges that touched the panel, mirroring the
store's cascade (which announces no event per edge).

**Drawing one** (`Plane.jsx`, `Link.jsx`, `CanvasSurface.jsx`).
React Flow's `Handle` plus `onConnect`: a connector knob in the panel
header, in the action row that already carries `nodrag`, so a drag from it
starts a connection and never moves the panel; the body keeps `nodrag
nowheel` and `.cv-head` stays the drag handle. The knob is where the gesture
*starts*; where the finished line *touches* is computed per link (*Where a
link touches*, below). It is hidden until the panel
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

**The two questions** (`canvasGrants.js`; ADR-0116 §5, *an edge never enrols
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

**Reading broken** (§7). `canvasGrants.js` joins a canvas's edges and panels
with `GET /api/communication` — whose `connections[].active` *is* the
store's `peerCurrent` — and answers `grants`, the reason, and whether the
pair crosses folders. It is the only place that decides, so the plane and
the audit list can never disagree. The four reasons are kept apart because
each has a different next action: `gone` (the session left the fleet),
`off` (never enrolled), `revoked` (the owner revoked it), `stale` (the
recorded session moved). A link that grants nothing is drawn in the warn
colour with an unlink mark, the word **Broken**, and the one line naming
which end and why — never a dotted line, which is a thing a viewer learns
to ignore. The connection list is read only when the canvas holds an edge,
refreshed on `peer.*` **and** on `agent.updated` / `terminal.updated` /
`*.deleted` (a grant can stop granting because the *owner* changed session,
which no `peer.*` row announces), debounced 400 ms — the feed, never a
timer. A failed read keeps the last answer: flipping every live link to
broken is the one direction a wrong answer is dangerous in.

**Where a link touches** (`web/shared/domain/canvasAnchors.js`, `Link.jsx`,
`Plane.jsx`). The line is a **floating edge**: each end is computed on the
border of its panel that faces the other panel, never read off a fixed
handle. Until 2026-09-11 the plane declared one `Handle` per panel at
`Position.Top` and drew with `getSimpleBezierPath`, whose control points only
distinguish the horizontal sides — "top" was never honoured, so the line
touched wherever the curve happened to land (a capture of the shipped build
has it leaving one panel near its bottom-right corner and entering the
other's top edge, left of centre), and every link a panel carried stacked on
that one point whatever direction its target lay in.

Two rules, pure functions of two rectangles and a count, so they are proved
in `canvasAnchors.test.js` and not in a browser:

1. **Direction** (`facingSide`, `borderPoint`). The anchor is where the ray
   from this panel's centre to the other's leaves this panel's border. The
   comparison is `|Δx|·h` against `|Δy|·w`, which measures the ray against
   the rectangle's own half-extents: the diagonal that switches sides is the
   panel's corner, not a fixed 45°, so a wide panel hands a target 45° away
   to its top or bottom and a tall one to its left or right. A panel to the
   right is joined right-edge to left-edge, one below bottom to top, and a
   link can never cross the panel it belongs to. The point is clamped
   `ANCHOR_MARGIN` (14 px) off both corners — a link touching the corner
   reads as touching neither side, and the 8 px rounded border has no line
   there.
2. **Spread** (`spreadAlong`). Rule 1 runs per link, so two targets in
   nearly the same direction want nearly the same point. Anchors sharing one
   border are pushed apart to `ANCHOR_GAP` (18 px) **and no further**: sort
   by the wanted position (that ordering is the order of the targets along
   the border, which is what keeps two links from crossing on their way
   out), subtract `i·gap` so the min-gap constraint becomes "non-decreasing",
   take the nearest non-decreasing sequence by pool-adjacent-violators, add
   `i·gap` back, then slide the block inside the border if it overhangs. Two
   anchors already far enough apart do not move at all — which is the
   property that matters, because it is what lets each link keep pointing at
   its own target. A side that cannot hold them at 18 px shares itself
   evenly; the gap shrinks rather than the anchors leaving the panel.

**The gesture is unchanged.** The connector in the header is still a real
`Handle` — a connection cannot start without one, and the knob is what the
owner drags. It is no longer where the line attaches: the drawn edge floats
independently of it, and `position` on that handle is only the side React
Flow measures the knob against. The **preview follows the same rule**, via
`connectionLineComponent` (`ConnectionPreview` in `Link.jsx`): the pointer
stands in for the far rectangle as a zero-sized one, so the fixed end picks
the border facing the pointer, and once the pointer is over a panel the drop
can land on, the far end jumps to *that* panel's facing border — the line
you drag is the line you get. The one thing the preview cannot show is the
spread, because the link being drawn is not in the set yet; a drop onto a
side that already carries links moves the line by up to one gap on release.

**The curve is ours** (`linkPath`), not React Flow's, for one reason: the
chip needs the curve's own centre and its normal *at that centre*, and only
whoever placed the control points can answer that. Each control point sits
along its border's outward normal, half the facing distance out and never
less than `CURVE_MIN` (24 px), so the line leaves and enters at a right
angle — which is what makes it read as attached to the two cards rather than
ruled across them — and the shape is symmetric, so swapping the ends (which
the store's sorted pair does arbitrarily) draws the same curve. `labelX` /
`labelY` are B(0.5); the chip's normal is ¾·[(P2+P3) − (P0+P1)] turned
perpendicular, because a normal taken from the chord would slide the chip
along the curve instead of off it. Still no arrowhead — the mailbox is
symmetric (§6).

The straight line between two **aligned** panels did not go away, and it
should not: the shortest tie between two facing borders at the same height
is a straight segment. What went away is the defect behind it — that segment
is now a short perpendicular connector in the gap between the two panels,
not a diagonal rule drawn from one header to the other across whatever sat
in between.

**What it costs.** `anchorLinks` runs in `Plane.jsx` over `nodes`, so it
recomputes on every frame in which a linked panel moves — a drag, a resize,
not a pan (the camera is a CSS transform and the node positions do not
change; a 2 s pan at 0.4 and at 1.0 produced exactly one distinct edge-path
state, and 60.5 fps). Only panels that carry a link are collected, so a
plane of 200 panels and 4 links pays for 4. Measured on a scratch instance,
26 panels and 20 links, 2 s gestures driven over CDP at 60 Hz:

| gesture | zoom | fps |
|---|---|---|
| idle (ceiling) | — | 60.0 |
| drag a panel with 0 links | 1.0 | 60.1 |
| drag a panel with 1 link | 1.0 | 60.1 |
| drag a panel with 4 links | 1.0 | 60.1 (59 distinct edge states / 121 frames) |
| drag a panel with 1 link | 0.6 | 60.1 |
| drag a panel with 4 links | 0.5 | 59.8 |
| pan, all 20 links mounted | 1.0 / 0.4 | 60.5 |

The pass itself, benchmarked directly: **0.017 ms** for this board,
**0.147 ms** at 200 links, **0.743 ms** at the 1000-link cap — 4 % of a
16.7 ms frame at a board no canvas has yet. If that ever bites, the fix is
not a cheaper rule but a narrower one: recompute only the links whose two
panels changed, which the memo already has the information to do.

There is no parallel-edge separation to keep: `validateEdge`, the plane's
`isValidConnection` and the store's unique ordered pair all refuse a second
link between the same pair. The spread is between links to *different*
panels; if a second edge per pair is ever allowed, an index-varied curvature
is where it goes.

**The band a pointer has to find** is React Flow's second, transparent path
at the edge's `interactionWidth` (22). That width is in SVG user units —
plane pixels — so it used to thin with the camera: 8.8 screen px at zoom
0.4 and 4.4 at the 0.2 floor. `canvas.css` gives that path
`non-scaling-stroke`, the same SVG idiom the minimap uses, so the band is
**22 screen px at every zoom** while the painted line keeps scaling. It
does not fix C4's other finding — two adjacent panels can still hide their
whole link behind themselves — which is what the header's link chip is for.

**Where the chip sits**, and why it is the way it is (the browser pass,
below, is the evidence). React Flow draws edge labels *under* the nodes and
the midpoint of a line between two headers lands on a panel more often than
not, so the chip is lifted above the nodes and pushed off the line by half
its own size plus ten screen pixels along the **curve's** normal — its
`labelX` / `labelY` come from the same function that drew the path, and the
derivative of that cubic at its centre is (2·Δx, Δy), not the chord, so
pushing along the chord's normal would slide the chip along the curve
instead of off it. Forced to one side, because the arithmetic's side
follows the stored pair order, which is the panel ids sorted and nothing a
viewer can see. A link that **grants**
shows its chip only while it is asked for (hover or selection, with a grace
period so the pointer can cross the gap to it); a link that **grants
nothing** shows it always. Hovering the chip — not only selecting the line —
is what reveals the reason, because two panels close together hide their
whole link behind themselves and then the line has no hittable pixel.

**The header's link chip** (`linkCounts` / `linkChipTitle` in
`canvasGrants.js`, drawn by `PanelHead`). The plane draws the lines, but a
line is not always readable *as* a line: it can be off the camera, run
entirely behind the two panels it joins (the C4 pass found a pair with no
hittable pixel at all), or be a few pixels long when zoomed out. So a panel
that carries links says so on its own header — a count, with an unlink mark
and the warn colour when any of them grants nothing — wherever the camera
is, and pressing it opens the audit list below, which is where a link is
read and revoked. `linkCounts(edges, panels, peers, names)` answers
`{count, broken}` per panel from the same join everything else reads, so the
chip can never disagree with the line; `linkChipTitle` writes the words,
because a bare number teaches nothing. A panel with no link has no chip.

**The audit list** (`web/desktop/src/components/GrantedContacts.jsx`,
heading **Granted contacts**), the non-spatial place ADR-0116's Consequences
ask for by name. It is **Messages' own section**, not the Canvas reaching
outside its app: the grant lives in `peer_connections` and decides who the
mailbox lets talk, and a canvas is only where a human drew it — the door
ADR-0109's 2026-09-11 amendment declares, on the condition that it imports
nothing from `components/canvas/`. It lists **every** live edge: both ends by
name and kind, the canvas it lives on, whether it grants right now, and a
Remove that calls the same `DELETE` the canvas does. It reuses the
participants list's row shape — "who may reach whom" should read the same
whether the pairing came from a workspace or from a line someone drew — and
it is deliberately **not** scoped to the workspace picker above it, because
an edge is per pair and half of what it is for is pairing two folders.
Reads: the canvas list plus one detail per canvas (which already carries
`edges` and `panels`) and the connection list, refreshed by the same feed
rows, debounced.

Decision table (every row has a store test in
`internal/store/canvas_edges_test.go` and, where it is an HTTP answer, a
handler test in `internal/server/canvas_test.go`):

| Operation | Condition | Result |
|---|---|---|
| add edge | two `agent`/`terminal` panels of this canvas | 201 `{id, updatedAt, edge}`; `canvas.edge.added`; the pair stored ordered |
| add edge | the same pair drawn backwards | 409 "These panels are already linked"; still one row, one event |
| add edge | `aPanel == bPanel` | 400 "an edge needs two different panels" |
| add edge | either id empty | 400 "aPanel and bPanel are required" |
| add edge | a panel of another canvas, or one that does not exist | 400 "panel ⟨id⟩ is not on this canvas" |
| add edge | a `note`, `file` or `diff` panel | 400 "panel ⟨id⟩ is a ⟨kind⟩ panel and has no mailbox: an edge links agent or terminal panels" |
| add edge | 1000 edges exist | 400 "limit: 1000 edges per canvas" |
| add edge | unknown canvas | 404 |
| remove edge | ok | 204; `canvas.edge.removed`; the grant goes with the row |
| remove edge | already gone | 404 "edge not found" |
| remove panel | the panel had edges | they cascade; the feed carries only `canvas.panel.removed` |
| delete canvas | — | the edges go with it, without an event each |
| MCP | any tool | no verb creates, lists or implies an edge |

## Surface (phase 3)

The app (`internal/apps/canvas.go`: id `canvas`, icon `canvas`, `surface:
"native"`, no badge — ADR-0109) is registered by the desktop as `canvas`
in `web/desktop/src/lib/nativeApps.js` and mounts
`web/desktop/src/components/canvas/CanvasSurface.jsx` with `host` plus
`initialPath` / `onPathChange`, so `#/app/canvas/<canvasId>` opens that
canvas and switching updates the hash. The phone lists the tile as
*Desktop only*.

**The surface is a chunk of its own** (ADR-0118's Consequences). `App.jsx`
lazy-imports it and the tab mount wraps every native surface in one
`Suspense` with a `null` fallback, so a reader who never opens Canvas
carries none of it: the desktop's main chunk dropped 15 376 B of JS and
2 680 B of CSS gzip when it moved. React Flow is lazy again *inside* it
(**Canvas** below), so opening the app and opening a plane are two fetches.

**The old links still work** (ADR-0118 §4). `#/app/matrix` and
`#/app/matrix/<id>` are replaced — never pushed — with the canvas route,
path and all, and an `x:matrix` restored from `localStorage` is rewritten to
`x:canvas` on the way out of `readOpenTabs`. Both read one map,
`RENAMED_APPS` in `web/desktop/src/lib/routes.js` (`renamedAppId` /
`renamedAppHash` / `renamedTabId`), because a bookmark and a saved tab strip
disagreeing is exactly what a second mechanism would eventually do. The
redirect is its own effect declared before the one that resolves a hash into
a tab, and that one returns early on a renamed hash, so the old id never
reaches the *That app is gone.* branch. The two per-viewer keys are
**migrated on first read**, not left to lapse: `picode-matrix-last` →
`picode-canvas-last` (`readLast`) and `picode-matrix-view:<id>` →
`picode-canvas-view:<id>` (`readView`), old key deleted. A rename nobody
asked for must not cost a reader the canvas they had open or the camera they
parked it at. The **API** keeps no compatibility layer: its only clients ship
in this binary.

| File | Holds |
|---|---|
| `CanvasSurface.jsx` | the stage the host fills edge to edge, the top-left chrome cluster floating on it (**Chrome** below), the empty states, the store `{ list, byId }` reduced by `applyCanvasEvent`, the save flow, "last canvas opened" in `localStorage` `picode-canvas-last` |
| `Plane.jsx` | the plane's host: `@xyflow/react` 12.11.6, lazy-imported, one custom node type rendering the same `Panel`; the camera, the zoom rule and the minimap. Not `CanvasCanvas.jsx`: it is named for what it is. **Canvas** below |
| `PanelStill.jsx`, `stills.js` | the two bodies a canvas panel has instead of a live pane — the text still and the name-plate — and the memory-only map of captured screens |
| `Panel.jsx` | the wrapper every panel keeps (and `PanelHead`, which the maximize layer reuses): face, name, hint, the sidebar's chip (`agentRowStatus` / `terminalStatus`), the link chip, Open · Maximize · Remove from canvas; it observes its own visibility; two memo layers so a re-render never touches a body |
| `AgentChatPanel.jsx`, `web/desktop/src/hooks/useAgentSocket.js` | a managed agent's body (phase 4): one `/ws/agent?agent=` per mount driving the desktop's own `lib/agentEvents.js` reducer and reconciling `…/sessions/transcript?agent=&tail=200`, rendered by the tab's `Conversation` in `readOnly` mode. The **Chat body** section below |
| `PanelBody.jsx`, `TerminalPanel.jsx`, `NotePanel.jsx`, `FilePanel.jsx`, `DiffPanel.jsx` | `PanelBody` routes by **kind**: a terminal or an agent is `TermSurface` (the tab's engine; a terminal first `POST /api/terminals/{id}/open`s for its live record), a note is `NotePanel` — one `GET /api/pins/{id}` per mount, `react-markdown` + `remarkGfm` over the app's `.md` styles, refetched on that pin's `pin.updated` (it subscribes to the feed itself, as the Inspector does for `git.updated`) — a file is `FilePanel`, which is `FilePane` with `variant="embedded"` plus the two things the canvas adds (`docKey`, `onDirty`) — a diff is `DiffPanel`, `WorkingDiff` under a nonce the feed bumps. Unloaded → one muted line with the feed's last state; the rows below as one line + one action |
| `PanelFace.jsx` | the mark that says what a panel is bound to, in the header and on the name-plate: a provider face, a CLI badge, or the pin mark |
| `PanelPicker.jsx`, `NameDialog.jsx` | cmdk list **grouped by kind** — Agents, Terminals, Pins, Open files, Changes to an open file — of what is not yet on this canvas, with the sidebar's faces and words; a group with nothing in it says so under the list with its one action (*No pins yet.* — New pin); the name form (`canvasNameSchema`, Zod, the store's messages, `noValidate`) |
| `chunkLoader.js`, `paneOwnership.js` | the `IntersectionObserver` glue over the pure `loadPolicy`; what unloading does to an attach |
| `Link.jsx`, `web/desktop/src/components/GrantedContacts.jsx` | how one edge draws on the plane, and Messages' **Granted contacts** audit list — the **Edges** section above |
| `web/desktop/src/lib/fileDocs.js` | the open documents of `file` panels, keyed by ref and held outside React — `paneOwnership.js` for an editor, so maximizing a panel keeps unsaved text |
| `web/shared/domain/canvas.js` | `nextSlot`, `layoutDiff`, `parseRef` / `buildRef` / `validateRef`, `refOwner`, `gitTouches`, `bindingState`, `hasPane`, `loadPolicy` (with the zoom and the per-row `pane`), `zoomBody`, `pointerAtZoom`, `unitsToPx` / `pxToUnits`, `tidyCanvas`, `normalizeViewport`, `suspendedToDispose`, `hasChat` / `CHAT_STATES` / `CHAT_LIVE_MAX` / `chatBudget`, `panelOrder`, `neighborPanel` (+ `PANEL_DEFAULT_CANVAS` 32×42, `CANVAS_ZOOM`, `PANEL_DIRECTIONS`) — one test per row below in `canvas.test.js` |

**The fleet decides what a panel is bound to**, and a note's pin follows the
same rule from the surface's own list: `pins` is `null` until
`GET /api/pins` answers, and `bindingState` reads a missing list as "not
read yet", never as gone. That list is read **only when something needs
it** — the canvas holds a note, or the picker is open — then kept current by
`pin.*` on the feed and a stale reveal; a canvas of terminals costs no pin
request at all. It carries summaries, never bodies (`docs/architecture/pins.md`),
which is why the body reads its own pin. The fleet's own rule: the surface
draws the skeleton until the desktop says it has one: `host.fleet.loaded` is the
App's `bootstrapped`, and until it turns true an empty fleet means "not
read yet", not "deleted". Without it every terminal panel showed *That
terminal is gone.* for the length of the boot fetch — 15 s on a fixture
with 45 terminals (each row carries git state). A host that passes no
`loaded` is taken at its word.

**Reads.** `GET /api/canvases` and `GET /api/canvases/{id}` once on open
and on a reveal older than 10 s; then the feed only: `canvas.*` through
`applyCanvasEvent`, agents and terminals through the host's fleet,
`agent.tui` for the Working chip after one `GET /api/tui-working` for the
agents on the canvas, `GET /api/pins` when a note or the picker needs it and
again on `pin.*`, and one `gitdiff` read per `diff` panel, again on a
`git.updated` for its owner's folder. No timer.

**Binding × fleet** (`bindingState`, plan §4.4):

| Row | Body when loaded | Actions |
|---|---|---|
| `terminal-running` (also a plain shell whose tmux died: opening revives it) | live xterm | Open · Maximize · Remove |
| `terminal-stopped` (a CLI terminal, `launchCli` + `running: false`) | `TermSurface`'s own stopped state — Resume last session / Start from Agent CLIs | same |
| `terminal-gone` | "That terminal is gone." | Remove |
| `agent-interactive` | live TUI | Open · Maximize · Remove |
| `agent-managed` | the agent's **live conversation, read-only** — `AgentChatPanel`, the **Chat body** section below. Not live (unloaded, over the cap, or a name-plate): one line and **Open** | Open · Maximize · Remove |
| `agent-stopped` | "Agent is stopped." — **Run** (`POST /api/agents/{id}/open`; the feed flips the mode and the body swaps in place) | Open · Remove |
| `agent-gone` | "That agent is gone." | Remove |
| `note-ready` | the pin's markdown, read-only | Open in Pin Studio · Maximize · Remove |
| `note-gone` | "That pin is gone." | Remove |
| `file-ready` | `FilePane`, embedded: the editor, its Preview/Raw toggle and Save. The chip says **Unsaved** while it is dirty | Open in its own tab · Maximize · Remove |
| `file-gone` | "Where this file was read from is gone." | Remove |
| `diff-ready` | `WorkingDiff` for that path, refetched on `git.updated` for the owner's folder (`gitTouches`) — its own header has **Open file** | View this diff in a tab · Maximize · Remove |
| `diff-gone` | "Where this file was read from is gone." | Remove |

**Chunk loading** (`loadPolicy`, plan §4.5). Observer root: the React Flow
pane, `rootMargin: 100%` on both axes, because a plane pans sideways as
well as down — C0 measured 49 panels in the band against 28 for a band that
only grew vertically.

| Situation | Behaviour |
|---|---|
| wrapper near the viewport for 300 ms | body mounts |
| leaves before 300 ms | the pending load is dropped |
| loaded, then leaves | body unmounts after 5 s; coming back sooner cancels |
| dragged, resized, focused or maximized | pinned: never unloads |
| canvas tab hidden | every panel reads as far and unpinned: all unload after 5 s, the focused one too |

**A wrapper that unmounts stays loaded for one tick.** `unobserve` does not
drop the loaded flag; it arms a zero-delay timer and drops it only if the id
has not registered a new element by then. A body moves host inside a single
commit — maximize unmounts the wrapper's body and mounts the layer's,
restore does the reverse — and dropping the flag synchronously would turn
every such move into an unload: a socket suspend and a kick per panel, for a
pane the viewer is looking at. It is the loader's half of the rule
`paneOwnership.js` keeps for the pane itself (`mounted` counted, never
guessed), and the two together are why maximizing a live terminal is the
same xterm, refitted, with the socket open in both directions.

**Unsaved work is never unmounted silently.** A `file` body reports its
dirty bit up (`FilePane`'s `onDirty`) and the surface `keep`s that panel in
the loader: `keep` is the `pinned` set a drag and the focus already use, with
one difference — it survives a hidden canvas, because a hidden tab throwing
away a draft is exactly the silent unmount the rule exists to stop. The chip
says **Unsaved** while it holds. The text itself outlives a remount either
way: the document lives in `lib/fileDocs.js` keyed by the panel's ref, so
maximizing a panel moves the body between hosts
without losing it, the same way `paneOwnership.js` keeps one xterm across
hosts. Removing a dirty panel asks first (Undo restores the panel, never the
text) and is the one thing that forgets the document.

**Where a file or a diff panel comes from.** The picker must not become a
file manager, so it offers **only what is already open as a file tab** — a
path this browser already has, read from `host.openTabs` and turned into a
ref by `buildRef` — once as *Open files* and once as *Changes to an open
file*, because the same path as a `file` and as a `diff` is two panels. Everything else stays a debt on purpose: an *Add to canvas* row
on an Inspector change or an entry in a file tab's own menu would need the
desktop to know which canvas is open and where a panel would land, and that
state lives inside the Canvas app's surface — apps read the desktop through
`host`, never the other way round (ADR-0109). Adding it means giving the
desktop a canvas client of its own, which is a decision, not a line.

### Chrome (2026-09-11)

**The surface has no header.** Until now it wore the app bar every other
surface wears — icon, title, the switcher, Add panel, New canvas, a `⋯`
menu and Close — and on a plane that bar was a strip of the work area spent
on things said twice: the tab strip already names the app, and the tab's own
× already closes it. The plane now fills the stage edge to edge and the
chrome floats **on** it, adapted from nodeterm's canvas
([`docs/benchmarks/2026-09-10-node-canvas.md`](../benchmarks/2026-09-10-node-canvas.md),
"Chrome"; the owner's ask was "menus estilo overlays e não aquela barra fixa
no topo"). Two clusters, and nothing else:

| Cluster | Holds | Why there |
|---|---|---|
| top-left (`.cv-chrome`, `CanvasSurface.jsx`) | the switcher (a label or a `<select>` — below), **Add panel**, and a `⋯` menu: **New canvas**, **Tidy panels** (only with panels), **Rename**, **Delete canvas**, then **Close tab** | the two things a reader reaches for constantly are *which canvas* and *one more panel*; everything else is one press away |
| bottom-left (`.cv-zoom`, `Plane.jsx`) | a **column**: zoom in / zoom out / the 100 % readout / **Fit** | React Flow's own `<Controls>` stands here (owner, 2026-09-12); a plane has least to say at its bottom-left |
| bottom-right (`<MiniMap>`, `Plane.jsx`) | the map of the plane | opposite the camera controls rather than stacked over them, so neither corner carries both |

Both are the same `.cv-cluster`, one turned: a row at `--ctl-h` top-left and
a `--ctl-h`-wide column bottom-left, segmented by hairlines, on
`--bg-elevated` with a `--border` hairline and the minimap's shadow. A
column carries no `data-align-row` — that attribute asserts the children
share a top edge (`web/shared/domain/overlayAudit.js`), which stacked ones
do not. **Fit** is the icon `Maximize2` there and not the word: the word was
the one child that set the column's width. **Opaque, never translucent and never blurred** — a cluster has to
read over a dark plane, a light plane and a live terminal parked underneath,
and a scrim over a terminal is the one case where a reader sees the text
through their own chrome.

**What moved, and what it cost.** The icon, the `<h2>` title and **Close**
left: the first two are the tab's, and the third is the tab's ×. **New
canvas** left the bar for the menu, because a reader creates one canvas and
then works on it for days. **Close tab** is in the menu and not on the
plane: a floating Close would be chrome competing with chrome, and the menu
keeps a keyboard path to it for a reader who never reaches the strip.
**Add panel** stayed out of the menu because it is the verb the surface
exists for. The empty states keep their own one line and one action — *No
canvas yet.* → New canvas (the cluster is not drawn: there is nothing to
switch), and, for a canvas with no panels, the centred card below.

#### The switcher: a label with one canvas, a select with two (2026-09-11)

The owner opened the top-left `<select>`, found a single option already
chosen, and asked what it was for. A control that offers no choice is a
promise of one, so it is no longer a control:

| Canvases | What is drawn | Behaviour |
|---|---|---|
| 1 | `<span class="cv-switch cv-switch-one">` — the canvas name | a label: no chevron, no tab stop, no hover highlight, `title` for a name too long for the box |
| 2 or more | `<select class="cv-switch cv-select">` — every canvas | unchanged: pick one and the surface switches |

**Creating another stays in the `⋯` menu** either way, which is what turns
the label back into a select. **The geometry does not move across the
switch**: both shapes are `.cv-switch` — the same `--ctl-h`, the same 10 px
leading padding and the same 104 px floor — and only the select adds the
16 px a chevron needs. A short name sits on the floor in both, so the cluster
is pixel-identical before and after the second canvas exists.

**The name stays reachable without being focusable.** A label is not a tab
stop — a stop that leads nowhere is the same lie in the keyboard as the
chevron was in the pointer — so it is read as the group's content, and the
`⋯` button beside it carries the name in its own accessible name ("More
actions for *<name>*"). A keyboard reader still hears which canvas they are
on, on the way to the only actions there are.

#### An empty canvas (2026-09-11)

A canvas with no panels **keeps its plane** and floats one line and one
action in the middle of it (`.cv-blank`, `CanvasSurface.jsx`):

> A panel is one agent, terminal, note or file, live on this canvas.
> **[+ Add panel]**

Until now the empty canvas *replaced* the plane with the page-level
blankslate every list uses. Two things were wrong with that: the reader lost
the ground they had chosen (the plane, its texture, the minimap and the zoom
cluster all disappeared until the first panel existed), and the first panel
remounted React Flow — a plane that fades in under the panel that was just
added, instead of a panel landing on a plane. The layer is
**pointer-transparent** apart from the card itself, so the plane behind it
still pans and zooms; it sits at `z-index: 5`, below the floating clusters'
6, so the two never fight; and it is gone the frame the first panel lands,
including the optimistic pending one. The card wears the cluster's own make —
opaque, `--bg-elevated`, one `--border-strong` hairline, the same shadow —
for the cluster's own reason: it has to read in both themes over dots, a
grid, a cross, or plain ground, and a translucent panel over a texture does
not. **The minimap goes with the panels**: with none on the plane it is not
drawn (`Plane.jsx`), because a map of nothing is a box of nothing in the
corner of a plane that is already saying, in its middle, that it is empty.
The zoom cluster stays — it is a control row, not an empty frame.

#### The library's mark (2026-09-11)

React Flow's attribution badge is **not drawn**: `proOptions={{
hideAttribution: true }}` on the `<ReactFlow>` element (`Plane.jsx`). The
owner asked for it hidden. `@xyflow/react` is **MIT**, which permits removing
the mark; **the licence is unchanged** — the dependency is still MIT, its
licence text still ships in `node_modules`, and this document still names the
library and its version wherever it explains the plane. Nothing was removed
from the repository: `NOTICE` carries our own required notice and has never
listed dependencies, so there is no house list for React Flow to be added to.

This reverses the reading in
[`docs/handoff/2026-09-10-matrix-canvas-surface.md`](../handoff/2026-09-10-matrix-canvas-surface.md)
— "removing it is a licence question, not a styling one". For an MIT
dependency it is not a licence question at all. The badge's own CSS
(`--xy-attribution-*` and the two `.react-flow__attribution` rules, including
the one that hid it under the maximize layer) went with it.

**A maximized panel takes the clusters with it.** They belong to the plane,
not to the surface, and the plane is what the layer covers; leaving them
would be a switcher floating over a body that is not on the canvas any more.
They are *removed*, not merely covered, so `Tab` cannot reach a control
nobody can see. `Esc` restores, and the layer's own header carries Restore.
(The `.cv-body[inert]` rule that used to hide React Flow's badge under this
layer went with the badge itself — see *The library's mark* above.)

**Fit reserves the corners.** `fitView` centres the content inside its
padded rectangle, so a symmetric padding put the first panel's header under
the top-left cluster on the very first open. The fit's padding is the
cluster geometry instead — 60 px at the top, 204 px at the bottom, 24 px at
the sides, shrunk together if a short pane cannot spare a third of its
height — and both the first-open fit and the **Fit** button use it. A reader
can still drag a panel under a cluster; what this fixes is the one camera
the surface chooses for them.

**Keyboard.** Both clusters are declared **before** the plane in the DOM
even though both are positioned, so `Tab` off the tab strip runs switcher →
Add panel → `⋯` → zoom out → 100 % → zoom in → Fit and only then reaches the
canvas's roving panel. Every control keeps the focus ring; it is drawn
`outline-offset: -2px` because a cluster clips its own segments.

### Background (2026-09-11)

**The plane's texture is the reader's choice.** The plane is the one surface
in the app that is mostly empty ground, so what that ground looks like is a
reading preference rather than a design constant: a grid helps someone
lining panels up, dots stay out of the way, plain is for a reader who finds
any texture noise. Four options, in the canvas's own **`⋯` → Background**
submenu, as a radio group of four rows:

| Option | What is drawn | Geometry |
|---|---|---|
| **Plain** | nothing — no `<Background>` is rendered at all | — |
| **Dots** (default) | React Flow `BackgroundVariant.Dots` | `gap` 32 px (four 8 px cells), `size` 2 |
| **Grid** | `BackgroundVariant.Lines` | `gap` 32 px, `lineWidth` 1 |
| **Cross** | `BackgroundVariant.Cross` | `gap` 32 px, `size` 6 |

The gap is four cells for all three, so changing texture never moves a panel
or changes what snapping means. Dots is the default because it is what the
plane shipped with: a reader who never opens the preference sees no change.

**The value** lives in `web/shared/domain/canvasPattern.js`, shaped exactly
like `theme.js`: `localStorage` key **`picode-canvas-pattern`**, a value
outside the four (junk, empty, a blocked store) reading as the default, and
a `picode-canvas-pattern` window event on write so every open plane and the
preferences page re-read without a reload. `Plane.jsx` holds it in state and
stamps `data-bg` on `.cv-canvas-flow`; the preference page is the only
writer.

**The colour is CSS, not React, and it has a token of its own.**
`canvas.css` keys `--xy-background-pattern-color` off that `data-bg`, so
switching theme re-tints the ground on the next paint — no reload, no
observer, no prop. All three variants take **`--canvas-pattern`**
(`web/shared/tokens/theme.css`), and the single token is the point of it.
They used to take the two border tokens, which is the wrong family: a
hairline's job is to be almost invisible between two surfaces, a texture's
is to be read across a whole empty plane. Measured against the plane
(`--bg-base`): `--border` was **1.16:1** on the light ground and **1.23:1**
on the dark one, and the split handed the weakest of the two to **Grid** —
the densest variant was the one nobody could see. `--canvas-pattern` is
`#a9b3c6` on light (**1.88:1** on `#f0f2f7`) and `#4a4a58` on dark
(**2.21:1** on `#0e0e11`): texture at a glance, and far enough from
`--text-secondary` (4.9:1 on dark) that it can never read as a foreground
mark. One geometry change came with it, and only one: the **dot** is 2 px
rather than 1. Dots is the sparsest variant and the default, and at the new
weight the grid and the cross read on both planes while a single-pixel dot
every four cells still did not. The **gap** is the number that must never
move — four 8 px cells, which is what panels snap to — so the mark grew
instead.

**The control is the `⋯` menu's Background submenu, and it is the only
writer.** It was a four-card group in Preferences → Appearance until
2026-09-11, with this menu carrying a `Background…` item that only navigated
there — an app's control inside PiCode's own chrome, which is the leak
ADR-0109's 2026-09-11 amendment closes. It is now a Radix
`DropdownMenu.Sub` off the same `⋯` menu the canvas's other settings live
in: four `RadioItem` rows, each with a real sample of its pattern and a tick
on the current one, and `onSelect` prevented so the rows stay open while you
pick — the plane behind the menu is the preview. `CanvasSurface.jsx` holds
the value in state, writes it through `persistCanvasPattern` and listens for
the same `picode-canvas-pattern` event the planes do. Four values are a
menu, never a second dialog.

**What did not move is the value.** `web/shared/domain/canvasPattern.js`
keeps its key and its shape: the preference is per viewer and the app owns
it, so only the *control* changed places. Preferences carries nothing about
a canvas — its Appearance section is the theme and nothing else.

**Scope: the canvas plane, and nothing else.** The rest of the app gets the
theme's ground (`web/shared/tokens/theme.css`), never a texture. This is not
an unfinished app-wide feature — a texture under a file tree, a transcript
or a git graph is noise behind content, and those surfaces are content.

**The rows show the pattern, not a word.**
`web/desktop/src/components/canvas/PatternSwatch.jsx` draws React Flow's own
geometry with the plane's own tokens (`--canvas-pattern` over `--bg-base`),
so a row cannot drift from what the plane draws. The one deliberate
difference is density: the swatch tiles every 8 px against the plane's 32,
because the sample is a 22 px square and at the plane's spacing a **Cross**
row would hold no mark at all and read as **Plain**. It lives inside
`components/canvas/` because it is the app's, not the host's.

### Chat body (phase 4)

A managed agent's panel is its **live conversation, read-only** — the reason
the Canvas shows agents at all rather than only terminals. It is the tab's
own `Conversation` (same turns, tool cards, markdown and diff lines) given
the panel's height: one scroller — `.conversation` is `position: absolute;
inset: 0` in a tab, so a panel makes it a flex row instead and there is
never a second scrollbar — with `--chat-gutter` and `--chat-col` narrowed,
the tab's 280 px composer floor removed, and the body at the panel's own
density rather than the tab's 16 px reading column. Sticky to the bottom
the way the tab is, with two corrections a panel needs: `stuckToBottom`
takes a pad (the tab's 320 px is its composer floating over the list; a
panel that is 365 px tall uses `PANEL_PAD` 64, or every position would read
as "the bottom" and yank a reader who scrolled up), and a `ResizeObserver`
on the column re-pins while the reader is at the bottom, because content
that grows after the commit — a code block measuring itself, KaTeX, the
needs-you bar taking 45 px out of the scroller — fires no scroll event. Everything that *writes* is left out —
no composer, no queue Edit/Remove, no ask form, no snippet Run — because the
one way to answer is **Open**, which is the agent's tab. Nothing is
disabled-with-a-tooltip: a control that cannot act is not drawn.
`Conversation` gained exactly two props for this: `readOnly` and `id` (the
tab keeps `conversation`; a canvas may hold several at once, so a panel
passes null and no id repeats).

**The socket is per mount** (`web/desktop/src/hooks/useAgentSocket.js`, the
desktop port of the phone's hook): one `/ws/agent?agent=<id>`, the desktop's
own pure reducer (`lib/agentEvents.js` — not a third copy), and the
transcript window reconciled on every `snapshot` and `agent_settled`
(`reconcileTranscript` / `liveSince` / `transcriptGate`). The workspace id
comes from the panel's model, because the transcript route refuses an agent
that is not in the workspace it names. Of the reducer's effects only
`scroll` is executed: `replyUI` would make a reader answer, and `toast`
would let N panels shout one agent's error — which is already an `alert`
item in the conversation the panel renders, and the selected agent's tab is
what toasts. The composer verbs (`send`, `abort`, `replyAsk`, `runBash`)
are deliberately **not** ported; phase 5's quick reply line is where they
land, verbatim from the mobile hook.

**Two sockets for one agent are fine** — the server's `Hub.Subscribe` hands
every connection its own buffered channel and `Broadcast` writes to all of
them. Measured: a second socket opened beside a panel's received all 31
events of a turn (snapshot, 21 `message_update`, settle) while the panel
rendered the same turn. In one page the App-level socket and a panel's
*hand off* rather than coexist for long — `App.jsx` closes its socket when
the agent's surface stops being selected, and a hidden canvas unloads its
panels after the 5 s hysteresis — so the overlap is the few seconds of a tab
switch, which is enough to prove neither drops nor duplicates. One
consequence is worth knowing: `Hub.Len()` is the server's *is anybody
watching* gate (ADR-0037), so a live chat panel suppresses the unobserved
`result` inbox item and the needs-you push exactly as an open tab does. That
is the same rule, not a new one — the conversation really is on screen — and
it holds only while the panel is live, because a hidden canvas holds no
sockets of its own.

**What a chat body costs, and the rule that follows** (`hasChat`,
`loadPolicy`, one test per row in `canvas.test.js`). There are three costs,
not two. A pane is an xterm plus a tmux attach and is the only body with a
**cell**, so the zoom's pointer and still rules bind it alone. A note, a
file or a diff is one fetch and some DOM. A chat is neither: no cell, so it
never goes still — but a WebSocket and a transcript, so leaving it mounted
where it cannot be read is waste. **Its socket is open exactly while its
body reads `live`**; no component holds an `if` about it.

| Condition | Chat body |
|---|---|
| in the band, `zoom ≥ 0.4` | **live**: socket open, conversation rendered |
| `zoom < 0.4` | **name-plate**, socket closed — a plate cannot show a conversation |
| outside the band, any zoom | **unmounted** and socket closed, after the same 5 s hysteresis the panes get; loading again reconnects and refetches the tail |
| more than `CHAT_LIVE_MAX` live in the band | **quiet**: the panels that entered the band most recently keep the sockets, the rest say *Paused — 12 conversations are already live.* + Open |
| needs-you while not live | the **header** still says it — the chip is the fleet's (`agentRowStatus`), never this socket's, which is what lets a zoomed-out canvas tell you which agent is blocked |

**The cap is 12** (`CHAT_LIVE_MAX`), and it is an LRU on the moment a panel
entered the band — `suspendedToDispose`'s shape, applied to sockets instead
of xterm instances. The order is fixed while a panel stays in the band, so
nothing thrashes; a panel that leaves frees its slot through the same 5 s
hysteresis, so scrolling back up returns the sockets to where the reader is.
Measured on a scratch at 1600 × 1100 with **twenty** managed agents (twenty
live `pi` processes) on one board of 4×14 grid panels, before ADR-0118
removed the grid: at rest 9 in the band and **9 sockets**; scrolled to a quarter, 12 and 12; scrolled to the
middle, **18 chat bodies in the band, 12 sockets and 6 quiet**; at the
bottom 11; and 8 once the 5 s hysteresis settled. It never reached 13. The
band alone therefore bounds a screenful (9–12); the cap is what bounds the
*band*, which at 4×14 panels holds about 18.

**Needs you** is the state this phase exists for. The chip is already the
fleet's, so it is right for an unloaded panel, a quiet one and a name-plate.
The body adds the question: the open ask renders read-only (what was asked
and what the choices are, as static pills), and a bar on the body's floor —
one line and one action — carries the question's title and **Open**. Both
are built from the fleet's `dialog` (`lib/needsYou.js`'s `methodLabel` for a
dialog with no title of its own), so the header never depends on the socket.

**Ownership** (`paneOwnership.js`, plan §4.6):

| Situation | Unload | Load |
|---|---|---|
| the terminal's tab (`t:<id>`) or the agent's tab is in `host.openTabs` | park the pane only; the tab keeps the attach | `TermSurface` re-claims the pane |
| only on canvases | `suspendTermSocket` — socket closed, xterm and scrollback kept | `ShellTerm` kicks the suspended socket |
| more than 24 suspended, or one for 10 min | `closeShellTerm`, only while still suspended (a kicked entry belongs to someone again) | a fresh xterm; tmux holds the screen |
| its tab closes while the panel shows it | — | the body remounts with a fresh xterm (the tab disposed the old one) |
| the body moves host (maximize, restore) | **counted, not guessed**: `mounted` holds how many bodies show that pane, so only the last one out parks it, and even then on the next tick. The canvas mounts the maximize layer's body a commit *before* its React Flow node drops the wrapper's, because a node's data is applied in an effect — with a tick alone that order suspended the socket of the pane the viewer was looking at (measured: `readyState` 3 and keystrokes lost, C2 session 2) | the same xterm is re-parented (`ShellTerm`) and refits |
| `terminal.deleted` / `agent.deleted` on the feed | `forgetPane`: a pane the Canvas holds suspended is disposed at once; one still mounted is disposed when its body unmounts (the gone row replaces it); one a tab owns is the tab's | — |
| focus | one focused panel per canvas (click or arrows); only the focused **and engaged** panel gets `autoFocus`, so only it calls `term.focus()`; every visible panel re-claims its pane and fits on reveal |
| the same terminal twice on one canvas | the picker does not offer it; a second browser that has not heard yet gets the server's 409 and its message, and the optimistic wrapper goes |

**Saving** (plan §4.7). A drag or resize stop moves the local rows at
once; `layoutDiff` against the last saved rows accumulates the subset;
`PATCH …/layout {ifUpdatedAt, panels}` goes 500 ms later, and at once on
hide and on unmount. The 200 is applied like a
`canvas.layout` frame; a 409 refetches the canvas and toasts "Canvas
changed elsewhere — reloaded." A move is ignored while hidden,
at width 0 and during a gesture; feed frames that arrive during a gesture
are applied after it. A `canvas.panel.added` frame that beats its own POST answer
settles the pending wrapper of that binding at once. Adding a panel places it at `nextSlot` (32×42), shows
the wrapper at once and settles on the 201 — a refusal shows its message
and removes the wrapper; Remove and Delete apply locally on the 204;
Delete confirms through the app's alert dialog and the tab moves to the
last-opened or first canvas, else the empty state.

**Nothing is compacted, on a reveal or ever.** Until ADR-0118 the surface
re-ran the grid's own compactor over the whole store on every panels change
and saved the result — including the arrangement a *second* browser's drag
or remove had just produced, which was written back compacted on the next
reveal of this tab. The plane has no compactor, so that pass is gone and
what is saved is exactly what was dragged. Two things remain in its place:
**Tidy**, which packs in reading order only when asked (below), and the
**reveal refetch** — a tab revealed more than 10 s after its last read asks
`/api/canvases` and `/api/canvases/{id}` again, which repairs a store that
missed a feed frame while it was hidden without ever rewriting a rectangle.
The `compact` column survives in the schema and decides nothing.

**Copy** (plan §4.8): "No canvas yet. A canvas shows many agents,
terminals and notes side by side, live." — New canvas; "Add your first
panel." / "A panel is one agent, terminal or note on this canvas." — Add
panel; picker: "Everything is already on this canvas." / "No agents,
terminals or pins yet." — Close.

**Keyboard** (plan §4.6 "Focus"). The surface holds two bits: which panel
is focused (`focusedId`) and whether the keyboard is inside its terminal
(`engaged`). The wrapper is the roving tab stop of its canvas — `tabIndex`
0 on the focused panel, or the first in reading order until one is
focused, -1 on the rest — and hands every key to the surface, which owns
the model. `panelOrder` and `neighborPanel` (`web/shared/domain/canvas.js`)
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
node: the maximized panel's body renders in a layer over the plane
(`.cv-max`, absolute inside `.cv-stage` — no transformed ancestor to
break), the wrapper keeps its slot and says *Shown maximized* with a
Restore button, `.cv-body` goes `inert`, the plane's two floating clusters
go with the plane (**Chrome** below), and the layout is untouched. The
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

`Plane.jsx` is the plane's host: `@xyflow/react` 12.11.6,
**lazy-imported** (eager it is +63.3 KB gzip on the desktop's main chunk;
split it costs 205 B and lands in a 49 KB gzip chunk only a viewer who opens
a canvas fetches). It is a host and nothing more — the surface keeps the
store, the save flow, the focus model, the keyboard, the picker and the
maximize layer, and every panel is the **same `Panel` wrapper** rendered as
one custom node type, so a panel has the same header, chip, actions and keys
on the plane and in the maximize layer. Configuration is C0's
([`docs/benchmarks/2026-09-10-node-canvas.md`](../benchmarks/2026-09-10-node-canvas.md)):
`minZoom` 0.2, `maxZoom` 1.5, `snapToGrid` on an 8 px `snapGrid` (the canvas
unit, so a drag lands on whole units), `onlyRenderVisibleElements` **off**
(on, it unmounts a node with no dwell — 54 socket suspends and kicks over
three fast pans), three `NodeResizeControl`s for resize (`onResizeEnd`
fires once; they do not honour `snapGrid`, so `pxToUnits` rounds), the body marked `nodrag
nowheel` with the header as the drag handle, marquee selection on a left
drag, pan on the middle or right button or Space, and a `<MiniMap />` and a
zoom cluster themed from our tokens (React Flow's defaults are a near-white
panel over the app's dark surface). Node position is `x · 8, y · 8` px and
size `w · 8, h · 8`; `unitsToPx` / `pxToUnits` are the only place that
multiplies.

**The resize targets, and the counter-scaling rule** (`canvas.css`,
"Resize targets"). The library's controls drive the resize and fire
`onResizeEnd` once per gesture, which is what the save path debounces on —
that part is the library's and stays.

**Three targets, not eight** (owner, 2026-09-12). `NodeResizer` ships four
edges and four corners; a panel offers the **bottom** band, the **right**
band and the **bottom-right** corner, and the other five are not rendered
at all — `RESIZE_CONTROLS` in `Plane.jsx` names them one by one through
`NodeResizeControl`. Hiding them would not have been the same thing: a
control at zero opacity is still a target the pointer finds. The five that
went are the ones that move a panel's top-left corner while resizing it,
which on a plane read left to right and top to bottom means the thing you
were aiming at slides away under the drag. What the library *draws* could not be used: its
handles and line controls live **inside the transformed plane**, so their
size on screen is their size divided by the zoom, and its line control is
one plane pixel wide with a transparent border. Measured on a real plane
before the change (`getBoundingClientRect`, intersected with the panel
because `.cv-panel` is `overflow: hidden` and clips the outward half of
every control): an **edge band** was **1.0 / 0.8 / 0.4** screen px thick at
zoom 1.0 / 0.8 / 0.4, invisible and effectively unhittable; a **corner**
was a 5 × 5 px box of which **3.5 / 3.3 / 2.9** px was inside the panel.

So every length in that block is written as a multiple of `--cv-px`, which
is **one screen pixel expressed in plane units**: `calc(1px /
var(--cv-zoom))`, where `--cv-zoom` is the live zoom `Plane.jsx` publishes
on the flow root on mount and on every viewport change. A corner target is
20 screen px and an edge band 12 screen px at 0.4, at 1.0 and at 1.5 alike.
`NodeResizer`'s own `autoScale` is off, because it answers the same
question worse — an inline `scale: max(1 / zoom, 1)` on corner handles
only, doing nothing to an edge band and nothing at all above zoom 1 — and
two counter-scalings would divide by the zoom twice. This is the
rule someone deletes as an unnecessary `calc`: remove it and the targets
shrink out of reach with no sign on screen that anything changed.

**The hit area and the paint are deliberately not the same rectangle.**
The control element is the hit area: transparent, generous, and entirely
*inside* the panel, because `.cv-panel` is `overflow: hidden` and clips
anything that sticks out. Its `::after` is the paint, and keeps the grid's
visual language — a 24 × 3 bar at the middle of each edge, an L at each
corner, both `--text-secondary`, appearing on hover, focus and selection.
The seeable rectangle has to be the smaller of the two; collapsing them
into one is the defect this replaced.

**Zoom decides the body** (plan §4.3, `loadPolicy` with `{zoom, band}`,
`zoomBody`, `pointerAtZoom` — one test per row in `canvas.test.js`). The
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
| in the band, `zoom < 0.75` | **still**: `term.buffer.active` as text, monospace, inert, **on the terminal's own ground**; the header stays live from the feed |
| `zoom < 0.4` | **name-plate**: face, name and status colour sized by `1 / zoom`, because a still down there is grey texture and the header does not resolve either |
| outside the band, any zoom | today's placeholder; the socket suspends after the 5 s hysteresis |
| focused and engaged | the plane animates to `zoom = 1` before the pane takes keys — Enter engages through the same snap |

**A still wears the terminal's colours** (owner, 2026-09-12: "enquanto
tiver escrita de terminal precisa exibir o terminal"). It used to take the
panel's own `--bg-panel` and `--text-secondary`, so crossing 0.75 turned a
black pane white while the words on it stayed the same — the band read as
the panel having become something else, which is the one thing it must not
say. `Plane.jsx` sets `--cv-term-bg` / `--cv-term-fg` on the plane root from
`xtermTheme(readTermTheme())` — the very pair xterm itself is configured
with — and refreshes them on `picode-term-theme`, one listener for the whole
plane rather than one per panel. What a still still cannot show is **colour
in the text**: `translateToString` drops every cell attribute, so the
capture is plain text and a coloured prompt reads monochrome down there.
Recovering it would mean capturing per-cell attributes, which is the cost
the band exists to avoid.

The rule is about a **cell**, so it only binds a body that has one.
`hasPane` (`web/shared/domain/canvas.js`, the allow-list `PANE_STATES`) is
the gate on all three halves — the `is-inert` / `is-still` pointer-events
rule, the `.cv-snap` layer, and **the still row of `loadPolicy` itself**:
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

A **chat** body (phase 4) is the one exception to the last row's "no socket
to suspend": it has no cell, so the still never reaches it, but it does hold
a connection, so the plate and the band close it. The **Chat body** section
above is its table. Measured on the canvas at 1600 × 1100 with two managed
agents and a shell: at 100 % and 81 % both chats live with 2 sockets; at
47 % the shell is a still and the two chats are **still live** with their
2 sockets; at 23 % and 20 % three name-plates and **0 sockets**, the panels
still loaded; back at 50 % the 2 sockets and both conversations (10 and 3
turns) returned.

The band it reports back is still the pane band, because the band is what
bounds the *attaches*; `zoomBody(zoom, band, pane)` answers both rows from
one table and `loadPolicy` picks per entry (`canvas.test.js`, three rows).

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
are live. A camera that *arrives* below the band (a stored viewport) fills
the gaps from the xterm instances a suspend keeps. A panel with no capture
yet shows the unloaded body's one muted line, and a still whose panel has
moved on the feed since carries its age in the header.

**The camera is per viewer**: `localStorage` `picode-canvas-view:<id>` holds
`{x, y, zoom}`, restored on open, debounced on `onMoveEnd`, with a fit to
the panels the first time a viewer opens that canvas. It never reaches the
store — a pan is not an edit (ADR-0113).

**Tidy** (`tidyCanvas`, the header menu) is the plane's answer to automatic
compaction: explicit instead of silent. It lays the panels out in reading
order, sizes kept, three default-sized panels across (98 units), each row as
tall as its tallest panel, and saves once through the layout PATCH.

**Tidy arranges; it never resizes.** The compactor it stands in for never
resized either — it only ever moved items up into free space — so a Tidy
that also normalised sizes would silently throw away a resize the owner of
the panel made on purpose. Rows are as tall as their tallest panel for the
same reason: the row gives way to the panel, not the other way round.

**What the plane adds to the keyboard** (everything phase 3 shipped keeps
working — `neighborPanel` was already a 2D spatial search, so the arrows
need no change on a free plane): `+` / `-` zoom, `0` fits, Space-drag pans,
Enter engages through the snap to 1, and moving focus to an off-screen panel
**pans it into view**, which is what loads it. A panel already on screen is
left where it is: an arrow key must not shove the plane about.

**Accepted in a browser (2026-09-10, C2 session 2)** on a scratch instance
with eight panels — an interactive agent's TUI, a managed agent and six
shells — plus a 24-panel canvas, driven at 1600 × 1200:

| Row | Measured |
|---|---|
| the pointer rule, end to end | at **1.0** a real click aimed at cell (25, 13) of a 29 × 16 pane came back from the terminal as `^[[<0;25;13M` — the cell aimed at, through tmux `mouse on`. At **0.83** xterm's own mapping for the same point answered **(21, 11)** — 4 columns and 2 rows out — the body read `pointer-events: none`, the click produced **no** report at all and moved the plane to 100 %, and the click after it reported (25, 13) again |
| bodies per zoom | 1.0: 7 live, 7 attached. 0.83: 7 live, every panel `is-inert`. 0.69: stills, **0 attached, 7 suspended, 7 instances kept**. 0.30: name-plates, 0 attached |
| chunk loading under the transform | at rest 7 attaches and **7 tmux clients**, one per session. A 3 883 px pan at 4 678 px/s (61.4 fps) and +10 s: 0 attached, 7 suspended, the 7 instances still there. Back at 4 781 px/s (62.8 fps): the same 7 instances reattached, 7 tmux clients. Zoom 0.2 → 0 attaches; back to 1.0 → the same 7 |
| a marquee of many | 24 nodes selected at **61.9 fps**, dragging all 24 at **62.7 fps** (worst frame 16.8 ms), **one** layout PATCH of 24 rows, every rectangle moved by exactly (25, 19) units. C0's 31 fps was 20 nodes among 500 with nine live bodies; this is a lighter scene and not a refutation of it |
| maximize on a canvas | the pane refits 29 × 16 → **120 × 58**, tmux follows, the wrapper says *Shown maximized*, and the socket stays open in **both** directions |
| a real 409, from a second browser | a layout PATCH answered 409, refetched (200), toasted *Matrix changed elsewhere — reloaded.*, rolled the optimistic move back and took the other browser's, with no lingering wrapper |
| the camera is per viewer | two browsers on one canvas at (−562, −623.4, 1.0) and (565, 288, 0.712); neither moved the other, and a reload restored each its own |
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
| the picker | five groups (Agents · Terminals · Pins · Open files · Changes to an open file) from `GET /api/pins` and `host.openTabs`; a binding already on the canvas is not offered, and with no file tab open the Files line reads *No file is open — open one in a tab…*. `__picodeOverlayAudit()` `ok` with it open |
| the bodies | the note renders the pin's markdown (headings, task list, table, quote, fenced code); the file is the editor with Preview/Raw/Save; the diff shows `+2 −0` with **Open file**; the shell is a live xterm |
| the feed | a `PATCH /api/pins/{id}` from another client changed the note's body in place, and `canvas.panel.added` from a second client drew a fifth panel — no timer either side |
| the zoom rows | 100 %: all four live, none inert. 78 %: only the terminal is `is-inert` with the snap layer. 64 % and 52 %: the terminal is a still, the note, file and diff still render. 42 %: same. 35 %: every panel a name-plate. Back at 100 %: all four live again |
| the gone rows | deleting the pin left *That pin is gone.* — Remove (name kept until the page reloads, then the ref); deleting the terminal left the file and the diff at *Where this file was read from is gone.* — Remove, chip **Gone**, with only Remove offered |
| unsaved work | typing in the file panel turned the chip to **Unsaved**; with the canvas tab hidden for 9 s every other body unloaded (`data-loaded="0"`) and that one did not; maximizing it kept the text and the chip |
| the store's answers | 409 *This note is already on this canvas* and *This diff is already on this canvas*; 400 *ref must be `<owner>:<id>:<path>` with owner t, a or w* and *kind must be agent, terminal, note, file or diff*; the same path as a `file` and as a `diff` is two panels |
| Open in Pin Studio | the note header's Open lands on `#/pins/<id>` with that pin loaded |

**Accepted in a browser (2026-09-10, C4)** on a scratch instance at
1440 × 900 in both themes, seeded with two workspaces (`Alpha` at the
worktree, `Beta` at `/tmp/picode-qa-beta`), four `pi` agents with recorded
sessions — Atlas, Bravo, Cleo in Alpha, Delta in Beta — and a five-panel
canvas whose fifth panel is a note. Every mailbox claim was **driven
through the MCP endpoint** with a bearer the scratch minted, never inferred
from the UI:

| Row | Observed |
|---|---|
| baseline, before any edge | `list_contacts` — Atlas sees `Bravo` (same workspace, ADR-0104 unchanged); Delta sees **nothing** |
| draw between two enrolled panels, one workspace | a real pointer drag from Atlas's connector to Bravo's: the line appeared, **one** `POST …/edges` → 201, no dialog, and a **second browser session** that never touched the plane drew the same `edge-90d707f3e244` from `canvas.edge.added` alone |
| draw when an end is not enrolled | *Connect Cleo?* — "…lets Atlas and Cleo message each other. Neither can read the other's history." **Cancel**: still 1 edge, Cleo still has no connection, and the only request the whole gesture made was the one `GET /api/communication` it asks with. Confirm: `POST /api/communication` 201 **then** `POST …/edges` 201, and both lines read `is-live` |
| draw across two workspaces | a second confirm, on its own: *"Atlas works in ~/picode/.worktrees/matrix-edges and Delta in /tmp/picode-qa-beta…"*. After it, **`list_contacts` for Delta returned `Atlas`** (workspace `alpha-ed1885`) where it had returned nothing, and `send_message` Delta → Atlas was **accepted** and landed in the Messages history |
| remove the line | the chip on the line → *Remove this link? Delta and Atlas will no longer be able to message each other.* → Delta's contacts **empty again**, Atlas no longer lists Delta, and `send_message` answered *connection is disabled or its recorded conversation changed* |
| revoke one connection in Messages | Advanced → Cleo → **Disable**: the audit row turned amber with *Cleo's connection was revoked; the link grants nothing.* (no reload — the `peer.*` row), the plane drew the edge `is-broken`, and Atlas's contacts dropped Cleo |
| re-connect | drawing Bravo → Cleo offered the enrolment again; confirming it made **all three** edges live at once, including the one that had been broken — the grant is derived, so nothing had to be repaired |
| the session moves | `PATCH /api/agents/{delta}` to a new `sessionPath`: the link read *Delta's session changed; the link grants nothing.*, and the old bearer stopped answering. Putting the path back made it live again — **both ways, with no reload** |
| delete one panel | removing the Delta panel took its edge with it (3 → 2 edges, no per-edge event) and Delta's contacts went **empty** |
| the two meanings of Delete | selecting the line and pressing <kbd>Delete</kbd> asked the same one-line confirm and removed the edge (3 → 2); with a **panel** focused, <kbd>Delete</kbd> still removed the panel (5 → 4) with its Undo toast and left the edges alone |
| a note panel | no connector on it, and a hand-made `POST …/edges` with the note's id answered **400** *panel … is a note panel and has no mailbox: an edge links agent or terminal panels* |
| the audit list | the same edges as the plane, each with both ends, both kinds, the canvas, and whether it grants; its **Remove** confirmed in one line and revoked for real (Delta's contacts `["Cleo"]` → `(none)`); the list survived a reload; it listed the cross-folder pair with **no** workspace chosen |
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

**Accepted in a browser (2026-09-11, ADR-0118 session 2)** on a scratch
instance at 1280 × 633 in both themes, with a canvas holding a live shell, two
`pi` agents and a note:

| Row | Observed |
|---|---|
| the tile | **Canvas**, `lucide-frame`, no letter fallback; opening it mounts `CanvasSurface` from its own chunk |
| the empty states | *No canvas yet.* — New canvas, then *Add your first panel.* — Add panel, each one line and one action (the second was re-shaped on 2026-09-11 into the centred card on the plane — *The switcher* / *An empty canvas* above) |
| three panel kinds | a live xterm, a stopped agent's *Agent is stopped.* — Run, and a note's markdown, all on one plane with the minimap and the zoom cluster |
| pan | a real middle-button drag moved the camera `translate(134, 102)` → `translate(-108, 50)` at scale 1 |
| the zoom rule, step by step | 100 %: the pane live and pointer-taking. 83 %: `is-inert`. 69 %, 58 %, 48 %, 40 %: `is-still is-inert`. 33 %, 28 %, 23 %: every panel `is-plate`. Back at 100 %: all live again, no class left behind. The note and the agent never go still — they have no cell (C3's rule) |
| `#/app/matrix` | cold load: the address bar read `#/app/canvas` before the first paint and settled on `#/app/canvas/<last>`; no *That app is gone.* |
| `#/app/matrix/<id>` | cold load and in-session `location.hash =`: both landed on `#/app/canvas/<id>` with that canvas selected and its three panels drawn |
| `x:matrix` in `localStorage` | `picode-tabs` set by hand to `{ids:["x:matrix"],selected:"x:matrix"}` with `picode-matrix-last` and `picode-matrix-view:<id>` = `{-321, -77, 0.72}`: the reload opened the **Canvas** tab on that canvas at exactly `translate(-321px, -77px) scale(0.72)`, and storage afterwards held only `x:canvas`, `picode-canvas-last` and `picode-canvas-view:<id>` — the old keys were gone |
| the picker | grouped Agents · Terminals · Pins, a binding already on the canvas not offered, and *No file is open…* under the list. `__picodeOverlayAudit()` `ok` with it open |
| links | a real drag from a header connector drew the line; an end with no conversation was **refused in a toast**, not offered a failing action; two enrollable ends got *Connect Atlas and Bravo?* with the one line ADR-0116 §5 asks for. `__picodeOverlayAudit()` `ok: true`, dialog buttons 36 px = `--ctl-h`, in both themes |
| the header link chip | both ends of an edge wore **1**, amber, titled *1 link, broken: it grants nothing. Read them in Messages.*, and the chip showed **in the maximize layer too**, where there is no line to read |
| maximize | the note and then a shell: the layer fills the surface, the xterm refits, the wrapper says *Shown maximized*, Restore puts it back |
| the audit list | **Granted contacts** in Messages (**Canvas links** until 2026-09-11), both rows with ends, kinds, the canvas name, the reason and Remove |
| the old API | **zero** requests to `/api/matrices` (the network log and a grep of the built bundle); `GET /api/matrices` answers 404. No console error names anything renamed |

Two things this pass found and fixed. `linkCounts` / `linkChipTitle` had been
computing a chip nothing rendered — grid mode was its only consumer, and
deleting grid mode left the surface passing it nowhere. It is wired to the
plane and the maximize layer instead of deleted, for the reason the paragraph
above gives. And `readView` now carries a camera across the key rename rather
than fitting the plane afresh.

**Re-measured on a scratch (2026-09-09, phase 3 session 2)** against
[`docs/benchmarks/2026-09-09-matrix-live-grid.md`](../benchmarks/2026-09-09-matrix-live-grid.md):
nine live TUI bodies at 60.0 fps in-page, zero long tasks and 4.2 % of one
core in the page's renderer (0.1 % idle); dragging with those nine live at
60.3–60.9 fps; a pass at the study's speed (16 000 px/s, each panel 0.17 s
in the band) mounted 8 bodies. The dwell is a *speed* rule, not a distance
one: the same 4 s pass over a short 46-panel canvas leaves every panel
0.70 s in the band, so all of them legitimately load, and the 5 s
hysteresis returns to the steady 12–21 within 8 s.
