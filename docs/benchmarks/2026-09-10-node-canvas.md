# Study: a canvas of live terminals on @xyflow/react 12.11.6 (Matrix v2, phase C0)

- **Date:** 2026-09-10
- **Plan:** [`docs/plans/matrix-canvas.md`](../plans/matrix-canvas.md) §5 asked this
  spike whether React Flow v12 can host a plane of live xterm panels under
  React 19: 500 nodes, nine live terminal bodies, the §4.3 still/live rule,
  chunk loading under a canvas transform, `onlyRenderVisibleElements` both
  ways, the minimap, the bundle, and — the plan's central risk (§7) — xterm
  click and selection at zoom 0.8 and 1.0. The nodeterm benchmark table this
  study rests on is the plan's §2 and is not repeated here.
- **Method:** a throwaway `MatrixCanvasSpike` surface in the desktop bundle of
  worktree `feat/matrix-canvas-spike` (never merged; reverted in the same
  branch), gated by `?spike=canvas` in `main.jsx`, on a scratch instance
  (`scripts/qa-scratch.sh`, port 8473, UI embedded), driven by
  `agent-browser --session matrix-canvas-spike` (headless Chromium,
  1600×1200). 500 nodes in a 25 × 20 plane, node 480 × 340 px at a
  520 × 380 pitch (plane 13 000 × 7 600 px); the nine nodes of the 3 × 3
  block at the origin bound to nine terminals created through the scratch
  API with `top -d 1` running in each; the other 491 render a 24 × 80
  monospace `<pre>` so a loaded body costs real DOM. Chunk loading is the
  shipped pure `loadPolicy` (300 ms dwell, 5 s hysteresis) behind an
  `IntersectionObserver` rooted on the `.react-flow` pane with a settable
  `rootMargin`. Counters were read from `window.__mxc`; CPU from `/proc` of
  the page's renderer; terminal geometry from tmux. Receipts are quoted as
  measured; screenshots stayed in `var/screenshots/` (never committed).
- **Stack:** React 19.1.1 · @xyflow/react 12.11.6 (`zustand` 4, `classcat`,
  `@xyflow/system` 0.0.82) · @xterm/xterm 6.0.0, **DOM renderer** · tmux 3.6
  · node 24.11 · WSL2 6.6.

## Verdict

**GO on `@xyflow/react`, with one rule the plan does not yet have:** a live
terminal is only correct at **zoom exactly 1.0**. Everything else the plan
assumed held — 60 fps at 500 nodes, `nodrag`/`nowheel` work, chunk loading
bounds the attaches under the transform, the still costs 0.02 ms and never
flickers, `NodeResizer` resizes a live pane and fires its end once — but
xterm's pointer→cell mapping ignores the CSS transform, so **§4.3's 0.8
threshold is wrong**: at 0.8 a click lands up to 10 columns and 3 rows away
from where the user pointed.

## Results

