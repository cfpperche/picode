# 2026-09-08 — feat/gg-mobile-sheet: the graph's actions on the phone

Mobile gets ADR-0096's write vocabulary as a sheet, over the same shared
module and the same doors; presentation is mobile's own (ADR-0072/0095).

- `lib/git/graphSheet.js` (pure, 7 tests): target from a row or view, the
  shared menu grouped into sections, doors per running occupant (agents and
  pi terminals alike), the shared gate re-exported so the two apps cannot
  disagree about the typed phrase.
- `GitGraphActionSheet.jsx`: list → form; the preview is the server's
  `git/compose` answer; delivery through the host's `prepareGit` / `askGit`
  (`askGit` now routes a terminal occupant to `/api/terminals/{id}/ask`).
- Entry: a 500 ms press on a history row (moving 8 px cancels; the click that
  follows a fired press is swallowed), a real contextmenu in the desktop
  preview, and an **Actions** button on the commit and worktree views.
- After a delivery the screen watches `git/head` for ≤30 s while visible and
  reloads on change or at the deadline.

Two defects found in the browser: `ownerKey` referenced in a dependency
array before its `const` (the mobile boundary said "Couldn't open this
screen"; fixed by depending on `owner`'s fields), and sheet rows shrinking
to their label because the new group wrapper broke the list's flex stretch
(all 20 rows now measure the list's width).

visual-review: PASS (gg6-mobile-sheet-list, gg6-mobile-sheet-tierc at
390×844; overlayAudit ok, `[data-align-row]` 44/44; card 5/5).

Debts: the ask door was not exercised from the phone (no running occupant on
the scratch); a commit opened by deep link before History loads has no graph,
so its sheet offers commit rows without the refs until History is visited.
