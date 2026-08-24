---
name: visual-review
description: Capture and visually judge UI screenshots against PiCode's design benchmarks (Cursor bar, tokens, density). Use for any UI work needing visual validation; NEVER claim a visual verdict from code alone.
---

# Visual review

Code review cannot see pixels. This skill closes the loop between "the CSS
looks right" and "the UI looks right", using Pi's native image reading
(the `read` tool renders PNG/JPG/GIF/WebP to the model).

## The loop

1. **Serve** — ensure the UI is running (`make dev` → 127.0.0.1:7331) or
   use an already-running instance.

2. **Capture** — `bin/picode screenshot --url <url> --out var/screenshots/<name>.png`
   (lands as M1 step 0; until then capture with any local tool — or ask the
   user — and state exactly how the image was obtained).
   For **interactive** verification (click-through flows, state changes),
   use the `agent-browser` skill — it drives a real Chromium via bash.

3. **Look** — `read` the PNG. Study the actual rendering: layout, spacing,
   density, contrast, hierarchy, empty states. Zoom mentally into details
   (alignment, truncation, overflow).

4. **Judge** — against the bars:
   - `docs/benchmark-cursor.md` — design tokens, density rules (13px base,
     tool-call rows ≤32px, 4px grid), motion, truthful status.
   - `docs/benchmarks.md` — clarity bars (progressive disclosure, empty
     states teach, jargon audit) and anti-benchmarks (instant FAIL list).

5. **Verdict + evidence** — file the screenshot under
   `docs/screenshots/<milestone>-<view>-<state>.png` (e.g.
   `m1-termgrid-home-empty.png`), reference it in the PR/handoff, and emit
   the verdict.

## Rules (honesty clauses)

- **NEVER issue a visual verdict without reading an actual image** of the
  running UI. Static analysis of HTML/CSS is not visual review.
- Capture the relevant **states**, not just the happy one: empty, loading,
  streaming, error — when applicable.
- If capture is impossible in the current environment, say so explicitly —
  `visual: UNVERIFIED (no capture available)` — and add it to
  `docs/handoff.md` known debts.
- Compare against the benchmark, not against taste. Cite the rule each
  finding violates (e.g. "violates density: rows >32px collapsed").

## Verdict format

```
visual-review: PASS (m1-home.png: tokens ✓ density ✓ empty-state teaches ✓)
visual-review: FIX (m1-termgrid.png: row height 40px > 32px bar [density];
                     spinner-only loading [anti-benchmark]; 'PTY' label
                     unexplained [jargon audit])
```

FIX items either get fixed in-session or land in `docs/handoff.md` debts
with the reason.
