# 2026-09-16 — overlay-sweep: the page gets out of the way of anything that crosses it

Owner report (screenshot 20:28): the editor's tab-strip menu came up invisible
under the work browser, "vou ter que ficar descobrindo bugs de sobreposição".
Fair — the fix so far was a whack-a-mole: `.dlg[data-state="open"]` was the
only layer the tab watched, so the command palette (raw Radix Dialog, class
`.palette`) and every chrome menu were never seen at all.

The rule is now geometry, over one shared list of layers:
- `web/shared/domain/overlayAudit.js` exports `OVERLAY_SELECTORS` — role+state
  for Radix (`[role="dialog"|"alertdialog"|"menu"|"listbox"][data-state="open"]`,
  `[data-radix-popper-content-wrapper]`) plus the app's own classes
  (`.dlg`, `.cockpit-pop`, `.rail-pop`, `.pkg-job`, `.img-lite`, sonner's
  `[data-sonner-toast]`, `#inspector`). Two consumers: the clipping audit and
  the hide rule, so a new surface is seen by both or by neither.
- `web/browser/src/lib/floatingLayers.js` collects the visible ones, with
  geometry (`view frame` exclusion so parked chrome and off-screen menus do
  not count) and one intersection test; the tab hides the native view when a
  layer reaches its rectangle and shows the frozen page behind it
  (`btab_preview`), clearing the still when the layer goes.
- The observer coalesces with a 100 ms floor and skips a hidden document: the
  app mutates constantly while an agent streams.

Verified in a stub scratch (screenshots read): the "All tabs" menu (the
reported case, 18 tabs), the command palette, the user menu, the New
workspace dialog, and an inserted toast-class layer each park the page and
restore it on close; a sidebar menu that does *not* reach the page leaves it
alone; the inspector docks without a false cover (the pane resized 1280→960 in
the same check). `__picodeOverlayAudit()` now also fails when a layer sits
over an unmarked page — verified `ok: true` with the dialog up, and the unit
tests cover the failing row. 1097 JS tests, `make web`, `make close`.

## Next up

- Owner: use the app the way you did when it bit — tab menus, palette,
  dialogs, dropdowns over a loaded page.

## Debts

- Focus mode's edge strips and its fixed sidebar are excluded from the layer
  list on purpose (always-on chrome would hide the page for as long as focus
  mode lasts). Over the browser pane those handles therefore do not respond;
  a transition-aware recompute is the fix if it ever matters.
