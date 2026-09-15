# 2026-09-15 — ask-prompt: the Ask prompt closes Browser permissions v1

Shipped: the shell defers a site's request (btab.rs + the pure
`permissions.rs` decision table), the store distinguishes policy from a
reported decision (migration 051, `standing`), and the tab renders Allow /
Block / Always allow between the toolbar and the page. `cargo xwin build` ✓;
`make close` PASS (full).
Visual: the Site settings dialog was read in a scratch (empty, Ask selected,
saved row, scrolled to Close); the Ask bar needed a forced demo ask — the
shell is Windows-only (`var/screenshots/ask-bar-forced.png`, real component +
CSS, stubbed invoke).
visual-review: PASS (dialog 5/5 in the scratch; forced bar inside the
viewport with `__picodeOverlayAudit` ok, disclosed as a forced state)
Not done: the COM half — deferral, answer, watchdog — has never run outside
Windows.
Merge: fast-forward ready.

## Next up

- Owner: live-check Ask in the shell (deploy + restart) — the first
  deferral run.

## Debts

- The Ask COM path (deferral, answer, watchdog) has never run outside
  Windows.
