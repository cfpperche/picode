---
name: uiux-review
description: Review UI/UX work against PiCode's design benchmarks (Linear-speed, Stripe-clarity, dark-first, terminal-averse friendly). Use when touching internal/web/public, web/, or any user-facing copy.
---

# UI/UX review

Checklist against `docs/benchmarks.md` (UI/UX section) and, for new
surfaces, `docs/benchmarks/` (Cursor / t3code / paseo). Use whenever this
session touched anything the user sees: pages, components, copy, terminal
styling, error messages.

## Checklist

### Feel (Linear bar)
- [ ] Feedback in <100ms for every interaction; nothing freezes silently.
- [ ] Keyboard-first: primary actions reachable without a mouse.
- [ ] Dark-first: design looks native in dark mode; light mode derived.
- [ ] Density with breathing room (power tool, not toy).

### Clarity (Stripe/HIG bar)
- [ ] Progressive disclosure: advanced options behind deliberate reveals;
      core flow is naked and obvious.
- [ ] Empty states teach: what this is + the one next action.
- [ ] Destructive actions confirmed, reversible where possible.
- [ ] Jargon audit for terminal-averse users:
      PTY → "terminal integration"; RPC → "control channel"; real terms
      that must stay (tmux, workspace) get a one-line tooltip.
- [ ] Buttons/menus use plain verbs: "Create agent", not "Initialize".

### Deference (HIG bar)
- [ ] Agent output is the hero; chrome recedes.
- [ ] No gratuitous animation; no fake progress — streaming shows
      streaming, stuck shows why, unknown says unknown.

### Terminal honesty (ttyd bar)
- [ ] Embedded terminal: selection, copy/paste, scrollback, resize all work.
- [ ] Nothing pretends to *be* the terminal — we embed the real one.

### Product benchmarks (docs/benchmarks/)
New or redesigned surfaces cite at least one of Cursor, t3code, or paseo
and the PiCode adaptation (or why we refused the pattern). Do not copy
multi-runtime SDKs (ADR-0003) or editor chrome (philosophy).

### Cursor bar (docs/benchmark-cursor.md)
Apply the product checks to surfaces that already exist (patterns land per
milestone); apply tokens and density from day one.

- [ ] Density: 13px base UI font, 4px spacing grid, tool-call rows ≤32px
      collapsed — power-tool density, no marketing-airy spacing.
- [ ] Activity feed: agent actions render as compact collapsible rows with
      status + duration, never walls of text.
- [ ] Streaming: markdown + syntax-highlighted code while streaming;
      input never blocks.
- [ ] Diffs: edits surface inline/expandable diffs with per-hunk
      accept/reject where applicable.
- [ ] Checkpoints: every turn leaves a visible restore point (session tree).
- [ ] Reachability: model picker ≤2 clicks from any agent view;
      Ctrl+K command palette for navigation (when present).
- [ ] Motion: 100–200ms transitions; skeletons over spinner-only waits.
- [ ] Optimistic UI: first load = skeleton matching the real layout;
      refetch = keep last results. Blank content well while fetching = FAIL.
      Skeletons are chrome, never invented rows/names.
- [ ] Design tokens: surfaces, type, accent and status colors follow the
      token set in docs/benchmark-cursor.md.

### Anti-benchmarks (instant FAIL)
- Generic dashboard slop: card grids of nothing, gradient depth,
  spinner-only loading, unexplained jargon.
- **Clipped overlays** (menus cut by the viewport). Geometry +
  screenshot required — see `/skill:visual-review`.

## Verdict

```
uiux-review: PASS (9/9 checks, 0 jargon hits)
uiux-review: FIX (loading state has spinner-only feedback on /agents)
```

Failures get fixed or explicitly listed as debts in `docs/handoff.md`
with the reason.
