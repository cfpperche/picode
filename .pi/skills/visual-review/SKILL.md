---
name: visual-review
description: Capture and visually judge UI screenshots. NEVER claim a visual verdict from code or eval alone. A clipped overlay is an instant FAIL.
---

# Visual review

Code review cannot see pixels. `eval` JSON cannot see pixels. This skill
is the only valid visual verdict.

## Instant FAIL (do not commit)

Any one of these on a captured screenshot or geometry audit:

1. **Clipped overlay** — menu, popover, toast, slash list cut by the viewport
2. **Unreadable items** — labels truncated to meaninglessness, date wrapping into sludge
3. **Occlusion** — a control covered so it cannot be used
4. **Double scrollbar** on one surface
5. **Dead hover** — primary action paints the same color as its parent
6. **Geometry clip** — `overlayAudit` reports `ok: false`

If you **noticed** a defect in reasoning and still said done, that is a
contract violation (AGENTS.md honesty). Fix it or emit FAIL. Never ship it.

## The loop

1. **Serve** the rebuilt binary (go:embed — `make web && go build`).
2. **Act** with `agent-browser` (open the surface, click the control).
3. **Measure** after every overlay opens:

```bash
agent-browser eval 'JSON.stringify(window.__picodeOverlayAudit())'
```

`ok: false` → FAIL. Do not proceed to a verbal PASS.

4. **Capture** a full-viewport PNG (`agent-browser screenshot`).
5. **Read** the PNG with the `read` tool. Zoom mentally: edges, overflow,
   contrast, hierarchy.
6. **Answer the card** (required in the reply, before "done"):

```
visual-card:
1. Overlay fully inside the screenshot? yes/no
2. Every item readable? yes/no
3. Trigger still usable? yes/no
4. Clip / double-scroll / dead hover? yes/no
5. Terminal-averse next click obvious? yes/no
```

Any **no** on 1–3 or **yes** on 4 → `visual-review: FAIL`. Fix in-session
or record the debt in `docs/handoff.md` and do **not** claim quality-gate PASS.

7. **Verdict**

```
visual-review: PASS (sessions-open.png + overlayAudit ok; card 5/5)
visual-review: FAIL (sessions4.png: popover clipped at viewport top)
```

## Placement rule (product)

- Chrome at the **top** of the pane (tabs, session bar) opens overlays
  **down** into the canvas.
- Composer at the **bottom** opens overlays **up**.
- If space is short: scroll **inside** the overlay. Never clip outside
  the window.

## Honesty clauses

- NEVER issue a visual verdict without reading an actual image.
- NEVER treat `eval` / snapshot text as a visual PASS.
- Capture the relevant **states** (empty, open, hover) when they matter.
- If capture is impossible: `visual: UNVERIFIED` + handoff debt.
