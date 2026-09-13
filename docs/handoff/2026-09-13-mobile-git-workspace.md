# 2026-09-13 — feat/mobile-git-workspace: the phone picks the workspace too

Shipped: `GitWorkspaceSheet.jsx` and a workspace control in the Git screen's folder row (`web/mobile/src/screens/Git.jsx`), plus the rule both shells now share — `@picode/shared/domain/gitWorkspace.js` (`pickerOptions`, `workspaceForOwner`, `triggerLabel`), moved out of the browser lib so the phone asks the same question; the browser lib keeps only its tab/URL identities. A pick navigates (`openGit({kind:"workspace", id})`), so Back returns to the previous workspace's screen.
Verified: `make ci-scoped` PASS; 306 browser + 653 shared + 356 mobile lib tests; scratch `mgw` (3 workspaces over 2 repositories, one pair of sibling worktrees) on the 390×844 shell driven by agent-browser: control reads the workspace with its branch, the sheet lists all three with the current checked (overlay audit ok), a pick lands on the chosen workspace's screen (`#/git/w/orikami-152e30`), and with one workspace left the row falls back to a plain line. Deleting workspaces in that run also re-confirmed the watcher fix (`health 200`).
visual-review: PASS (var/screenshots/mgw-{1,2,3,4}.png read; card 5/5)
Not done / debts: the control lives in the folder row, not the header — the header's one right slot is Actions (ADR-0095 keeps the phone's own presentation).
Merge: fast-forward ready
