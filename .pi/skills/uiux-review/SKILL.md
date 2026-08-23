---
name: uiux-review
description: Review UI/UX work against PiCode's design benchmarks (Linear-speed, Stripe-clarity, dark-first, terminal-averse friendly). Use when touching internal/web/public, web/, or any user-facing copy.
---

# UI/UX review

Checklist against `docs/benchmarks.md` (UI/UX section). Use whenever this
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

### Anti-benchmarks (instant FAIL)
- Generic dashboard slop: card grids of nothing, gradient depth,
  spinner-only loading, unexplained jargon.

## Verdict

```
uiux-review: PASS (9/9 checks, 0 jargon hits)
uiux-review: FIX (loading state has spinner-only feedback on /agents)
```

Failures get fixed or explicitly listed as debts in `docs/handoff.md`
with the reason.
