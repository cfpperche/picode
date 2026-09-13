# 2026-09-13 — feat/gg-workspace-picker: the graph's first toolbar item is the workspace

Shipped: the git graph toolbar's first item names the **workspace** whose folder the history is read through and switches it from a dropdown (`WorkspacePicker.jsx`, `lib/gitWorkspacePicker.js`, `App.switchGraphWorkspace`); the owner is the read and the write context (ADR-0022/0096), so a sibling worktree swaps in place and another repository renames the tab to `g:<key>` (or the tab that already holds it takes the pick); ADR-0022 amendment + `docs/architecture/routes.md`.
Verified: `make ci-scoped` PASS; `node --test` 304 browser lib tests (6 new, one per pure decision-table row) + 649 shared; scratch `ggws`/`ggws1` with 4 workspaces over 2 repositories (one pair of sibling worktrees) driven by agent-browser — picker open, same-repo swap (HEAD dot and `this worktree` move, tab unchanged), cross-repo rename (tab label changes, still one tab), adopt, single workspace → plain title, terminal owner → names its workspace.
visual-review: PASS (var/screenshots/ggws-{2,3,4,5,6,8,9}.png; overlayAudit ok; card 5/5)
Not done / debts: rows 8 and 9 of the decision table had no automated test in this branch; both were paid on 2026-09-13 (`feat/gg-debts`, the note beside this one). Mobile Git has no switcher.
Merge: fast-forward ready (the mobile Git switcher came with `901f289f`).
