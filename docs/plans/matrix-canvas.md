# Matrix v2 — the infinite canvas (plan)

- **Date:** 2026-09-10
- **Status:** proposed — waiting for the owner's call on §8
- **Ask (owner):** "o matrix ser uma view para trabalho inspirada nesse
  projeto" ([nodeterm](https://nodeterm.dev)), "continua como view a mais,
  a ideia é essa mesmo virar um canvas infinito". Library question answered
  in §3: `@xyflow/react` (React Flow).
- **Builds on:** `docs/plans/matrix-app.md` (v1, phases 0–3, shipped
  2026-09-09/10), `docs/architecture/matrix.md` (ADR-0108 persistence,
  ADR-0109 native app surfaces), `docs/benchmarks/2026-09-09-matrix-live-grid.md`
  (the phase-0 measurements this plan reuses).
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
| Edges wiring agents for shared transcripts | transport exists (broker, ADR-0104 peer communication); no visual edges, and the sharing rule is a security decision (§6, C4) |
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
| **`@xyflow/react` 12.11.6** | DOM | **chosen.** MIT, published 2026-09-01, 6.2 M downloads/week, deps `zustand` + `classcat` + `@xyflow/system`, 1.2 MB unpacked |
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
by 1/zoom". §4.3 turns that into a rule instead of a bug.

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
why one library is the target. The single thing RGL does that React Flow
does not is **automatic vertical compaction** (remove a panel and the ones
below rise); it becomes a pure function of ours (~40 lines, `nextSlot` is
most of it) behind a **Tidy** action, optionally run on every removal.

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
| `note` | pin id | the pin's markdown, read-only, with **Open in Pin Studio** | C3 |
| `file` | `<owner>:<path>` | the existing `FilePane` in its embedded layout | C3 |
| `diff` | `<owner>:<path>` | `WorkingDiff` | C3 |
| `web` | url | sandboxed iframe — inherits the ADR-0036 marketplace stance; **owner decision before it is built** | later |

The store's `kind` is already an open text column with validation in one
place; adding a kind is a validator edit plus a body component, never a
migration.

### 4.3 Zoom, live terminals and chunk loading

The rule that makes a big canvas cheap, extending `loadPolicy` with one
input:

| Condition | Panel body |
|---|---|
| in the viewport band **and** `zoom ≥ 0.8` | live pane (mounted, attached) |
| in the band, `zoom < 0.8` | **still**: the terminal's visible buffer captured as text (`term.buffer.active`, cheap and exact), monospace, non-interactive; header stays live from the feed |
| outside the band (any zoom) | today's placeholder, socket suspended after the 5 s hysteresis |
| focused and engaged | the canvas animates to `zoom = 1` before the pane takes keys — "snap to 1 on engage", which also removes every 1/zoom mouse-mapping question while typing |

`minZoom` 0.2, `maxZoom` 1.5. `IntersectionObserver` keeps working under
the canvas transform (it accounts for transforms), so the existing
observer stays; `loadPolicy` gains `zoom` and the still/live split, and
stays pure and unit-tested per row. The still is memory-only, never
persisted, refreshed when a live pane unmounts.

### 4.4 Chrome on the canvas

`<MiniMap />` bottom-right (a 200-panel matrix is unnavigable without
one), zoom controls beside it, `snapToGrid` on by default with a **Tidy**
action in the header menu, marquee selection for multi-panel moves, and
the existing header (switcher, Add panel, New matrix, Rename/Delete) with a
**Grid | Canvas** segmented control added. Panels keep the header they
have — face, name, status chip, Open · Maximize · Remove — with the body
marked `nodrag nowheel` and the header as the drag handle, exactly as
today.

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
| C0 | `feat/matrix-canvas-spike` | React Flow in the desktop bundle with React 19: 500 nodes, nine live xterm bodies, zoom 0.2→1.5, the still/live swap, minimap, `onlyRenderVisibleElements` on and off, `IntersectionObserver` under transform, xterm click/selection at 0.8 and 1.0, bundle delta, fps while panning and dragging. Study `docs/benchmarks/2026-09-10-node-canvas.md` (this §2 plus the measurements). **Decides one engine or two.** Nothing merges but the note and this plan | owner reads the note |
| C1 | `feat/matrix-canvas-model` | ADR amending ADR-0108 (mode column, per-mode units, the switch transform), migration, store + handlers + events, `matrix.js` per-mode validation and the pack function, OpenAPI | `make close` |
| C2 | `feat/matrix-canvas-surface` (two sessions) | canvas mode end to end: React Flow host, node wrapper reuse, zoom-aware `loadPolicy`, stills, minimap, mode switch, Tidy, marquee, keyboard, save/409, `docs-site` guide update, QA + visual review | `make close`; visual card |
| C3 | `feat/matrix-node-kinds` | `note`, then `file` and `diff` bodies; picker groups by kind | `make close` |
| C4 | `feat/matrix-edges` | **ADR first — it crosses the security model**: an edge grants two sessions the right to read each other, built on ADR-0104's transport; drawing, removing (which revokes), and what an edge shows when one end dies | ADR accepted before code |
| C5 | later | group nodes bound to a worktree, the one-library end state chosen per §3 (port grid mode or delete it, then remove RGL), `web` nodes if the owner wants them | its own plan |

C1 and C2's first session can overlap only if C1 lands the migration
first; the surface reads the mode from the API.

## 6. Decisions this plan does not take alone

1. **Edges mean access.** nodeterm's "context links let agents read each
   other's transcripts". PiCode can do it — the peer transport exists —
   but who may read whose session is a security-model decision, so C4
   starts with an ADR and the owner's approval, never with a line.
2. **`web` nodes** are an iframe on a surface that today hosts only
   first-party code (ADR-0036 keeps iframes for the marketplace era).
3. **Which end state** of §3 to take, decided after the canvas has been used, not now.

## 7. Risks

| Risk | Mitigation |
|---|---|
| xterm under a CSS transform (blur, mouse mapping) | live only at `zoom ≥ 0.8`, snap to 1 on engage; C0 proves click and selection at both zooms before anything ships |
| bundle growth on top of RGL | measured in C0; the canvas host is lazy-imported; retiring RGL in C5 gives most of it back |
| 500 nodes with a minimap | C0 measures with `onlyRenderVisibleElements` both ways; our own chunk loading already bounds the expensive part (attaches), not the wrappers |
| two engines during C2 | time-boxed to one release; the mode switch is server-side, so a rollback is one column |
| a still that lies (stale text) | the still is captured at unmount and stamped with its age in the header; anything older than the feed's last event says so |
| coordinate migration | the switch transform is one server transaction that answers with every moved panel, and it is reversible by switching back |

## 8. For the owner

| # | Question | Recommendation |
|---|---|---|
| 1 | Library | `@xyflow/react` (MIT), per §3 |
| 2 | Canvas as a **mode** of a matrix, not a second app | yes — one store, one API, one surface |
| 3 | Viewport per viewer, panel positions shared | yes |
| 4 | Ship the canvas with terminals and agents only, then add note/file/diff | yes — C2 before C3 |
| 5 | Edges wait for their own ADR | yes |
| 6 | Kanban board is refused as a Matrix mode | agree, or say so and it becomes its own plan |
