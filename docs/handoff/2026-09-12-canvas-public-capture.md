# 2026-09-12 — the Canvas gets a public capture

`feat/canvas-public-capture` → `main` (82c82ca5). The owner's item 4.

## What is new

- `desktop-canvas` profile in `scripts/lib/docs-surfaces.mjs` (the canvas
  components plus `canvas.go`, `canvas_edges.go`, `canvas.js`, `canvas.css`),
  and `app-canvas` in `DOC_SCREENSHOT_SURFACES`.
- A surface entry in `docs-shots.mjs` that finds the canvas id with
  `hashEval`, the way `app-inspector` finds its agent.
- `seedCanvas` in the docs fixture: two agent CLIs, a shell and a text panel.
  Rectangles are **written out, not packed** — the product no longer packs,
  and a capture with a neat automatic grid would advertise a gone behaviour.
- `docs-site/guide/canvas.md` rewritten where it was stale.

## The guide was publicly wrong

It still taught **Add panel**, "panels land in the first free slot", "with
only one canvas there is nothing to pick, so it is just the name", and the
card in the middle of an empty plane. All four changed today. Worth a habit:
a canvas chrome change is a guide change, because that page describes the
chrome sentence by sentence.

## Known limitation

Every panel in the capture reads **Stopped**. `running` is derived from tmux
presence and the fixture spawns no processes by design, so this is a property
of the whole capture harness, not of this image. Making it read live would
mean the docs fixture starting real sessions — a decision about the harness,
not about the Canvas.
