# Matrix v2 — the infinite canvas (plan)

- **Date:** 2026-09-10
- **Status:** accepted — every question in §8 was taken by the owner on
  2026-09-10, question 7 included. **C0, C1, C2, C3 and C4 are done**; C5 is
  the only phase left, and it is deferred to its own plan.
  **C0's numbers are folded in below**
  ([`docs/benchmarks/2026-09-10-node-canvas.md`](../benchmarks/2026-09-10-node-canvas.md)):
  GO on `@xyflow/react`, with one rule this plan did not have — a live
  terminal is only correct at zoom exactly 1.0.
- **Ask (owner):** "o matrix ser uma view para trabalho inspirada nesse
  projeto" ([nodeterm](https://nodeterm.dev)), "continua como view a mais,
  a ideia é essa mesmo virar um canvas infinito". Library question answered
  in §3: `@xyflow/react` (React Flow).
- **Builds on:** `docs/plans/matrix-app.md` (v1, phases 0–3, shipped
  2026-09-09/10), `docs/architecture/matrix.md` (ADR-0108 persistence,
  ADR-0109 native app surfaces), `docs/benchmarks/2026-09-09-matrix-live-grid.md`
  (the phase-0 measurements this plan reuses),
  `docs/benchmarks/2026-09-10-node-canvas.md` (the C0 spike that gates it).
- **Scope guard:** Matrix stays **one more view**, not the shell. The tab
  strip, the sidebar and the routes are untouched; a matrix simply gains a
  second layout mode. Fullscreen (`picode-focus`) composes with it — canvas
  plus fullscreen is the "work view" the owner described.

## 1. What v2 is, in one paragraph

A matrix gains a **canvas mode**: the same panels, freed from the
12-column grid, placed anywhere on a plane the viewer pans and zooms, with
a minimap and spatial keyboard movement. Live terminals stay live near
100 % zoom; zoomed out, a panel shows its last screen as text with its
status header still live, so a 200-panel matrix costs the attaches of the
dozen you are actually reading. Grid mode stays exactly as it shipped; the
mode is a property of the matrix, switchable from its header.

## 2. Benchmark — nodeterm (studied 2026-09-10, public docs only)

Sources: <https://nodeterm.dev> and the public README at
`eneskirca/nodeterm` (1.8k stars). **BUSL-1.1** — studied as a benchmark,
never vendored: no code, no assets, no copied strings. Everything below is
inference from their published material.

| What they ship | Where PiCode already stands |
|---|---|
| xterm + tmux sessions surviving reboots | the core since ADR-0002/0018 |
| `RUNNING` / `NEEDS YOU` badges on nodes | Working / Needs you / Ready from wrapper leases and lifecycle hooks (ADR-0062), already on Matrix panel headers |
| Claude Code, Codex, Gemini nodes | Claude Code, Codex, Grok, Hermes, OpenCode and Pi (ADR-0069) |
| QR pairing to a phone, E2E, push | paired devices (ADR-0043/0049) and Web Push (ADR-0047) |
| Voice dictation | speech input in the composer |
| "Server Edition" for browser access | PiCode **is** the browser product; for them it is an add-on to Electron |
| Editors, diffs, notes, web as nodes | CodeMirror file tabs, `WorkingDiff`, pins with sketches — as tabs, not as nodes |
| Infinite canvas, pan and zoom, React Flow | **the gap this plan closes** |
| Edges wiring agents for shared transcripts | **matched as a mailbox link, refused as transcript sharing** (ADR-0116, C4): an edge grants ADR-0104's contact and nothing else. Someone comparing PiCode to nodeterm will find no transcript sharing; that is the decision, not an omission |
| Kanban board of live sessions | **refused here** — a board is its own surface, and PiCode's queue is the Inbox (ADR-0037). If it is ever wanted, it is a separate app, not a mode of this one |

**The adaptation:** take the spatial canvas and the node vocabulary;
refuse the board; make edges mean a capability PiCode can honestly grant,
or do not draw them.

## 3. Library — `@xyflow/react` (React Flow v12)

The decision is made by one requirement the usual comparison tables miss:
**a node's body is a live terminal** — DOM plus a canvas, with its own
focus, selection, IME and key handling.