| Question | Measured | Verdict |
|---|---|---|
| 500 nodes under React 19 | 500 wrappers mount, 0 console errors, 0 page errors over the whole session; idle with nine live `top` panes **60.0 fps**, 0 long tasks, worst frame gap 16.8 ms | GO |
| Panning | 12 000 px in 4.0 s (3 000 px/s) **60.0 fps** / 0 long tasks; the same in 1.5 s (8 000 px/s) **60.0 fps**; 6 000 × 3 800 px in 3.0 s **60.0 fps**; 4 000 × 2 500 px in 2.0 s at zoom 0.5 with 500 wrappers and 169 stills mounted **60.0 fps** | GO |
| Zooming | 1.0 → 0.2 over 2.0 s: **58.5 fps**, worst gap 33.4 ms, 0 long tasks | GO |
| Dragging one node | 24-step drag at zoom 1: **60.0 fps**; the same at zoom 0.5 with 500 wrappers and 169 stills: **60.0 fps** | GO |
| Marquee of 20 nodes | selection gesture **60.0 fps** either way; **dragging the 20** is the only expensive gesture: **31.2 fps** (worst 50 ms, 0 long tasks) in one run and **18.2 fps** (17 long tasks, 1 252 ms) in a slower one with `onlyRenderVisibleElements` **off** — **60.0 fps, 0 long tasks** with it **on** | see below |
| `onlyRenderVisibleElements` | on: 16 nodes rendered at rest instead of 500, 49 instead of 500 at zoom 0.5, and the 20-node drag goes 31 → 60 fps. **But** it unmounts the node — and with it the live body: 3 fast pan round trips churned **54** terminal-body swaps (9 terminals × 6 legs, each a socket suspend + kick) against **0** with it off | **off** — see §"culling" |
| Minimap at 500 nodes | 500 `.react-flow__minimap-node` rects render either way (culling does not reach the minimap). Pan at zoom 0.5: **60.0 fps** with, **60.0 fps** without. 20-node drag: **18.2 fps** with, **19.0 fps** without | free — keep it |
| Bundle delta (desktop, `vite build`) | eager: JS **+187 291 B raw / +60 761 B gzip**, CSS **+15 868 / +2 552** (63.3 KB gzip total). Lazy-imported: the main chunk moves **+519 B raw / +205 B gzip** and the library lands in its own chunk of **46 181 B gzip** JS + **2 687 B gzip** CSS | **lazy-import it** |
| Chunk loading under the transform | at rest (zoom 1, 1600 × 1200, `rootMargin: 100%`): 49 loaded, **9 attached**, 0 suspended. After a 8 000 px/s pass across the plane: 175 loaded during the pass, and at the far end +10 s **0 attached, 9 suspended**, still 9 xterm instances. Back at the origin +8 s: 49 loaded, **9 attached, the same instances** | GO |
| Chunk loading under zoom | zoom 0.2: **500 loaded, all still, 0 attached, 9 suspended** — the band is a *screen-space* rule, so zooming out multiplies the plane area it covers; what bounds the attaches at low zoom is the still rule, not the band | see §"the band is screen-space" |
| The still/live threshold | zoom **0.79**: 0 live, 64 still, **0 attached**; zoom **0.80**: 64 live, **9 attached** — the rule fires exactly, and the same nine sockets suspend and are kicked back | works, but see the zoom rule below |
| Still capture | a full screen (83 × 26 = 2 158 cells) captured from `term.buffer.active`: **0.02 ms** (200 runs in 4.0 ms); 54 × 16 ≈ 0.01 ms | free |
| Still swap flicker | no blank frame — the still body and the live body swap in one commit, and one rAF later the body already carries its 775 characters. **But the still is one crossing late**: React renders the still body *before* it runs the live body's unmount cleanup, so on a fresh page the first crossing paints `(no still captured yet)` and only the second shows the screen (measured: boot 0 stills → first crossing 9 captures but the body reads the empty state → second crossing shows `top - 13:31:17 up 5 days, 4:06, 1 user, load averag`) | fix by capturing **before** the flip |
| Still legibility | **zoom 0.5**: the still is sharp but small — readable when magnified, marginal at native size; headers (name, `STILL`) read straight off the frame; ~45 cards on screen. **zoom 0.2**: the body is an even grey texture that never resolves into characters even magnified 6×, and the headers do not resolve either; ~300–400 cards on screen | a still earns its place down to ~0.4, not to 0.2 |
| Live legibility under the transform | at **0.8** the live xterm is **as crisp as at 1.0** — no bitmap smear, no uneven stroke weight (the DOM renderer scales as text, not as a raster); `%Cpu(s):  3.1 us,  5.2 sy,  0.0 ni, 90.2 id,  0.0 wa,` read off a card at 0.8 | rendering is not the risk; the pointer is |
| `nodrag` | a body drag-select left the node's position at (520, 0) unchanged at zoom 1.0, 0.8, 0.5 and 1.5 | GO |
| `nowheel` | a wheel over a body inside `.nowheel` left the zoom at **1.000**; the same wheel over `.react-flow__pane` took it **1.0 → 0.758** | GO |
| Header drags the node | header drag moved n1 (520, 0) → (664, 80), `onNodeDragStart`/`onNodeDragStop` once each | GO |
| Keystrokes reach the right pane | clicking cvs2's body and typing `echo CANVAS-KEYS-OK` echoed in **cvs2 only**; cvs1, cvs5 and cvs9 read empty | GO |
| `NodeResizer` on a live terminal | node 480 × 340 → 720 × 516; the pane refit **54 × 16 → 83 × 26** through the existing fit path and the tmux pane followed (`83x26`); **`onResizeStart` and `onResizeEnd` fired once each**; 0 errors | GO |
| `snapToGrid` 8 px | on: a drag lands on (584, 40) — both multiples of 8. Off: (648, 78), 78 mod 8 = 6. Resizing does **not** snap the height (516) | works for drag, not for resize |
| Renderer CPU | nine live `top -d 1` at 500 nodes, zoom 1: **4.2 / 6.8 / 7.2 / 10.7 / 14.0 %** of one core across five 12–15 s samples (noisy on WSL2). The same page at zoom 0.2 with 500 stills and 0 attaches: **0.0 %** | the still rule is the whole saving |
| **xterm under the transform** | **broken at every zoom ≠ 1** — the table below | **the plan's rule must change** |

