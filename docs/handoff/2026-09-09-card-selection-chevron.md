# 2026-09-09 — feat/card-selection-chevron: quiet card selection + chevron on the card grid

Shipped:
- `.ws-item.active` drops the `inset 2px 0 0 var(--accent)` bar (owner report:
  it read as a blue border on selected agent/terminal cards); selection is
  `--bg-hover` + `--border`. `is-needs-you` keeps the bar (attention, not
  selection). Focus rings unchanged.
- Checklist disclosure on the card grid, superseding the 2026-09-06
  own-gutter refinement (owner asked for alignment): disclosure margin-left
  23px, chevron box 12px (`IconChevronRight size={12}`), gap 4px — chevron
  column = title text = folder/branch icons (23px), text column = folder
  label (39px); expanded steps use the same 12/4 columns.
- Fixed en route: `.ws-context-btn svg { flex: none }` — long paths
  flex-shrank the folder/git icon to ~3px (caught on a deep worktree seed).

Verified: `make ci-scoped` PASS; scratch instance (qa-scratch) with seeded
agent+terminal+checklists — geometry via getBoundingClientRect (chevron 50/12,
folder 50/12, texts 66) and screenshots read in light + dark, collapsed +
expanded, agent + terminal selected; `__picodeOverlayAudit` ok.

visual-review: PASS
Not done / debts: none known.
Merge: fast-forward ready (after re-close over main).