| Candidate | Renders | Verdict |
|---|---|---|
| **`@xyflow/react` 12.11.6** | DOM | **chosen, and measured in C0.** MIT, published 2026-09-01, 6.2 M downloads/week, deps `zustand` + `classcat` + `@xyflow/system`, 1.2 MB unpacked. 500 nodes pan, zoom and drag at 60 fps; `nodrag`/`nowheel` hold; `NodeResizer` resizes a live pane |
| GoJS 4.0.3 | HTML5 canvas | impossible: a live terminal cannot live inside a canvas scene graph. Also a paid licence |
| JointJS 3.7.7 | SVG | interactive DOM only through `foreignObject` (focus/IME/selection are fragile); OSS package last published 2024-03; the maintained edition is commercial |
| Rete.js 2.0.6 | DOM | a dataflow/visual-scripting framework with sockets; our nodes are not dataflow. 34 k downloads/week |
| Hand-rolled (`react-zoom-pan-pinch` + absolute panels + an SVG layer) | DOM | honest fallback — our graph needs are shallow — but it re-implements selection, marquee, minimap, culling and keyboard. AGENTS.md prefers a popular primitive |

What React Flow gives us that we would otherwise write: pan, zoom,
marquee selection, `<MiniMap />`, `<NodeResizer />`, a serialisable
node/edge model, and — the reason it fits a terminal — documented escape
hatches for interactive node content: **`nodrag`** (element does not drag
the node), **`nowheel`** (wheel does not zoom the canvas), **`nopan`**
(drag does not pan). Props this plan relies on: `defaultViewport`,
`onMoveEnd`, `minZoom`/`maxZoom` (defaults 0.5/2), `translateExtent`,
`onlyRenderVisibleElements`, `snapToGrid`/`snapGrid`, `panOnDrag`,
`zoomOnScroll`, `onNodesChange`.

**Their documented caveat is exactly our case**: "mouse events aren't
consistent when a node contains a `<canvas />` — scale computed positions
by 1/zoom". C0 measured it and it is worse than a caveat: **xterm divides
the pointer's offset inside the transformed rect by the untransformed cell
size**, so the cell it reports is `cell × zoom` and the error grows with
the distance from the pane's corner (10 columns and 3 rows at zoom 0.8 on
a 54 × 16 pane). xterm exposes no scale to correct, so §4.3 makes zoom 1.0
the condition for a pointer, not a preference.

**Bundle, measured (C0).** Eager, the library costs the desktop main chunk
**+60 761 B gzip** of JS and **+2 552 B gzip** of CSS — 2.6× the
+24 309 B gzip react-grid-layout cost, and over the 60 KB threshold the
phase-0 study used. **Lazy-imported** it moves the main chunk by 205 B
gzip and lands in a chunk of **46 181 B gzip** JS + **2 687 B gzip** CSS
that only a viewer who opens a canvas fetches. The canvas host is lazy —
decided, not optional.

**Do we swap the grid library?** Yes, eventually — in two steps, never in
one, and the second step waits for the owner to have used the canvas.

1. **C2 builds canvas mode on React Flow while grid mode keeps
   react-grid-layout.** Rewriting a mode that shipped yesterday, with its
   tests, QA and visual review, *in the same branch that introduces a new
   library* would make a regression impossible to attribute.
2. **After the canvas has been used for real**, one of three end states is
   chosen with evidence rather than argument:

| End state | What it means |
|---|---|
| One library, two modes | grid mode is ported to React Flow — `snapToGrid` plus our pack function — and RGL and `react-resizable` are removed |
| One library, one mode | the canvas with snapping proves better than the grid at everything the grid did, and grid mode is deleted (the mode transform in §4.1 already converts the coordinates) |
| Two libraries | only if C0 finds a blocker in React Flow that RGL does not have |

The recurring cost of two engines is not the 24 KB gzip: it is that every
later panel feature — a new node kind, a header action, a keyboard rule —
has to work in two hosts with two drag/resize/selection models. That tax is
why one library is the target.

**What porting grid mode would cost, after C0.** `snapToGrid` with an 8 px
`snapGrid` places exactly (a drag lands on multiples of 8; with it off the
same drag lands at 78). Three grid behaviours have no React Flow
equivalent, not one:

