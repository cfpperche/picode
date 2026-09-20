# Inspector session changes (feat/inspector-session-changes)

Product: the Changes tab follows the session's dirty worktrees, not just the
anchor folder. When an agent or terminal works in a linked worktree while
anchored at the repo root, the rail shows that worktree's changes instead of
"No changes". No CLI parsing, no session reading: git ground truth only, so
it works for any CLI, shell terminal, and project.

## Design

- `gitstatus` gains additive `worktrees[]`: dirty linked worktrees only
  (self/bare/prunable excluded, cap 8, `worktreesTruncated` counts the rest).
  Each entry is shaped like the root page (path, branch, upstream,
  ahead/behind, changes, totals).
- `?worktree=<branch-or-commit>` (the gitgraph precedent, `WorktreeOfRef`)
  extends to the file routes (text/blob/workdiff/save), so a worktree file
  opens in the center like any other. Server-validated selection, never an
  arbitrary address; the `root` precondition still asserts the resolved cwd.
- The fleet watcher also inspects linked worktrees (cap 8/repo), so
  `git.updated` fires for followed roots and the rail stays live.
- Client resolves one view from root + groups (pure, tested):

## Decision table

| # | Root | Dirty WTs | State | View |
|---|---|---|---|---|
| 1 | clean | 0 | — | empty state (unchanged) |
| 2 | dirty | 0 | — | root tree (unchanged) |
| 3 | clean | 1 | not dismissed | FOLLOW: worktree group + "Following" pill with Back |
| 4 | clean | 1 | dismissed | root empty + "Touched in <branch> (N)" switcher with View |
| 5 | dirty | ≥1 | — | grouped: root first, then worktrees |
| 6 | clean | ≥2 | — | grouped: worktrees only, no pill |
| 7 | followed WT clean/gone | — | following | fall back to root view, pill gone |
| 8 | scope = this agent | any | — | intersect each group with touched paths; hide emptied groups, all-empty keeps "Show all" |
| 9 | `git.updated` for a shown root | — | — | coalesced refetch (existing queue) |
| 10 | non-git root | — | — | unchanged (Files only) |

Back fixes the anchor until Refresh/manual View; View sets follow. The PR
tab stays on the anchor (pill copy is scoped to Changes). File tabs carry
their worktree root in persisted viewer state — a reload must never read a
worktree-relative path through the anchor root (same path, different file).

## Out of scope (refused with reason, not debt)

- Cross-repo edits (agent writing outside the repo): needs session-touched
  parsing per CLI; the worktree signal covers the branch workflow universally.
- Nested non-linked clones: invisible to `git worktree list`.
- Live-cwd for TUI agents and readers-fallback touched extraction: subsumed
  by worktree-following for Changes purposes.
