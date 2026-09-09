# Study: a live grid of terminals on react-grid-layout 2.2.4 (Matrix, phase 0)

- **Date:** 2026-09-09
- **Plan:** [`docs/plans/matrix-app.md`](../plans/matrix-app.md) §10 asked this
  spike three questions: does react-grid-layout 2.2.4 (v2 API) hold up under
  React 19 with 300 items and live xterm bodies, what does it cost, and does
  chunk loading (bodies only near the viewport, sockets suspended out of view)
  bound the attaches the way §4.5 claims.
- **Method:** a throwaway `MatrixSpike` surface in the desktop bundle of
  worktree `feat/matrix-spike` (never merged), gated by `?spike=matrix`, on a
  scratch instance (`scripts/qa-scratch.sh`, port 8473, embedded UI build),
  driven by `agent-browser --session matrix-spike` (headless Chromium,
  1600×1000). 300 wrappers in a 12-column grid (`rowHeight 24`, margin 8,
  panels 4×10 cells = 512×312 px at width 1584); the first 12 wrappers bound
  to free terminals created through the scratch API, `top -d 1` running in
  nine of them; the rest static placeholders. Counters were read from the
  page (`window.__mx`) and from tmux (`list-clients`); CPU from `/proc` of
  every Chromium renderer on the box. Receipts are quoted as measured; the
  screenshots stayed in `var/screenshots/` (never committed).
- **Stack:** React 19.1.1 · react-grid-layout 2.2.4 (+ react-resizable 3.2.0
  for its handle CSS) · @xterm/xterm 6.0.0 · tmux 3.6 · node 24.11 · WSL2
  6.6.

## Results

| Question | Measured | Verdict |
|---|---|---|
| Renders and drags under React 19 | 300 wrappers mount; drag of a wrapper over 60 frames: **58.5 fps** (worst gap 27 ms) with the `extras` fast compactor, **60.5 fps** (worst 25 ms) with the standard one; items moved (transforms changed); console errors/warnings captured during drag, resize and 15 minutes of use: **0** | GO |
| Compaction cost at 300 items | `compact()` of the whole layout: **0.06 ms** fast, **1.02 ms** standard (20-run average) | either is free; keep `fastVerticalCompactor` for headroom |
| Resize a live terminal panel | SE-handle drag: panel 312 → 472 px; the tmux pane followed **58×14 → 58×23** through the existing fit path; no errors | GO |
| Bundle delta (desktop, `vite build`) | JS **+75 454 B raw / +24 309 B gzip**, CSS +3 729 B / +872 B gzip, all in the main chunk (2 791 486 → 2 866 940 B raw) | below the plan's 60 KB gzip lazy-import threshold |
| Idle cost, nine `top` panels on screen | in-page 60.2 fps, 0 % long tasks; Chromium renderers **6 %** CPU with the nine live panes vs **2 %** with none (system-wide sample, other sessions' browsers included) | ≈ 4 % for nine redrawing TUIs |
| Chunk loading bounds the attaches | scroll to the bottom of the 32 008 px grid: at +2 s still 12 attached (hysteresis), at +8 s **0 attached, 12 suspended**, 12 xterm instances kept, 300 placeholder bodies; tmux clients on the twelve sessions **3 → 0 within 10 s**; back to the top: 12 reattached, **the same xterm objects** (`sameInstances: true`), the nine `top` screens visible again | GO |
| Fast fly-over must not churn sockets | 4.0 s round trip top → bottom → top over all 300 wrappers: attaches stayed at 12 and the socket objects were the same before and after; **but** the prototype's "load now" rule mounted **341** bodies during the pass. With a **300 ms load dwell**: **6** loads during the same pass, while a 1 000 px/s scroll still loaded 27 wrappers | adopt the dwell |
| Unload timing | parked with every terminal out of the margin: 4 suspended at +4 s, all 12 at +8 s (timers start when each wrapper leaves), attached 0 | as designed |

## What the spike changes in the plan

1. **Load after a 300 ms dwell, unload after 5 s** (§4.5 said "load now"). A
   fly-over must not attach what it passes; a normal scroll still loads.
2. **Panel defaults:** a 4×10-cell panel at 1584 px is a 58×14 terminal —
   readable but cramped for a TUI. Default **4×14** cells (≈ 58×20) and
   **minW 4, minH 8**; `minW 3` (≈ 43 columns) is too narrow for any real TUI.
3. **Declare `react-resizable` directly** (its handle CSS is imported; the
   boundary check `web/tools/boundaries.mjs` refuses an undeclared import).
   Two dependency lines in the PR, not one.
4. **No lazy import needed** for the surface: +24 KB gzip.
5. **`suspendTermSocket` works as designed**: a reversible stop beside the
   one-way `closedByUser`, `kickTermSocket` lifts it; `ShellTerm`'s existing
   "reattach the same instance" path resumes without a new xterm.
6. **Teardown on navigation is `pongWait`-bound (60 s):** after a page reload
   the previous page's twelve attaches stayed as tmux clients until the
   bridge's pong timeout (24 clients observed, then 12). Unload closes the
   socket explicitly, so its teardown is immediate (3 stragglers ≤ 10 s).
   Existing bridge behaviour, now written down.
7. **Unknown hashes are rewritten to `#/` at boot**, which unmounted the first
   hash-gated spike; the real surface is an app tab (`#/app/matrix`), a route
   the shell already keeps — nothing to change, one thing to know.

## Alternatives, re-checked

dockview 8.3.0 stays the answer for tabs-inside-panels and split trees, which
v1 does not want; gridstack 13.2.0 offers the same grid with a DOM-first API.
Nothing measured here argues for either over react-grid-layout: the v2 API
worked first time under React 19, the numbers above have headroom, and the
`legacy` entry remains the documented fallback.

## Receipts

- Counters: `window.__mx` (loaded, attached, suspended, instances, stats,
  `compactBench`, `drag`, `idle`, `log`) in the spike component; tmux:
  `tmux list-clients -F '#{session_name}'` filtered to the twelve sessions
  named by the scratch API; pane size: `tmux display -p -t '=<session>:0.0'
  '#{pane_width}x#{pane_height}'`.
- QA notes: `tmux send-keys` inside a `while read` loop needs `< /dev/null`
  (tmux reads stdin); a tmux session for a new terminal exists only after
  its first attach, so seed, open, then type.