| RGL does | React Flow |
|---|---|
| automatic vertical compaction | ours — a pure pack function (~40 lines, `nextSlot` is most of it) behind a **Tidy** action |
| **collision resolution** — items never overlap, the ones in the way are pushed | none: C0 left a dragged node overlapping its neighbour. Ours to write |
| **reflow to container width** — 12 columns of variable pixel width | none: canvas coordinates are absolute. The one a user notices on a laptop |
| per-item min sizes and edge handles | `NodeResizer` (`minWidth`/`minHeight`, 8 handles), proven against a live terminal |
| keyboard placement | neither; the Matrix's arrows are ours (`neighborPanel`) |

C0 found **no blocker that forces "two libraries"** — the end state stays
the owner's call after using the canvas.

> **Answered, 2026-09-11: "one library, one mode".** The owner took it
> ("quero remover já o grid e renomear a funcionalidade de matrix para
> canvas") on the evidence of using the canvas: the only board in the
> instance was in canvas mode, and none of the three grid behaviours above
> had been asked for since it shipped. [ADR-0118](../decisions/0118-canvas-replaces-matrix.md)
> is the decision. Grid mode, the mode column and the two dependencies are
> gone; migration 045 converted every grid rectangle once; the app is
> called **Canvas**, all the way down to the tables, the routes and the
> events. Vertical compaction is deferred, not refused — it could become a
> canvas behaviour later without a second engine.

## 4. Design

### 4.1 Coordinates and the data model

`matrix_panels.x/y/w/h` stay integers and keep their meaning **per mode**,
documented once and validated per mode:

| Mode | Unit | Rules |
|---|---|---|
| `grid` (today) | grid cell (12 columns, `rowHeight` 24 px) | `x + w ≤ 12`, `w ≥ 4`, `h ≥ 8` |
| `canvas` | **8 px canvas unit** | no column cap; `w ≥ 32` (256 px), `h ≥ 28` (224 px); `x`, `y` may be negative |

Integers keep the layout PATCH, the events and the diff exactly as they
are — no floats, no rounding drift, and `snapGrid` of 8 px falls out for
free. Migration (number allocated by `make adr`/the migration convention at
implementation time; 041 is Matrix v1):

```
ALTER TABLE matrices ADD COLUMN mode TEXT NOT NULL DEFAULT 'grid';  -- grid | canvas
```

`compact` stays and keeps meaning only in grid mode. Switching mode is a
**documented one-time transform** applied server-side in the same
transaction as the mode change: grid → canvas multiplies cells to units
(x·(colWidth/8) rounded, y·3, w·(colWidth/8), h·3); canvas → grid packs
with the pure pack function and clamps to the 12-column rules. The switch
answers with every panel it moved, so the client has one reducer path, as
every other mutation does.

**The viewport is per viewer, not stored**: `localStorage`
`picode-matrix-view:<id>` holds `{x, y, zoom}`, restored on open, with
`fitView` the first time a viewer opens that matrix. Two browsers must not
yank each other's camera; this follows the same rule as the inspector
width and the open-tabs list. `x/y/w/h` stay shared — moving a panel is an
edit, moving the camera is not.

### 4.2 Node kinds

v1 binds `kind ∈ {agent, terminal}`. Canvas mode makes more kinds worth
having; each is a separate, small phase so the canvas itself ships first:

| Kind | `ref` | Body | Phase |
|---|---|---|---|
| `terminal`, `agent` | terminal / agent id | today's live pane | C2 |
| `note` | pin id | the pin's markdown, read-only, with **Open in Pin Studio** | **C3, done** |
| `file` | `<owner>:<id>:<path>` | the existing `FilePane` in its embedded layout | **C3, done** |
| `diff` | `<owner>:<id>:<path>` | `WorkingDiff` | **C3, done** |
| `web` | url | sandboxed iframe — inherits the ADR-0036 marketplace stance; **owner decision before it is built** | later |

The store's `kind` is already an open text column with validation in one
place; adding a kind is a validator edit plus a body component, never a
migration.

### 4.3 Zoom, live terminals and chunk loading

The rule that makes a big canvas cheap, extending `loadPolicy` with one
input. **Rewritten from C0's measurements** — the pointer, not the
renderer, is what the zoom decides:

| Condition | Panel body |
|---|---|
| in the band and `zoom = 1.0` | live pane, mounted, attached, **and the only state that takes a pointer** |
| in the band and `0.8 ≤ zoom < 1.0` (or `zoom > 1.0`) | live pane, readable and taking keys, but `pointer-events: none` on the body: a click on it runs "snap to 1" first and hands the pointer over afterwards |
| in the band, `zoom < 0.75` | **still**: the terminal's visible buffer as text (`term.buffer.active`), monospace, non-interactive; header stays live from the feed |
| `zoom < 0.4` | **name-plate**: face, name and status colour sized to the panel — at 0.2 a text still is grey texture, headers included |
| outside the band (any zoom) | today's placeholder, socket suspended after the 5 s hysteresis |
| focused and engaged | the canvas animates to `zoom = 1` before the pane takes keys — "snap to 1 on engage" |

**Why 1.0 and not 0.8.** C0 measured xterm's pointer→cell mapping under the
canvas transform: it divides the pointer's offset inside the *transformed*
rect by the *untransformed* cell size, so the cell it reports is
`cell × zoom`. At 0.8 a click aimed at column 50 row 15 is reported as
column 40 row 12, and a drag-select of ten characters returns seven; at 1.5
the same drag returns thirteen. PiCode ships tmux `mouse on`, so those
coordinates reach tmux copy mode and every mouse-aware TUI. Rendering is
*not* affected — the DOM renderer stays crisp at 0.8 — which is why the
rule has to be explicit: nothing on screen tells the user the clicks lie.
The band is hysteretic (live above 0.8, still below 0.75) so a viewer
parked on the boundary does not thrash nine sockets.

`minZoom` 0.2, `maxZoom` 1.5. `IntersectionObserver` does account for the
transform (C0 verified it: pan the plane away and the attaches fall to 0,
pan back and the same nine xterm instances reattach), so the existing
observer stays — with two corrections. Its `rootMargin` gains its
horizontal half (`100%`, not the grid's `100% 0px`: 49 panels in the band
at zoom 1 and 1600 × 1200, against 28). And the band is a **screen-space**
rule, so zooming out multiplies the plane it covers — at 0.2 all 500
panels were in the band. The still rule, not the band, is what bounds the
attaches when zoomed out; `loadPolicy` gains `zoom` and the still/live
split, and stays pure and unit-tested per row.

The still is memory-only, never persisted, and costs **0.02 ms** for a full
83 × 26 screen — so it is captured **before** the mode flips, not at
unmount: React renders the still body before it runs the live body's
cleanup, so a capture in the cleanup is always one crossing late (C0's
first crossing painted an empty still).

### 4.4 Chrome on the canvas

`<MiniMap />` bottom-right (a 200-panel matrix is unnavigable without
one), zoom controls beside it, `snapToGrid` on by default with a **Tidy**
action in the header menu, marquee selection for multi-panel moves, and
the existing header (switcher, Add panel, New matrix, Rename/Delete) with a
**Grid | Canvas** segmented control added. Panels keep the header they
have — face, name, status chip, Open · Maximize · Remove — with the body
marked `nodrag nowheel` and the header as the drag handle, exactly as
today. C0 proved all three gates: a wheel over the body left the zoom at
1.000 while the same wheel over the pane took it to 0.758; a body drag
selected text and left the node's position unchanged; the header dragged
the node and fired one `onNodeDragStart`/`onNodeDragStop` pair.

`onlyRenderVisibleElements` is **off**. It is the cheaper renderer (16
nodes mounted instead of 500 at rest, and the only thing that made a
20-panel drag 60 fps instead of 31) but it unmounts a node with no dwell
and no hysteresis, and a node's unmount is a live terminal's unmount: C0
churned 54 body swaps — 54 socket suspends and kicks — through three fast
pan round trips that cost **zero** with it off. Our own band already culls
the thing that matters.

The minimap costs nothing measurable at 500 panels (60 fps panning with
and without) but React Flow's default `<MiniMap />` and `<Controls />` are
a near-white panel over the app's dark surface, sitting on top of the
panels: C2 gives them `bgColor`/`nodeColor`/`maskColor` from the tokens
and a placement that does not cover a panel.

### 4.5 Keyboard

Everything from phase 3 stays: arrows move between panels (`neighborPanel`
is already a 2D spatial search — it needs no change for a free canvas),
Enter engages, Shift+Esc leaves the pane, Delete removes with an undo
notice. Canvas adds: `+`/`-` zoom, `0` fit-to-view, space-drag to pan, and
Enter's engage triggering the snap to 1.

### 4.6 What the canvas must not break

- One xterm and one attach per terminal id (`terms.js`), the pane
  following the visible host — the invariant from v1 §2.2, now with a
  third host (a canvas node) in the rotation.
- A hidden matrix still unloads everything after 5 s; fullscreen mode and
  the canvas compose without either owning the other.
- The layout PATCH still sends only the panels that moved, with
  `ifUpdatedAt`; a marquee drag of 50 panels is one request of 50 rows.
- The feed still drives every list; no polling.

## 5. Phases

| # | Branch | Delivers | Gate |
|---|---|---|---|
| C0 | `feat/matrix-canvas-spike` | **done 2026-09-10** — [`docs/benchmarks/2026-09-10-node-canvas.md`](../benchmarks/2026-09-10-node-canvas.md). **GO** on `@xyflow/react`: 500 nodes at 60 fps, `nodrag`/`nowheel`/header-drag hold, chunk loading bounds the attaches under the transform, `NodeResizer` resizes a live pane and ends once, the still costs 0.02 ms, +63 KB gzip eager / 49 KB lazy. One rule changed: **live terminals are only correct at zoom 1.0** (§4.3). No blocker forces two engines. Only the note and this plan merged | owner reads the note |
| C1 | `feat/matrix-canvas-model` | **done 2026-09-10** — ADR-0113 (amends ADR-0108) and migration 043: the `mode` column, per-mode units and plane bounds, `SetMatrixMode` (one transaction, the switch transform, the `matrix.mode` event), `mode` on `PATCH /api/matrices/{id}` answering summary + moved panels, and `matrix.js` per-mode validation with `gridToCanvas`/`canvasToGrid`. No UI; OpenAPI unchanged (no new route) | `make close` |
| C2 | `feat/matrix-canvas-surface` (two sessions) | **done 2026-09-10** — canvas mode end to end: the lazy React Flow host, the same `Panel` wrapper as a node type, zoom-aware `loadPolicy` with `zoomBody` / `pointerAtZoom`, stills captured before the flip, name-plates, a themed minimap and zoom cluster, the `Grid \| Canvas` switch with its lossy-direction confirm, Tidy, marquee, per-viewer camera, the canvas keys, and the `docs-site` guide. Accepted in a browser row by row (`docs/architecture/matrix.md`, *Accepted in a browser*): the pointer rule proved against real clicks and tmux SGR reports, agent panels, maximize, a 24-panel marquee at 62.7 fps in one save, 409s from a second browser, fullscreen, chunk loading under the transform. Two divergences kept: the band flips at gesture end, and Tidy never resizes | `make close`; visual card |
| C3 | `feat/matrix-node-kinds` | **done 2026-09-10** — all three kinds, one commit each. The store's `kind` grew by a validator line and its `ref` gained a shape per kind (`<owner>:<id>:<path>` for file and diff, parsed and built only in `matrix.js`); `bindingState` gained the gone rows (a pin deleted, an owner gone — a missing file is the body's news); `zoomBody`/`loadPolicy` gained a per-row `pane`, so a body with no cell never goes still and is a name-plate below 0.4; an editor with unsaved text is `keep`-pinned against the band **and** its document lives in `lib/fileDocs.js`, so maximize and the mode switch keep it. The picker groups by kind and offers files only from open file tabs. Accepted in a browser (`docs/architecture/matrix.md`, *Accepted in a browser (2026-09-10, C3)*) | `make close`; visual card |
| C4 | `feat/matrix-edges` (two sessions) | **done 2026-09-10** — ADR-0116 accepted before any code, and it refused the benchmark's headline: an edge grants ADR-0104's **mailbox contact and nothing else**, never a transcript. Session 1: migration 044, the `…/edges` routes, `edges` on the matrix read, contacts = workspace ∪ live edges (derived on every read, never cached; `SendPeerMessage` obeys the same union; the MCP surface gained no verb). Session 2: the header connector (`Handle` + `onConnect`, agent and terminal panels only), the two consent dialogs that are never merged (cross-folder naming both folders, then the existing enrolment), the broken state with its reason, a link chip with the count in grid mode, and **Matrix links** — every live edge, non-spatially, in the Messages view. Accepted in a browser row by row with the mailbox driven over MCP, not inferred (`docs/architecture/matrix.md`, *Accepted in a browser (2026-09-10, C4)*) | ADR accepted before code; `make close`; visual card |
| C5 | later | group nodes bound to a worktree, `web` nodes if the owner wants them. **The one-library end state is no longer C5's**: the owner took it on 2026-09-11 and `feat/canvas-only` shipped it (ADR-0118) — grid mode deleted, RGL and `react-resizable` removed, the whole family renamed to Canvas | its own plan |