## The central risk, measured: xterm's pointer mapping ignores the transform

Method: `__mxc.probe(term, col, row)` returns the page point at the centre of
a named cell, computed from `.xterm-screen`'s **transformed**
`getBoundingClientRect()`. Clicking that point should land on that cell.
Three independent read-outs agree.

**1. What xterm computes** (`_mouseService.getCoords` on the probe point):

| Cell aimed at | zoom 1.0 | zoom 0.8 |
|---|---|---|
| (1, 1) | (1, 1) | (1, 1) |
| (3, 5) | (3, 5) | (2, 4) |
| (12, 5) | (12, 5) | (10, 4) |
| (27, 8) | (27, 8) | (22, 6) |
| (50, 15) | (50, 15) | (40, 12) |

**2. What the terminal reports** (real mouse clicks, SGR mouse reporting on —
`printf '\033[?1000h\033[?1006h'; cat -v`):

| Cell clicked | zoom 1.0 | zoom 0.8 |
|---|---|---|
| (20, 8) | `^[[<0;20;8M` | `^[[<0;16;7M` |
| (50, 15) | — | `^[[<0;40;12M` |
| (1, 1) | — | `^[[<0;1;1M` |

**3. What a real drag-selects** (`term.getSelection()`, tmux `mouse off` so
xterm owns the selection; the pane holds nine rows of
`R<n>0123456789abcdefghijklmnopqrstuvwxyz+++++`):

| Drag | Selection |
|---|---|
| (3,5) → (12,5) at **1.0** | `0123456789` — exactly the ten characters aimed at |
| (3,5) → (12,5) at **0.8** | `0123456` — 7 characters |
| (3,5) → (12,5) at **0.5** | `30123` — 5 characters, one row too high |
| (3,5) → (12,5) at **1.5** | `23456789abcde` — 13 characters, one column late |
| (3,2) → (40,7) at **1.0** | rows 2–7, ending at column 40 |
| (3,2) → (40,7) at **0.8** | rows 2–6, ending at column ~32 |
| (3,2) → (40,7) at **1.5** | rows 3–10 |

