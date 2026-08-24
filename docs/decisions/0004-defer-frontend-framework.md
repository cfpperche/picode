# ADR-0004: Defer the frontend framework decision — vanilla ES modules for now

- **Status**: superseded by [ADR-0008](0008-react-vite-tailwind.md)
- **Date**: 2026-08-23

## Context

M1 (terminal grid) needs: a workspace list, tabs, xterm.js terminals, one
form. The M0 page was hand-written HTML/CSS/JS. Options:

1. Adopt React/Preact/Svelte + a build pipeline (vite) now.
2. Stay vanilla (ES modules, vendored xterm.js 5.5.0 + fit addon) and
   adopt a framework when complexity demands it.

PiCode's go:embed pipeline currently serves `internal/web/public/` with
zero build steps — one command (`make dev`) works for contributors and
agents, and the repo contract prizes that simplicity. The M1 UI surface
is small (~350 lines of app.js).

## Decision

Stay **vanilla ES modules with vendored dependencies** for M1–M2. Revisit
with a new ADR when ANY of these thresholds is hit:

- UI code exceeds ~1,000 lines or 3 distinct views with shared state;
- component duplication becomes a maintenance tax (3rd copy of anything);
- a rich agent panel (M2) requires reactive state management by hand.

## Consequences

- **Easier**: no node toolchain in the contribution loop; agents and
  humans run `make dev` with Go only; upgrades are file swaps.
- **Harder**: no component model — discipline required (small functions,
  one state object, explicit render functions); vendored xterm.js must be
  updated manually (security releases); a future migration pays a rewrite
  tax proportional to the vanilla surface.
- **If wrong**: the thresholds force the decision early; migrating ~1k
  lines of vanilla JS into a framework is a bounded, one-milestone task.

## Alternatives considered

- **Adopt Svelte now**: rejected — premature; M1 needs would not pay for
  the toolchain complexity (against AGENTS.md simplicity rule).
- **Adopt React now**: rejected — same, plus heavier vendoring story.
- **HTMX/server-rendered**: rejected — the terminal grid is fundamentally
  client-side (xterm.js + WebSocket); HTMX adds little there.
