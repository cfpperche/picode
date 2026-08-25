# Benchmarks — the bars we hold ourselves to

> Quality is not a vibe; it's a checklist backed by companies that set the
> standard. This file is the reference used by `/skill:quality-gate` and
> `/skill:uiux-review`. Update it when a benchmark stops serving us —
> explicitly, with rationale (it's a decision, see ADR process).

## Engineering benchmarks

Inspired by: **Google** (code review culture, small CLs), **Stripe**
(documentation as a product), **SQLite** (testing discipline), **Go team**
(stdlib-first minimalism), **Keep a Changelog / SemVer** (release hygiene),
**ADRs** (Michael Nygard) for decision records.

### The bar

1. **Build must never break.** CI is green on `main`, always. Work-in-progress
   that compiles + passes tests beats "finished" code that doesn't.
2. **Small, reviewable changes.** One logical change per commit/PR. If a diff
   needs a meeting to explain, split it.
3. **Tests travel with code.** New endpoint → handler test. New parser →
   table-driven test with edge cases. Bug fix → regression test first.
4. **Stdlib first.** Each non-stdlib dependency needs justification. "It's
   popular" is not justification; "mature, small, does what we can't" is.
5. **Docs change with code** (see AGENTS.md — non-negotiable #1).
6. **Visual gate on UI.** Screenshot read + overlay geometry audit +
   visual-card. Clipped menus are FAIL (see `/skill:visual-review`).
7. **Changelog discipline.** User-visible change ⇒ `[Unreleased]` entry,
   Keep a Changelog format, honest verbs (no "various improvements").
8. **Decisions are recorded.** Architectural choice ⇒ ADR. Superseded ADRs
   stay; history is evidence, not clutter.

## UI/UX benchmarks

Inspired by: **Cursor** (product + aesthetic north star — see
[benchmark-cursor.md](benchmark-cursor.md)), **t3code** and **paseo**
(architecture + composer-depth — see [benchmarks/](benchmarks/)),
**Linear** (speed, keyboard-first, dark-first, density), **Vercel/Geist**
(typography, minimal chrome, user menu, theme), **Stripe Dashboard**
(progressive disclosure, empty states that teach), **Apple HIG**
(clarity, deference to content), **xterm.js/ttyd** (terminal honesty).

**UI copy rule (owner directive, 2026-08-23):** documentation does not live
in chrome. Explanatory prose, setup steps and product storytelling belong
in README/docs — UI surfaces carry state and actions only (minimal empty
state = one line + one action; statusbar = live state, not hints).

### The bar — every UI change must satisfy

**Feel (Linear)**
- [ ] Interactions respond in <100ms; never a frozen frame without feedback.
- [ ] Keyboard-first: every primary action reachable without the mouse.
- [ ] Dark-first design; light mode is a derivative, not the default.
- [ ] Density with breathing room — power tool, not toy.
- [ ] **Optimistic UI:** in-flight fetches show layout skeletons or keep
      last good results. A blank content well while loading is FAIL.

**Clarity (Stripe/HIG)**
- [ ] Progressive disclosure: advanced options hidden behind a deliberate
      reveal, core flows naked and obvious.
- [ ] Empty states teach: first-run screens show *what this is* and *the
      one action to take next*.
- [ ] Destructive actions are confirmed; reversible where possible.
- [ ] Language: no unexplained jargon for our terminal-averse audience.
      PTY → "terminal integration". RPC → "control channel". If a real term
      must appear (tmux), add a one-line tooltip.

**Deference (HIG)**
- [ ] The agent's output is the hero; chrome recedes. No gratuitous
      animation competing with content.
- [ ] Scrollbars are overlay chrome: ≤8px, no arrow buttons, transparent
      track (VS Code / Cursor / Linear). Native Windows 17px+arrows is FAIL.
- [ ] Status is always truthful: streaming shows streaming, stuck shows
      why, unknown says "unknown" — never fake progress.

**Terminal honesty (ttyd)**
- [ ] The embedded terminal behaves like a terminal: selection, copy/paste,
      scrollback, resize. Users of the real Pi TUI must feel at home.

### Anti-benchmarks (things we refuse)

- Homemade widgets when Radix/cmdk/native already cover it (AGENTS.md Style).
- AI-slop UI: generic dashboard shells, 12-card grids of nothing, gradients
  for depth, spinner-only loading states, **blank wells while fetching**.
- Hiding the terminal to "protect" users (see philosophy: door, not cage).
- Modals for flows longer than 2 fields — wizards use full pages.
