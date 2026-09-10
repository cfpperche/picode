# 2026-09-10 — feat/mobile-work-menu: workspace actions in a head-row menu; Work horizontal scroll fixed

Shipped: the five per-workspace actions (Communication, Files, Git, New agent,
New terminal) moved from a nowrap button strip at the bottom of each workspace
group into a Radix DropdownMenu on the group head row
(`web/mobile/src/components/WorkspaceMenu.jsx`, wired in `Work.jsx`; dead
`.m-work-group-actions` CSS removed, `.m-work-menu-btn` added in
`mobile-lists.css`).

Verified: scratch instance (qa-scratch, seeded) at 390×844 — `scrollWidth ==
390` on #/work (was overflowing ~120px, IMG_3689/3690); menu opens/closes,
overlayAudit ok, "New terminal" item really created Terminal 2; screenshots
read in light and dark. `make ci-scoped` PASS (fmt, vet, hooks, test-js, build).

visual-review: PASS (work-menu-open.png, work-menu-closed.png, work-menu-dark.png; card 5/5)

Not done / debts: none known. Menu reuses `.ws-row-menu` tokens, so dark mode
follows the existing pattern; no ADR — UI refinement, no boundary crossed.

Merge: fast-forward ready.