**Mechanism.** xterm divides the pointer's offset inside the *transformed*
rect by the *untransformed* cell size, so the mapped cell is
≈ `cell × zoom`. The error is zero at the pane's top-left corner and grows
linearly with distance from it: at zoom 0.8 on a 54 × 16 pane it reaches
10 columns and 3 rows. React Flow documents this class of bug ("scale
computed positions by 1/zoom") but xterm exposes no scale factor to set.

**It reaches the product on both paths.** PiCode ships tmux `mouse on` by
default (`internal/termopts/termopts.go`, "Mouse belongs to the terminal"),
so those wrong coordinates travel to tmux as SGR reports — tmux copy mode
and every mouse-aware TUI (top, vim, a CLI's own dialogs) are pointed at
the wrong cell. With `mouse off` it is xterm's own selection, wrong by the
same factor. Keyboard is unaffected; rendering is unaffected (the DOM
renderer scales as text, not as a bitmap).

## What the spike changes in the plan

1. **Live only at zoom 1.0 — the 0.8 threshold is too loose** (§4.3, §7).
   A live pane at 0.8 renders fine and takes keys fine, but its pointer
   lies. The rule becomes: **live and interactive at zoom 1.0; live but
   pointer-inert between 0.8 and 1.0** (the body takes `pointer-events:
   none`, and a click on it runs the plan's "snap to 1 on engage" first,
   then hands the pointer over); **still below 0.8**. The plan already had
   the snap; the spike says the snap is not an ergonomic nicety, it is the
   correctness condition, and it must also cover `maxZoom` 1.5 — zooming
   *in* breaks the mapping exactly as zooming out does.
2. **Add hysteresis to the zoom threshold.** Crossing 0.79/0.80 suspends
   and kicks nine sockets. Live above 0.8, still below 0.75, so a viewer
   parked at the boundary does not thrash the attaches.
3. **`onlyRenderVisibleElements` stays off** (§5 asked which). It is the
   cheaper renderer — 16 nodes instead of 500 at rest, and it is the only
   thing that made the 20-node drag 60 fps — but it unmounts nodes with no
   dwell and no hysteresis, and a node's unmount is a live terminal's
   unmount: 3 fast pan round trips churned **54** body swaps (54 socket
   suspends + kicks) against **0** with it off. Our own band already does
   the culling that matters (the attaches); React Flow's would undo the
   300 ms dwell and 5 s hysteresis the phase-0 study bought. If the
   20-node drag ever hurts in practice, the fix is to bound the *mounted
   body* count, not to let the library unmount the wrapper.
4. **The band is screen-space; the plan's §4.3 must say so.**
   `IntersectionObserver` does account for the transform — verified — but
   that means a fixed `rootMargin` covers `1/zoom²` times more plane as you
   zoom out: at 0.2 the band held **all 500** nodes. Attaches stayed at 0
   only because the still rule had already taken over. Either scale the
   margin by zoom, or cap the loaded set; the plan should not claim the
   band alone bounds anything below zoom 1.
5. **`rootMargin` must gain its horizontal half.** The grid uses
   `100% 0px` because it only scrolls vertically. Measured at rest, zoom 1,
   1600 × 1200: `100%` → 49 loaded, `100% 0px` → 28, `0px` → 16. All three
   bound the attaches; the canvas wants `100%`.
6. **Lazy-import the canvas host** (§7 said "measured in C0"). Eager it is
   **+63.3 KB gzip** on the main chunk — 2.6× react-grid-layout's
   +24.3 KB and above the 60 KB threshold the phase-0 study used. Lazy it
   costs the main chunk **205 B gzip** and puts 48.9 KB gzip in a chunk
   only a canvas user fetches.
7. **The minimap is free — and unstyled.** No measurable frame cost at 500
   nodes. But the default `<MiniMap />` paints a near-white panel on
   PiCode's dark surface and the default `<Controls />` sits on top of the
   nodes; both need tokens and a placement decision in C2, not a
   measurement.
8. **`NodeResizer` is the resize story** and its `onResizeEnd` fires once,
   so the save path's debounce is safe as designed. It does not honour
   `snapGrid` for size — C2 rounds `w/h` to the 8 px unit itself before
   the PATCH.
9. **Capture the still *before* the swap, not at unmount** (§4.3 says
   "refreshed when a live pane unmounts"). React renders the still body
   before it runs the live body's cleanup, so a capture in the cleanup is
   always one crossing late — the first crossing on a fresh page paints an
   empty still. At 0.02 ms a capture, the rule becomes: capture every live
   terminal when the zoom crosses down, then flip the mode; and refresh
   opportunistically while live. The plan's "stamped with its age" stays.
10. **Below ~0.4 a still is texture, not text** (§4.3 sets `minZoom` 0.2).
   Measured: at 0.5 the still is sharp but small — legible magnified,
   marginal at native size; at 0.2 it never resolves into characters and
   neither do the headers. `minZoom` 0.2 is still right for orientation,
   but the panel needs a third body below ~0.4: a name-plate (face, name,
   status colour) sized to the panel, not a text still. Otherwise the
   plan's "a 200-panel matrix is unnavigable without a minimap" is true of
   the plane itself.
11. **The 0.8 live band is a rendering win, not a mouse one.** Under the
   transform the DOM renderer stays crisp — a live pane at 0.8 reads as
   well as at 1.0 — which is exactly why the pointer rule in item 1 has to
   be explicit: nothing on screen tells the user the clicks are lying.

## The one-engine question (plan §3)

`snapToGrid` with `snapGrid={[8, 8]}` works: a drag lands on multiples of 8;
with it off the same drag lands at 78 (mod 8 = 6). So the *placement* half
of grid mode is one prop. What react-grid-layout does that React Flow plus
snapping does **not**, measured or read on this spike:

| Grid behaviour | React Flow |
|---|---|
| automatic vertical compaction | none — the plan's pure pack function behind **Tidy** (already §3) |
| **collision resolution** — RGL never lets two items overlap and pushes the ones in the way | **none**: a dragged node was left overlapping its neighbour (n1 at 776, 328 over n0 at 368, 280 sized 720 × 516). Grid mode would need a push/refuse pass of our own |
| **reflow to container width** — 12 columns of variable pixel width, so a narrower window rescales every panel | none: a canvas has absolute coordinates. This is the difference the mode transform in §4.1 hides, and the one a user would notice on a laptop |
| per-item `minW`/`minH` and edge handles | `NodeResizer` covers both (`minWidth`/`minHeight`, 8 handles), verified against a live terminal |
| keyboard placement | neither has one — the Matrix's arrows are ours (`neighborPanel`), so nothing is lost |

**Recommendation:** keep §3's two-step plan unchanged, and note that the
end state "one library, two modes" now has a named price: compaction *and*
collision *and* width reflow are ours to write, not one function. That is
still a few hundred lines against the recurring cost of two drag/resize/
selection models — but it is the decision the owner takes after using the
canvas, exactly as §3 says, and C0 found no React Flow blocker that would
force "two libraries" instead.

## What the spike component looked like (for C2 to rebuild)

One file, `web/desktop/src/components/matrix/MatrixCanvasSpike.jsx`
(~490 lines), plus three lines in `main.jsx` (`?spike=canvas` renders it
instead of `<App />`). Reverted before this branch closed.

- **Nodes:** a memoised array of 500 `{id, type: "panel", position, style:
  {width, height}, data}`; `defaultNodes` (uncontrolled) — with `nodes=`
  and no `onNodesChange` a production build drags nothing and says nothing.
- **The body rule:** a module-level bus (`loaded` set, `zoomHigh` flag,
  `modes` map, per-id subscriber sets) that every node reads through
  `useSyncExternalStore`, so a zoom-band flip re-renders only the nodes
  whose mode actually changed. `mode ∈ {live, still, off}` drives
  `TerminalPanel` / a `<pre>` still / a placeholder.
- **Stills:** captured in the live body's unmount cleanup from
  `term.buffer.active` (`getLine(viewportY + i).translateToString(true)`),
  timed into `bus.captures`, and sampled one rAF after each swap into
  `bus.swaps` (`{id, mode, hadStill, len}`).
- **Chunk loading:** a `CanvasLoader` — the shipped `ChunkLoader` with a
  settable `rootMargin` — rooted on the `.react-flow` element through the
  `ReactFlow` ref.
- **Gates:** `.mxc-head` is the drag handle by omission, `.mxc-body` carries
  `nodrag nowheel`, header buttons carry `nodrag`.
- **`window.__mxc`:** `stats()` (zoom, rendered, observed, loaded, modes,
  instances, attached, suspended, observations, errors), `setOnlyVisible`,
  `setMiniMap`, `setSnap`, `setMargin`, `forceLive`, `setViewport`,
  `zoomTo`, `fpsStart`/`fpsStop`, `panRun(dx, dy, ms)`, `zoomRun(to, ms)`,
  `captures()`, `swaps()`, `still(id)`, `counters()` (drag/resize/selection
  ends), `probe(term, col, row)`, `buf(term)`, `selection(term)`,
  `sizeOf(term)`, `nodeRect(id)`, `nodePos(id)`, `selectedIds()`, `log`.

## Receipts and QA notes

- Counters: `window.__mxc` above; CPU: sum of `utime+stime` deltas from
  `/proc/<pid>/stat` over the window for every `--type=renderer` process;
  terminal geometry: `tmux display -p -t '=picode-sh-<id>:0.0'
  '#{pane_width}x#{pane_height}'`.
- **`agent-browser mouse wheel` dispatches at (0, 0)**, not at the last
  `mouse move` — the first `nowheel` reading was a false negative. Wheel
  tests need a `WheelEvent` dispatched with real `clientX/clientY`.
- **A React Flow `nodes=` prop with no `onNodesChange` silently drags
  nothing** in a production build (the warning is development-only).
- **PiCode terminals run with tmux `mouse on`**, so xterm forwards clicks
  instead of selecting; `getSelection()` reads empty until the session's
  `mouse` option is off (or Shift is held). Not a spike artefact — it is
  the shipped default, and it is why the mapping bug matters.
- `tmux send-keys` inside a `while read` loop needs `< /dev/null`; a
  terminal's tmux session exists only after its first attach; the scratch
  serves the UI embedded, so a UI change needs `make web` + a rebuilt
  binary before the page can see it.

## Chrome, a second look (2026-09-11)

Item 7 above left the chrome half-answered: it said the minimap and the
default `<Controls />` need "tokens and a placement decision", and C2 gave
them both. It did not ask the larger question, which the owner did on
2026-09-11 — *"menus estilo overlays e não aquela barra fixa no topo"* —
with nodeterm's own canvas as the reference: **no bar over the plane at
all**. Same source and same licence as §2 of the plan (BUSL-1.1, public
material only: no code, no assets, no copied strings), so everything below
is inference from their published screenshots and README.

| What their canvas shows | What PiCode took |
|---|---|
| the plane edge to edge, no header over it | **taken**. The `.ft-head` bar is gone from `CanvasSurface.jsx` — the stage is the plane. The icon and the title were the tab strip's words repeated, and **Close** was the tab's own × repeated |
| a small floating control top-left | **taken**, as `.cv-chrome`: the canvas switcher, **Add panel**, and a `⋯` menu carrying New canvas, Tidy, Rename, Delete and Close tab |
| a floating toolbar bottom-centre (undo/redo/save/fit/zoom) | **taken in part, moved**: our zoom cluster and **Fit** already floated bottom-right beside the minimap, and moving them to the centre would have bought a second corner for nothing. Undo/redo and Save have no counterpart to take: a canvas saves the changed subset 500 ms after a drag (ADR-0108), and a removal's undo is the toast that already offers it |
| a minimap bottom-right | already shipped in C2; unchanged |
| a status pill bottom-left | **refused**. Theirs names the session; ours would name the canvas the switcher already names, on a corner React Flow's attribution already holds. A pill that repeats a control two corners away is chrome about chrome |
| translucent, blurred cluster grounds | **refused**. A cluster here can sit over a live terminal, and a scrim over a terminal lets its own text through the chrome. Both clusters are opaque `--bg-elevated` with a `--border` hairline and the minimap's shadow, which is also the only version that reads in the light theme |

**The adaptation, in one line:** take the plane edge to edge and the
corner clusters; refuse the pill, the blur and the centre toolbar; keep
every item the bar carried reachable — by pointer *and* by `Tab` — from a
cluster or the menu on it, because a canvas app with no chrome to focus is
a canvas app with no keyboard.

**One thing the pattern costs, and what we did about it.** Floating chrome
overlaps content by definition, and `fitView` centres inside its padded
rectangle, so a symmetric padding parked the first panel's header under the
top-left cluster on the very first open — an occlusion, measured on the
first capture of this pass. The fit's padding is now the cluster geometry
(60 px top, 204 px bottom, 24 px sides, shrunk together on a pane too short
to spare a third of its height), and both the first-open fit and **Fit**
use it. A reader can still drag a panel under a cluster; that is theirs to
do, and the minimap says where everything is.
