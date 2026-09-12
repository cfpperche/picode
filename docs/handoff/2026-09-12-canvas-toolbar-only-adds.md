# 2026-09-12 — the canvas toolbar is the only way in

`feat/canvas-toolbar-only-adds` → `main` (79e5ddc1).

## What went

- **Add panel…** from the `⋯` menu.
- The **Add panel** button on the empty canvas card; the card now reads
  "Pick one below, then draw where it goes."

Both placed a panel through `nextSlot`, which packs from the canvas origin —
the defect the toolbar was built to fix. `nextSlot` now has **no caller for
new panels**; it is still used by `tidyCanvas` and kept for the migration
record.

## The cost, stated plainly

**File and diff panels can no longer be created.** They have no tool: a file
reaches a canvas from a tab that is already open, not from a rectangle drawn
on a plane, and the unfiltered picker was their only door. Existing panels of
both kinds keep working — nothing was removed from the store, the renderers,
or the layout.

Giving them a tool is not just wiring: the tool has to answer what drawing a
rectangle means when the thing being placed is chosen from the tabs you
already have open. Worth a decision before a commit.

## Verified

Isolated instance at `/browser/`: the `⋯` menu is Show controls / New canvas /
Rename / Delete canvas / Background / Close tab; the empty card carries zero
buttons; arming Terminal and drawing still opens **Add terminal** with the
one seeded terminal in it.