C1 and C2's first session can overlap only if C1 lands the migration
first; the surface reads the mode from the API.

## 6. Decisions this plan does not take alone

1. **Edges mean access.** nodeterm's "context links let agents read each
   other's transcripts". PiCode can do it — the peer transport exists —
   but who may read whose session is a security-model decision, so C4
   starts with an ADR and the owner's approval, never with a line.
2. **`web` nodes** are an iframe on a surface that today hosts only
   first-party code (ADR-0036 keeps iframes for the marketplace era).
3. ~~**Which end state** of §3 to take, decided after the canvas has been
   used, not now.~~ **Taken by the owner on 2026-09-11: one library, one
   mode** — see §3 and [ADR-0118](../decisions/0118-canvas-replaces-matrix.md).

## 7. Risks

| Risk | C0 measurement | Mitigation |
|---|---|---|
| xterm under a CSS transform | **confirmed, and worse than expected**: the mapped cell is `cell × zoom`, so at 0.8 a click on (50, 15) reports (40, 12) and a ten-character drag returns seven; blur is *not* a problem — the DOM renderer stays crisp at 0.8 | a pointer only at `zoom = 1.0`; 0.8–1.0 is live but pointer-inert, and a click there snaps to 1 first (§4.3) |
| bundle growth on top of RGL | +60 761 B gzip JS + 2 552 B CSS eager; 46 181 + 2 687 B gzip in a split chunk lazy | the canvas host is lazy-imported — decided, not optional; retiring RGL in C5 gives most of it back |
| 500 nodes with a minimap | the minimap is free (60 fps panning with and without, 500 rects either way); the only expensive gesture is dragging 20 selected panels: 31 fps with `onlyRenderVisibleElements` off, 60 with it on | keep the culling **off** (it churns 54 socket suspends per three pan round trips); if the multi-drag hurts, bound the mounted bodies instead |
| two engines during C2 | — | time-boxed to one release; the mode switch is server-side, so a rollback is one column |
| a still that lies (stale text) | a capture is 0.02 ms, but one taken at unmount paints one crossing late (the first crossing showed an empty still) | capture before the flip and refresh while live; stamp the age in the header |
| a still nobody can read | at 0.5 it is sharp but marginal; at 0.2 it is grey texture and the headers do not resolve either | the name-plate body below ~0.4 (§4.3) |
| coordinate migration | — | the switch transform is one server transaction that answers with every moved panel, and it is reversible by switching back |

## 8. Decisions (taken by the owner in chat, 2026-09-10)

| # | Question | Decision |
|---|---|---|
| 1 | Library | `@xyflow/react` (MIT), per §3 — **C0 says GO**, lazy-imported |
| 2 | Canvas as a **mode** of a matrix, not a second app | yes — one store, one API, one surface |
| 3 | Viewport per viewer, panel positions shared | yes |
| 4 | Ship the canvas with terminals and agents only, then add note/file/diff | yes — C2 before C3 |
| 5 | Edges wait for their own ADR | yes |
| 6 | Kanban board is refused as a Matrix mode | agree, or say so and it becomes its own plan |
| 7 | **A live terminal takes the mouse only at zoom 1.0** (C0: xterm's pointer mapping ignores the canvas transform, and PiCode's tmux `mouse on` default carries the error into copy mode and every TUI) | yes — live and readable from 0.8, pointer only at 1.0, a click below 1.0 snaps to 1 first |

Every row was accepted as recommended, question 7 included: a live
terminal takes the mouse only at zoom 1.0, and a click below it snaps to 1
before the pointer reaches the pane. Changing any row is the owner's call
again (AGENTS.md, non-negotiable 6).
