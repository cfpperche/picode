# Matrix — a live grid of agent and terminal panels (app plan)

- **Date:** 2026-09-09 (revised the same day: name, unbounded matrices, chunk loading)
- **Status:** accepted by the owner on 2026-09-09 (name and every row of §9
  confirmed in chat). **Phase 0 done the same day** — measurements and the
  seven adjustments they forced are in
  [`docs/benchmarks/2026-09-09-matrix-live-grid.md`](../benchmarks/2026-09-09-matrix-live-grid.md)
  and folded into §3, §4.1, §4.3, §4.5, §8 below. Next: phases 1 and 2 (§5).
- **Ask (owner):** a PiCode app that opens a canvas where the user lays out a
  grid of panels; everything that opens as a tab today can become a panel and
  renders **live** (the agent's TUI or its chat). v1 supports Agent CLI
  terminals, project terminals and Pi agents. The canvas is unbounded, or
  bounded so high it feels unbounded, and only the panels in view render.
  Preferred library:
  [react-grid-layout](https://github.com/react-grid-layout/react-grid-layout).
  If the apps host must grow for an app of this size, it grows alongside.
- **Name:** **Matrix** — the owner's call (Boards, then Craft, were proposed
  and refused). App id `matrix`; one layout is *a matrix*; one live surface
  is a *panel*; the only-in-view rendering is called **chunk loading** in
  docs and code, never in UI copy. None of `matrix`, `matrices` collides
  with an identifier in `web/`, `internal/` or `docs/` (checked 2026-09-09).
- **Companion work this plan creates:** one ADR (number provisional, via
  `make adr`) — *native app surfaces*; `docs/architecture/matrix.md`;
  `docs/benchmarks/2026-09-09-matrix-live-grid.md` (phase 0 output);
  `docs-site/guide/matrix.md`; `docs/changelog.d/matrix.md`.
- **Reference observed:** the owner's screenshot of Overclock 1.3.18 (a
  mission board: 3×2 panels, each a live Claude Code transcript with a header
  of identity + status + token counters and a footer of model · cwd · turn,
  plus a live site preview). Inference only; nothing was cloned.

## 1. What ships in v1, in one paragraph

A first-party app, **Matrix**, in the Apps sidebar tab. Opening it is one
main tab (`x:matrix`, `#/app/matrix/<matrixId>`). A matrix is a 12-column
grid (react-grid-layout) with as many rows as the user needs; panels are
added from a picker, dragged by the header and resized by the corner. A
panel is bound to one **terminal** (project shell or Agent CLI terminal) or
one **agent** (Pi): a terminal panel and an interactive agent panel show the
real xterm — the same instance the tab shows; a managed agent panel shows
its live conversation, read-only, with **Open** to reply in the full tab.
Only panels near the viewport are alive (chunk loading, §4.5): a matrix
with 300 panels costs the attaches of the dozen on screen. Matrices persist
in SQLite and announce themselves on the change feed, so every browser and
the backup see the same matrices. Desktop only; the phone sees a "Desktop
only" tile.

## 2. Study — what the code already decides

### 2.1 The apps host cannot host a live pane (ADR-0036)

An app today is a manifest `{id, name, icon, apiVersion}` plus a **frozen**
primitive vocabulary (list / detail / form / actions) rendered by
`web/desktop/src/components/AppSurface.jsx` from `GET /api/apps/{id}/view`.
The 2026-08-31 amendment fixes the two surfaces forever: primitives for
simple apps and sensitive actions, a sandboxed iframe for third-party bodies
in the marketplace era. Neither can carry an xterm instance or the
conversation component: both live in the host bundle and hold host state
(sockets, the `terms` registry, the feed). A first-party app whose body is a
host component is a third surface kind, and the 2026-08-31 text already says
the pipeline "manifest → grid → tab → badges → routes is surface-agnostic and
carries over unchanged". The host grows one field, not a new host (§4.1).

### 2.2 One live pane per terminal — the invariant that shapes everything

`web/desktop/src/lib/terms.js` keeps **one xterm + one WebSocket attach per
terminal id** in a module-level map; `ShellTerm.jsx` *moves* that pane's DOM
node into whichever host mounts it and parks it in `#term-park` on unmount.
The agent TUI tab uses the same key (`sh:<agentId>`, `App.jsx:2953`), so a
matrix panel and the agent tab share the instance. Consequences:

- A terminal can be **shown in one place at a time**. The desktop shows one
  surface at a time already (`hidden` tabs stay mounted), so this costs
  nothing — except one gap: the `active` effect in `ShellTerm.jsx:139-146`
  fits and focuses but does not re-append the pane to its own host. A
  terminal tab revealed after its pane was shown in a matrix would be empty.
  Phase 1 fixes that in place: **the pane follows the visible host**.
- The same terminal **at most once per matrix** (400 from the store); a
  managed agent's chat is a separate socket per mount (the server
  subscribes per connection, `internal/server/agents.go:214`), so N chat
  panels are fine.
- `web/shared/client/termSocket.js` already reattaches the **same** xterm
  after a drop (`kickTermSocket`), but its only stop switch,
  `closedByUser`, is one-way. Chunk loading needs a reversible one
  (`suspendTermSocket`, §4.5) — a small addition to the same file.

### 2.3 Geometry: tmux `window-size latest`, resize on attach

Each attach is `tmux attach-session -t =name` in a PTY sized 80×24, then the
client's `{"type":"resize"}` sets it (`internal/term/bridge.go:115-118,
220`). PiCode sets no `window-size`, so tmux's default `latest` applies: the
last client that resized decides the window. With one pane per terminal and
one visible surface, a panel's size *is* the TUI's size while the matrix is
on screen — the TUI reflows to the panel (readable text, like the Overclock
reference), and reflows back when the tab is revealed (the fit runs on
reveal, `termFit.js`). Two browsers on one terminal already behave this way
today; the matrix adds no new case. A "thumbnail" mode that never resizes
the window (`tmux attach -f ignore-size`, tmux ≥ 3.2 — we require 3.5) is a
v2 option, not a v1 need.

### 2.4 Chat for managed agents

The desktop drives one App-level socket for the **selected** agent
(`App.jsx:1434`, inline `handleEvent`), so it cannot feed N panels. The
mobile shell already solved the per-mount shape: `web/mobile/src/hooks/
useAgentSocket.js` runs the pure reducer (`lib/agentEvents.js`, present in
both apps) over `/ws/agent?agent=` and reconciles the transcript window
(`…/sessions/transcript?agent=&tail=200`). Phase 4 ports that hook to the
desktop and renders `Conversation.jsx` read-only inside the panel.

### 2.5 Persistence precedents

Pins (`docs/architecture/pins.md`): machine-scoped rows, limits that
**refuse** with 400, `PATCH` with `ifUpdatedAt` → 409, `pin.created/updated/
deleted` on the feed, a row in `TestEveryMutationAppendsAnEvent`. Per-viewer
chrome (open tabs, rail width) stays in `localStorage` under `picode-*`.
Matrices follow pins; "last matrix opened" follows the chrome. Docker shows
the route precedent for an app with its own API family (`/api/docker/*`
beside `/api/apps/docker/view`).

## 3. Library assessment — react-grid-layout 2.2.4

Facts checked on 2026-09-09 (npm registry, GitHub API, a local smoke test):

| Fact | Value |
|---|---|
| Latest / legacy tags | 2.2.4 (2026-07-29) / 1.5.4 (2026-07-29) |
| License, stars, open issues | MIT, 22 416, 59 |
| Last push | 2026-08-31 (`useContainerWidth` debounce option; `allowMobileScroll` fix) |
| v2 line | 2.0.0 on 2025-12-09: TypeScript rewrite, hooks (`useContainerWidth`, `useGridLayout`, `useResponsiveLayout`), composable config (`gridConfig`, `dragConfig`, `resizeConfig`, `dropConfig`), pluggable compactors and constraints, position strategies, `GridBackground`, ESM + CJS, `react-grid-layout/legacy` for the v1 API |
| Large layouts | `react-grid-layout/extras` ships O(n log n) compactors, advertised at 45× (vertical) for 200+ items — the case a 300-panel matrix hits on every drag |
| Breaking in v2 | `width` is required (the hook measures it); `onDragStart` fires after a 3 px threshold; callback args are read-only; README says React 18+ (peer range still `>= 16.3`) |
| Runtime deps | `react-draggable` (4.7.1 resolved), `react-resizable` (3.2.0), `resize-observer-polyfill`, `clsx`, `fast-equals`, `prop-types`; two CSS files to import (`react-grid-layout/css/styles.css`, `react-resizable/css/styles.css`) — the second means **`react-resizable` is declared as a direct dependency too**, or `web/tools/boundaries.mjs` refuses the build |
| Size | 447 KB unpacked on disk (cjs + esm + types); **shipped delta measured in phase 0: +24 KB gzip JS, +0.9 KB gzip CSS**, all in the main chunk — no lazy import needed |
| React 19 | Not stated by the project. **Verified:** renders under React 19.1.1; the grid item passes `nodeRef` to `DraggableCore`, so React 19's removed `findDOMNode` is not on the path (issue #2073 closed 2025-12-30). **Phase 0 confirmed drag and resize in a real browser with 300 items and live xterm bodies: 58–60 fps, zero console warnings.** |
| Known open issues | drag-from-outside edge cases (#2262, #2263, needs-info); a v2 rendering-lag report closed 2026-08-05 (#2240) |
| Who uses it | Grafana, Metabase, Kibana, HubSpot, Monday (dashboards of live widgets — the same shape as a matrix) |

What it gives us for free: drag by handle (`dragConfig.handle`), cancel zones
for header buttons, corner/edge resize, min/max sizes per item, vertical
compaction (dashboard feel) or `noCompactor` + overlap (free canvas),
transform positioning (fine for xterm), a stable layout array
(`{i, x, y, w, h, minW, minH}`) that serializes as-is, drop from outside
(v1.1 sidebar drag), a scaled position strategy for a zoomed container (v2
canvas). What it does **not** do: tabs inside a panel, split trees, maximize
(host feature, trivial), nested layouts, virtualization (ours, §4.5 — it
positions every child, which is exactly what lets us keep the wrappers and
swap the bodies).

| Alternative | Shape | Why not v1 |
|---|---|---|
| dockview 8.3.0 (MIT, pushed 2026-09-09) | VS Code-style docking: tabs in groups, splits, floating, popout, `toJSON` | The better fit if the product wants an *IDE layout*; the owner drew a *canvas of tiles*. Revisit if v2 asks for tabs-inside-panels. |
| gridstack 13.2.0 (MIT) | Framework-agnostic grid with a React wrapper | Same shape as RGL with a DOM-first API; RGL is React-native and what the owner evaluated. |
| `react-resizable-panels` | Split panes only | No grid, no drag-to-reorder. |
| Hand-rolled CSS grid + `@dnd-kit` | — | "Prefer popular primitives over homemade widgets" (AGENTS.md): compaction and collision are the hard part. |

**Verdict (confirmed by phase 0):** adopt react-grid-layout 2.2.4, pinned
exactly, v2 API (not the legacy wrapper), with the `extras` fast compactor
(0.06 ms vs 1.02 ms per compaction of 300 items — both free, the fast one is
headroom). Dependency lines for the PR: *react-grid-layout — mature (12
years, MIT), small API for exactly drag/resize/compaction, used by Grafana
and Kibana for grids of live widgets, measured under React 19.1; nothing in
the repo covers collision and compaction. react-resizable — its resize-handle
CSS, imported by name.* `react-grid-layout/legacy` stays the documented
fallback.

## 4. Design

### 4.1 Apps host evolution — native surfaces (ADR, number provisional)

- **Boundary:** protocol (the manifest contract gains a field) and
  persistence (new tables with events). One ADR, amending ADR-0036.
- `apps.Manifest` gains `Surface string` (`""` = primitives, `"native"`),
  `omitempty`, `APIVersion` stays 1 (same additive precedent as the
  2026-09-01 amendments). A native app implements `Manifest()` and
  `Badge()`; its `View()` answers one detail block ("Matrix opens on the
  desktop.") so any client that cannot host the surface still renders an
  honest line; `Action()` refuses.
- Client gate: `supportedApp(manifest)` in `web/shared/contracts/
  appPrimitives.js` becomes `apiVersion === 1 && (surface !== "native" ||
  nativeSurfaces.has(id))`, where each shell passes its own registry. The
  desktop registers `matrix`; the mobile registers nothing and its tile
  renders the name plus a second line, *Desktop only* (the existing
  `app-tile-unsupported` look).
- Desktop mount: `App.jsx` maps app tabs to `AppSurface` today
  (`App.jsx:2734`); a manifest with `surface: "native"` mounts the
  registered component instead, `hidden` while unselected like every other
  surface, with a **narrow `host` object** — the client twin of Go's
  `apps.Host`: `{fleet: {workspaces, freeAgents, terminals}, openTabs,
  openTab, openInteractive, revealAgent, openFileTab, feed: subscribeFeed}`.
  The app never imports App state; the object is the documented API, and
  its growth is reviewed like `apps.Host`. `openTabs` is there for one rule
  only: a terminal with an open tab is owned by the tab (§4.5).
- Icon: manifest `icon: "matrix"`, drawn by the host as a 3×3 grid glyph in
  `Icons.jsx` / `AppIcon.jsx` (the map is host-owned, ADR-0036).
- Refuse (carried from ADR-0036): third-party native surfaces — native means
  compiled into the binary; outsiders get the iframe surface when that era
  arrives.
- `ShellTerm.jsx` `active` effect re-appends the pane when its host is not
  the current one. `termSocket.js` gains `suspendTermSocket(entry)`
  (reversible stop: close the socket, keep the xterm and the control block;
  `kickTermSocket` lifts it — prototyped in phase 0: the same xterm object
  resumed with its screen intact). Both are one behaviour each, both covered
  by the browser QA (§7).

### 4.2 Data model, API, events

Migration `040_matrix.sql`:

```
matrices(id TEXT PRIMARY KEY, name TEXT NOT NULL,
       compact TEXT NOT NULL DEFAULT 'vertical',
       created_at TEXT NOT NULL, updated_at TEXT NOT NULL)
matrix_panels(id TEXT PRIMARY KEY,
       matrix_id TEXT NOT NULL REFERENCES matrices(id) ON DELETE CASCADE,
       kind TEXT NOT NULL, ref TEXT NOT NULL,
       x INTEGER NOT NULL, y INTEGER NOT NULL, w INTEGER NOT NULL, h INTEGER NOT NULL,
       created_at TEXT NOT NULL,
       UNIQUE(matrix_id, kind, ref))
```

One row per panel, not one JSON blob: a drag in a 300-panel matrix must not
rewrite and re-broadcast 30 KB, and a duplicate binding must be refused by
the database, not by a loop. `id` is a random panel id (a slot), never
derived from `ref`. Limits refuse (400, the limit named): 64 matrices,
**500 panels per matrix**, 80 characters of name, `x + w ≤ 12`, `w ≥ 4`,
`h ≥ 8`, `y ≥ 0`. `cols` is fixed at 12 in v1 (not stored); rows are
unbounded.

| Route | Does |
|---|---|
| `GET /api/matrices` | summaries: id, name, panel count, updatedAt |
| `POST /api/matrices {name}` | create empty; 400 on limits |
| `GET /api/matrices/{id}` | matrix + every panel (500 panels ≈ 50 KB, one read on open) |
| `PATCH /api/matrices/{id} {name?, compact?, ifUpdatedAt}` | 409 when the row moved on |
| `PATCH /api/matrices/{id}/layout {ifUpdatedAt, panels: [{id, x, y, w, h}]}` | the changed subset only; one transaction; 409 when stale |
| `POST /api/matrices/{id}/panels {kind, ref, x, y, w, h}` | the client places (`nextSlot`), the server validates; 400 on duplicate or limit |
| `DELETE /api/matrices/{id}/panels/{panelId}` | — |
| `DELETE /api/matrices/{id}` | cascades |

Events, appended in the mutation's transaction, one row each in
`TestEveryMutationAppendsAnEvent`: `matrix.created` / `matrix.updated`
(summary), `matrix.layout {id, updatedAt, panels: subset}`,
`matrix.panel.added {id, panel}`, `matrix.panel.removed {id, panelId}`,
`matrix.deleted {id}`. `feedReducers.js` gains the cases; `touches(ev,
["matrix"])` keys on the entity prefix, so every type above reaches the
surface. Deleting an agent or terminal does **not** touch matrices (the
store stays ignorant of the binding); the panel renders the *gone* row of
§4.4 and the user removes it.

### 4.3 Client architecture

```
web/shared/domain/matrix.js            pure: normalize, limits, nextSlot(panels, w, h),
                                       bindingState(panel, fleet), layoutDiff(prev, next),
                                       loadPolicy(rects, viewport, now, pinned) → {load, unload}
web/desktop/src/components/matrix/
  MatrixSurface.jsx                    native surface root: header (matrix switcher, Add panel,
                                       New matrix, menu: Rename/Delete), empty states, save/409
                                       flow, keyboard, the scroll container that chunk loading observes
  MatrixGrid.jsx                       react-grid-layout wrapper (useContainerWidth on the surface,
                                       fast vertical compactor, memoized children keyed by panel.id)
  Panel.jsx                            wrapper always rendered: header (face + name + status chip +
                                       actions, drag handle) + body slot; observes its own visibility
  PanelBody.jsx                        loaded: TerminalPanel | AgentChatPanel; unloaded: placeholder
  TerminalPanel.jsx                    TermSurface/ShellTerm reuse (terminal or agent TUI)
  AgentChatPanel.jsx                   phase 4: useAgentSocket + Conversation, read-only
  PanelPicker.jsx                      cmdk list of agents and terminals not on this matrix
web/desktop/src/hooks/useAgentSocket.js  phase 4: port of the mobile hook
web/desktop/src/styles/matrix.css      imported from index.css like integrations.css
```

Grid configuration: `gridConfig {cols: 12, rowHeight: 24, margin: [8, 8]}`,
`dragConfig {handle: ".mx-head", cancel: ".mx-actions", threshold: 3}`,
`resizeConfig {handles: ["se", "s", "e"]}`, `compactor:` the `extras` fast
vertical compactor, `minW: 4, minH: 8` per panel (phase 0: a 4-column panel
at 1584 px is 58 terminal columns; 3 columns would be ~43, too narrow for any
real TUI). New panels land at `nextSlot` (first free slot scanning rows,
then the bottom), size **4×14** (≈ 58×20 characters; the 4×10 the spike used
gave 14 rows, cramped). Panels are `React.memo`; `onLayoutChange` is
ignored while the surface is hidden (width 0) and while a drag or resize is
in progress; the diff against the last saved layout is what gets sent.

### 4.4 Panel binding × fleet state (decision table; every row gets a test)

| Kind | State from the fleet / feed | Panel body (when loaded) | Actions |
|---|---|---|---|
| terminal (shell) | running | live xterm (`sh:<id>`) | Open · Maximize · Remove from matrix |
| terminal (Agent CLI) | stopped | `TermSurface`'s own stopped state: "This CLI terminal is stopped." + Resume last session / Start from Agent CLIs | same |
| terminal | id missing from the fleet | "That terminal is gone." + Remove | Remove |
| agent | interactive, running | live TUI xterm | Open · Maximize · Remove |
| agent | managed, running | phase 4: live read-only conversation; until then "Managed agent — open to read." + Open | Open · Maximize · Remove |
| agent | stopped | "Agent is stopped." + Run (`POST /api/agents/{id}/managed/start` or `…/open` per its mode) | Open · Remove |
| agent | id missing | "That agent is gone." + Remove | Remove |
| agent | mode flips while on the matrix (`agent.*` feed) | body swaps in place; the pane is parked, not disposed | — |

Header chip vocabulary is the sidebar's (ADR-0062): **Working**, **Needs
you**, **Ready**; CLI badge from `TerminalCliBadge`; agent status from the
`agent.*` feed view. The chip is fed by the feed, so it is right for
unloaded panels too. No pixel inference.

### 4.5 Chunk loading — only panels near the viewport are alive

The owner's ask is an unbounded matrix. The cost of a matrix is not its
panels but its **attaches**: every live terminal panel is one WebSocket and
one `tmux attach` process; every chat panel is one agent socket. The matrix
therefore renders every panel's *wrapper* (react-grid-layout needs each item
as a child to resolve collisions and compaction; a wrapper is one `div` and a
header) and mounts a *body* only near the viewport — the same idea as a
game engine loading the chunks around the player.

| Situation | Behaviour |
|---|---|
| Panel rect enters the scroll viewport ± one viewport height (`IntersectionObserver` on the surface's scroll container, `rootMargin: 100% 0px`) and **stays there 300 ms** | **load**: mount the body; attach (or reconnect). Phase 0: without the dwell a 4 s fly-over of 300 wrappers mounted 341 bodies; with it, 6 — while a 1 000 px/s scroll still loaded 27 |
| Panel rect leaves that margin | **unload after 5 s** if still outside; a pending load is cancelled at once (phase 0: sockets untouched across a fast pass; all twelve suspended 8 s after parking out of view) |
| Panel is being dragged, resized, focused or maximized | pinned: never unloads |
| Matrix tab not selected | every panel unloads after the same 5 s: a hidden matrix holds no attaches of its own |
| Terminal also open as a tab (`host.openTabs` has `t:<id>` or the agent's tab) | the tab owns the attach; unloading only parks the pane |
| Terminal exists only on matrices | unload = `suspendTermSocket` (socket closed, xterm kept with its scrollback); load = `kickTermSocket` through `ShellTerm`'s existing "reattach the same instance" path |
| More than 24 suspended instances, or one suspended for 10 min | LRU disposal (`closeTerm`); a later load builds a fresh xterm — tmux still holds the screen |
| Managed-agent panel unloads | agent socket closed, items dropped; load reconnects and refetches the tail (`useAgentSocket` already does this on connect) |
| Unloaded body | a muted placeholder with the feed's last state ("Working · 2 min"); the header stays live |
| Status of 500 unloaded panels | from `terminal.state` / `agent.*` feed events; zero attaches |

Bound: what fits in three viewport heights. On a 1440p screen with 4×14
panels (≈ 440 px tall) that is about 6 panels per screen, so ≤ 18 live
attaches for any matrix size (phase 0 measured nine live `top` panels at
≈ 4 % renderer CPU, 60 fps, no long tasks). `loadPolicy` is pure (rects,
viewport, clock, pinned set → load and unload sets) and unit-tested for the
dwell and hysteresis rows; the observer only feeds it rects. A "frozen
frame" placeholder (the xterm's last screen as text, captured before
suspending) is v1.1.

### 4.6 Pane ownership and geometry (decision table)

| Situation | Behaviour |
|---|---|
| Matrix tab selected; the same terminal's tab open but hidden | pane lives in the panel; tmux window = panel size |
| Terminal tab revealed | `ShellTerm` re-claims the pane and fits; the panel's host is empty while the matrix is hidden |
| Matrix revealed again | the visible panels load and re-claim their panes |
| Maximize inside the matrix | layout untouched; panel takes the surface; fit; Esc restores |
| Same terminal on two matrices | allowed — one visible at a time |
| Same terminal twice on one matrix | 400 "already on this matrix" (the `UNIQUE` row) |
| Another device resizes the terminal | existing `window-size latest` behaviour; nothing new |
| Focus | one focused panel per matrix (click / Enter); only it calls `term.focus()`; Esc returns focus to the panel chrome; arrows move between panels, scrolling the viewport (which loads them) |

### 4.7 Saving the layout (decision table)

| Trigger | Action |
|---|---|
| drag or resize stop, surface visible | `layoutDiff` → `PATCH …/layout` debounced 500 ms with `ifUpdatedAt`; a compaction that shifted 200 panels sends 200 rows once |
| surface hidden or width 0 | no save |
| 409 | refetch, replace, toast "Matrix changed elsewhere — reloaded." |
| `matrix.layout` / `matrix.panel.*` from another client while dragging | applied after the gesture ends |
| Rename / Delete | immediate; Delete confirms (Radix alert dialog) and moves the tab to the next matrix or the empty state |

### 4.8 Copy and empty states (one line + one action)

- No matrix: "No matrix yet. A matrix shows many agents and terminals side
  by side, live." — **New matrix**.
- Empty matrix: "Add your first panel." — **Add panel**.
- Picker with nothing left: "Everything is already on this matrix." /
  "No agents or terminals yet — create one from the sidebar." — **Close**.
- Phone tile: name + *Desktop only*; `#/app/matrix` on the phone: "Matrix
  is a desktop tool." — **Back**.

### 4.9 Geometry choice: rows without end now, a 2D canvas later

v1 is a 12-column grid whose rows never end, vertically compacted: the
page grows as panels are added, scrolls vertically, and every panel keeps a
readable width. That is the react-grid-layout shape the owner evaluated
(the toolbox demo) and what chunk loading is tuned for (one scroll axis).

A **2D free canvas** — pan in both axes, zoom out to survey, zoom in to
work — is the v2 matrix mode: `noCompactor`, a width larger than the
viewport with horizontal scroll, `createScaledStrategy` for the zoom. It
needs its own spike: xterm under a CSS `scale()` (mouse mapping, canvas
blur), and drag math under scale. Nothing in v1's data model blocks it
(`compact` is already a column; `x` grows past 12 when `cols` becomes a
column).

### 4.10 Non-goals for v1

File, tree, git and app panels; tabs inside a panel; the 2D canvas; a
per-panel composer for managed agents; drag from the sidebar; matrix
templates; the inspector following the focused panel (it keeps its last
anchor, the rule apps already have); a badge on the Matrix tile.

## 5. Phases (one branch, one session each — ADR-0105)

| # | Branch | Delivers | Gate |
|---|---|---|---|
| 0 | `feat/matrix-spike` | **Done 2026-09-09.** RGL 2.2.4 under React 19: drag, resize, 300 wrappers, nine live xterm bodies, the load/unload prototype; bundle, CPU and churn measured; GO for react-grid-layout. Note: `docs/benchmarks/2026-09-09-matrix-live-grid.md`. Only the note and this plan merged. | owner reads the note |
| 1 | `feat/apps-native-surface` | ADR; `Manifest.Surface`; `supportedApp` gate + tests; desktop native mount + `host` object; phone tile; `ShellTerm` re-claim; `suspendTermSocket` | `make close`; browser QA of tab↔matrix hand-off and suspend/resume |
| 2 | `feat/matrix-store` | migration 040, store + events + invariant rows, handlers with limits/409, OpenAPI regen, feed reducers | `make close` |
| 3 | `feat/matrix-surface` (two sessions) | grid, wrappers + chunk loading from day one, terminal + TUI bodies, picker, switcher, empty states, save/409, keyboard, `matrix.css`, visual review, guide page, changelog fragment, architecture file | `make close`; visual card |
| 4 | `feat/matrix-chat-panel` | desktop `useAgentSocket`, read-only conversation body, Needs-you chip | `make close` |
| 5 | v1.1 (owner's pick) | frozen-frame placeholders, maximize polish, drag from sidebar (`dropConfig`), quick reply line, inspector follows focus, tile badge, matrix templates | — |
| 6 | v2 | 2D canvas mode (pan, zoom, `noCompactor`), thumbnail attaches (`ignore-size`), other tab kinds as panels | its own plan |

Phases 1 and 2 can run in parallel worktrees (no shared files); 3 needs
both. Chunk loading is not deferrable: `loaded` is part of the panel model
and the suspend API shapes `terms.js` — retrofitting it later would touch
every panel component twice.

## 6. Why not simpler (the trades this plan makes)

| Simpler option | Why refused |
|---|---|
| One JSON blob per matrix | a drag rewrites and broadcasts the whole matrix; duplicates checked in code, not by the database |
| Render every panel live, cap at 32 | the owner's ask is an unbounded matrix; 32 attaches per browser is already the server's upper comfort, 300 is not |
| Keep hidden matrices attached like hidden tabs | a tab holds one attach; a matrix holds hundreds — the tab rule does not scale |
| `localStorage` layouts | not shared across browsers, not backed up, not on the feed; the repo keeps only ephemeral chrome there |
| Load bodies by scroll position math without an observer | the observer is the browser's own intersection test and survives resizes, rail toggles and zoom for free; the policy stays pure and tested |

## 7. Acceptance

- Go: `internal/store/matrix_test.go` (CRUD, limits, uniqueness, subset
  layout patch, 409, cascade, event rows), `internal/server/matrix_test.go`,
  `apps_test.go` (surface field).
- JS: `matrix.test.js` (nextSlot, layoutDiff, limits, bindingState per §4.4
  row, `loadPolicy` per §4.5 row), `appPrimitives.test.js` (gate),
  `termSocket.test.js` (suspend then kick reattaches the same entry),
  `feedReducers.test.js`, `routes.test.js`.
- Browser, on `scripts/qa-scratch.sh` with `agent-browser --session matrix`:
  build a matrix of a pi agent TUI + two shells; drag, resize, reload —
  layout persists; open the agent tab — pane re-claimed, tmux reports the
  tab's size (`tmux display -p '#{window_width}x#{window_height}'` on the
  scratch socket, exact session name from the fixture's API); back to the
  matrix — panel size again; add 40 shells (fixture), scroll to the bottom
  and back — `ss`/`lsof` on the scratch daemon shows ≤ 27 term sockets at
  any point and the same xterm instance (`window.__picodeTerms`) after a
  suspend/resume; delete a terminal — gone row; two browsers — the second
  follows the feed; 409 path by editing from both.
- Visual review card (5 questions) on: empty states, a 6-panel matrix, a
  40-panel matrix scrolled mid-way (loaded and placeholder bodies side by
  side), picker open (`window.__picodeOverlayAudit()` ok), maximize, dark
  and light.
- Budget (phase 0 set the bar): nine live TUI bodies on screen at ≤ 10 %
  renderer CPU and 60 fps (measured ≈ 4 % and 60 fps), dragging ≥ 55 fps
  with 300 wrappers (measured 58–60), and a 4 s fly-over of the whole matrix
  loading fewer than ten bodies (measured 6 with the 300 ms dwell).

## 8. Risks

| Risk | Mitigation |
|---|---|
| v2 API is nine months old with a bug tail | exact pin; legacy entry as fallback; phase 0 exercises drag, resize, drop, 300 items |
| attach churn while scrolling | 300 ms load dwell + one-viewport margin + 5 s unload hysteresis; measured in phase 0 (6 loads per fly-over) |
| `tmux attach` spawn latency on load | one process per load, tens of ms; the placeholder shows the feed state meanwhile — no blank flash |
| xterm reflow storms while resizing a panel | `termFit` already debounces 150 ms; measure; if needed fit on `onResizeStop` only |
| pane hand-off regressions in `ShellTerm` | one behaviour change, one browser test, both tab kinds (terminal tab hidden-not-unmounted; agent TUI unmounted) |
| a `matrix.layout` event after a big compaction is large | bounded by the panel cap (500 rows ≈ 50 KB), rare (removing a top panel), and still smaller than one blob per drag |
| bundle growth | measured in phase 0: +24 KB gzip — below the lazy-import threshold |
| attach teardown after a page reload waits for the bridge's `pongWait` (60 s) | existing behaviour, written down in the study; unload closes the socket explicitly, so its teardown is immediate |

## 9. Decisions (taken by the owner in chat, 2026-09-09)

| # | Question | Decision |
|---|---|---|
| 1 | Name | **Matrix** (app, id `matrix`) · *a matrix* (one layout) · *panel*; Boards and Craft refused |
| 2 | Where matrices live | the store (shared, backed up, on the feed); only "last matrix opened" per browser |
| 3 | Managed-agent panels in v1 | read-only conversation + Open; the quick reply line is v1.1 |
| 4 | Library | react-grid-layout 2.2.4 pinned, `extras` fast compactor; dockview only if v2 wants tabs-inside-panels |
| 5 | Limits | 64 matrices, **500 panels per matrix** (feels unbounded; the cap bounds one read and one event), name 80 chars, min panel 3×6 cells |
| 6 | Third surface kind in the manifest (`surface: "native"`, first-party only) | yes — the proposed-API tier ADR-0036 describes, now with a real consumer |
| 7 | Geometry | v1 rows without end (12 columns, vertical compaction, chunk loading on one axis); the 2D pan-and-zoom canvas as the v2 matrix mode after its own spike |

Changing any row is the owner's call again (AGENTS.md, non-negotiable 6).

## 10. Next step — phases 1 and 2

Phase 0 is done (§5). Two worktrees can start now, in parallel and in fresh
terminals: `make worktree NAME=apps-native-surface` (§4.1: the ADR, the
manifest field, the gate, the host object, the `ShellTerm` re-claim and
`suspendTermSocket` with its test) and `make worktree NAME=matrix-store`
(§4.2: migration 040, store, events, handlers, OpenAPI). Phase 3 starts when
both have fast-forwarded into `main`. The spike's throwaway component is not
in the tree; the study names the shape it had (`window.__mx` counters,
`IntersectionObserver` with `rootMargin: 100% 0px`, a 300 ms load timer and a
5 s unload timer per wrapper) for whoever writes `loadPolicy`.
