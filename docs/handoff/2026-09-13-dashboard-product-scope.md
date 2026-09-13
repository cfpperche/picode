# 2026-09-13 — feat/dashboard-product-scope

- Retired the dashboard's folder-scope mode (ADR-0127, amends ADR-0097):
  the "This machine | PiCode" chips leaked the product's own workspace
  into a consumer filter, filtered by a set that changed under the label,
  and rendered where they meant nothing (zero workspaces). The dashboard
  now answers one question — what the machine's agents did in the window.
- Server: `GET /api/sessions/stats` loses `?scope=` and the payload's
  `scope` field; cache identity is `root|range` again. `climetrics`
  drops `Scope`/`Claimed`/`InScope` (the InScope window-gating bug class
  dies with it); `claimedDirs` survives only as labelling input.
- Web: `ScopePicker.jsx` deleted; persisted scope preference cleaned from
  localStorage on load; the workspace card's tooltip says "outside your
  workspaces" instead of naming the product.
- Tests: `TestHandleSessionStatsIgnoresRetiredScopeParam` pins that a
  client still sending `scope=picode` gets the complete machine window.
  Decision table (claimed workspaces × session location) covered by that
  test plus the existing label/cache tests.
- Verified on a scratch instance (`qa-scratch.sh start dashscope`, seeded
  plus a minimal pi session fixture): head shows range chips only, range
  switch reloads cleanly, workspace row renders with the new tooltip,
  empty states speak. Screenshots read: `var/screenshots/dashscope-head.png`,
  `dashscope-empty.png`, `dashscope-workspace-row.png`,
  `dashscope-workspace-framed.png`.
- visual-review: PASS (dashscope-workspace-framed + overlayAudit ok; card 5/5)
- Board byte cap: reworded one `process.md` debt (same meaning, −30 B) so
  `make handoff` fits; the cap will bite the next branch anyway.
