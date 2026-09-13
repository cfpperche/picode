# 2026-09-13 — handoff-dialog-width

## What changed
- The "Continue in <CLI>" dialog kept a 520 px column on wide screens
  (owner: "péssima UX"). Desktop now uses the 720 px centred-dialog bar:
  `.dlg.dlg-handoff { width: min(720px, calc(100vw - 48px)) }` — the
  `.dlg` qualifier matters, the bare 400 px `.dlg` width was winning.
- How (Native/Brief) and How much (recent/all) render through a new
  `.handoff-choices` wrapper: stacked below 720 px, `grid 1fr 1fr` from
  720 px (both renderers: browser + mobile JSX).
- Brief preview: `max-height: min(42vh, 420px)`, wider padding; mobile
  caps at `min(36vh, 320px)` and keeps choices stacked (no media query
  in `mobile-sessions.css`), sheet capped at 720 px.

## Verification
- Scratch fixture on qa-scratch (:8473): dialog 720 px at 1280 vw, both
  radio groups one row (`distinctTops: 1`); at 420 vw → 372 px, fits,
  stacked. Mobile shell on #/more/clis (chunk CSS loaded): 388 px,
  stacked, fits. `__picodeOverlayAudit()` ok, no clipping.
- Screenshots: var/screenshots/handoff-width-{brief,native,narrow,mobile}.png
- ci-scoped PASS (fmt, vet, hooks, test-js, build).

## Debts / notes
- Fixture on a mobile route without the AgentClis chunk shows the
  classes unstyled — artefact only; the dialog imports the CSS with its
  own chunk. Not a bug.
- Loading/error lines were not separately screenshotted (text-only rows
  in the same container).
