# ADR-0073: The git graph shows every worktree's working tree

- **Status**: accepted
- **Date**: 2026-09-05
- **Amends**: 0022 (worktree visibility), 0038 (uncommitted row scope)

## Context

The graph (ADR-0022) collapses a repository's worktrees into one picture and
names the agents living in each checkout — but only implicitly: an agent
name inside a branch pill. Three real gaps follow:

1. **Uncommitted changes are owner-scoped.** `LoadFiltered` statuses only
   the cwd the graph was read through, so a sibling worktree mid-task —
   precisely the normal PiCode parallel-agent state — shows nothing. The
   user cannot see *what an agent's worktree has not committed yet*.
2. **A worktree with no agent is invisible.** Clean checkout on a branch
   reads as a bare branch; nothing says a checkout exists on disk.
3. **Detached checkouts are invisible.** No branch ref means no pill, no
   position marker.

Benchmarks studied before designing (per `docs/benchmarks/README.md`):

| Tool | What it does | What we take |
|---|---|---|
| Conductor (`conductor.build/docs/concepts/git-worktrees`) | One worktree per workspace; "keeps the diff visible"; branch + diff + PR state attached to the workspace | The worktree's dirty state is a first-class review unit |
| GitButler (`docs.gitbutler.com/ai-agents/parallel-agents`) | Branch list marks what is checked out where; documents "one checked-out branch per worktree" | The branch picker says which branches live in a worktree |
| mhutchie/vscode-git-graph (the port's reference) | Shows Uncommitted Changes only for the single opened checkout; no worktree concept at all | The gap is ours to own: PiCode knows worktrees *and* agents |
| Crystal/Nimbalyst, herdr (`docs/benchmarks/2026-08-27-herdr.md`) | Worktree-first agent isolation; `worktree.*` as layout convenience | The graph stays "agent *has* a cwd"; the graph maps cwd → worktree, it does not make an agent *be* a worktree (ADR-0011 unchanged) |

Measured receipts (this repo, git 2.53, 7 worktrees): per-worktree
`git status --porcelain -uall` costs 3–95 ms; `%(worktreepath)` in
`for-each-ref` answers branch→worktree in one call; `worktree list
--porcelain` already carries `bare`/`detached`/`prunable`.

## Decision

The graph draws **one dirty row per worktree with changes of its own**, not
only for the reader's checkout. Six additive pieces:

1. **Payload.** `Worktree` gains `bare`, `detached`, `prunable`, `self`
   (the checkout the graph was read through), and `uncommitted {count}` —
   one `git status` per healthy, non-bare, non-prunable worktree, capped at
   32; the reader's own status is computed once and reused. Top-level
   `uncommitted` stays for compatibility.
2. **Layout.** `layoutUncommitted(commits, anchors)` prepends one
   pseudo-commit per dirty worktree anchored at *that worktree's* HEAD
   (hash `"*<n>"`, collision-free with object names). Each pseudo's trail
   down to its anchor row draws dashed. Anchors whose HEAD is outside the
   loaded window are dropped (same rule as 0038's single row).
3. **Row labelling.** A dirty row carries the branch pill (or the directory
   chip when detached), the worktree directory chip, the agents living
   there, and — for the reader's own checkout — a "this worktree" marker.
4. **Detached checkouts** decorate their HEAD commit with a directory chip
   (dashed border), agents included, so a no-branch checkout has a position.
5. **Worktree-scoped reads.** `gitstatus`, `gitdiff`, `git/blob` and
   `blob` accept `?worktree=<branch|full head hash>` for agent and terminal
   owners. The value is a **ref, never a path**: `gitgraph.WorktreeOfRef`
   resolves it through `worktree list` + an exact `refs/heads` match, so the
   URL still names no directory (ADR-0022's confinement, unchanged). The
   uncommitted detail opens the sibling's files inline; empty state and
   error copy are one line each.
6. **Branch picker.** Local branches checked out in a worktree are marked
   with the checkout's directory name (GitButler's pattern).

Still read-only, still window-loaded, still manually refreshed (0030 as
amended): dirty states of *sibling* worktrees ride the same manual Refresh;
the unused `git/head` token endpoints are left as they are.

## Consequences

Decision table — every row is covered by tests
(`internal/gitgraph/worktree_test.go`, `internal/server/gitworktree_test.go`,
`web/src/lib/gitgraph.test.js`):

| Checkout | Health | Dirty | Graph shows | Detail opens |
|---|---|---|---|---|
| branch, is the reader | ok | yes | row: branch + dir chip + agents + "this worktree" | owner's files (unscoped) |
| branch, sibling | ok | yes | row: branch + dir chip + agents | sibling's files via `?worktree=<branch>` |
| detached | ok | yes | row: dir chip only, trail to its HEAD | sibling's files via `?worktree=<head hash>` |
| branch or detached | ok | no | nothing new (branch pill as before) | — |
| any | prunable | — | no row, no status call (dir may be gone) | — |
| bare entry | — | — | flagged, skipped | — |
| same HEAD as sibling | ok | yes | own row per worktree (two rows, one head) | each opens its own tree |
| dirty HEAD outside window | ok | yes | row dropped; branch pill remains | — |
| unknown `?worktree=` | — | — | — | 404, never the owner's tree |

- **Easier**: "what does each agent's checkout still have uncommitted" is
  the graph's home-screen question, answered without opening anything.
- **Harder**: graph load now scales with worktree count (bounded by the
  32-status cap; ~100 ms measured at this repo's 7 worktrees).
- **Cost accepted**: two worktrees on one HEAD are addressed ambiguously by
  hash (first detached wins); the UI always prefers the branch name, so the
  ambiguity needs two *detached* checkouts on one commit to bite.
- **If wrong**: every piece is additive JSON and one layout function;
  reverting the UI restores the 0038 single-row behaviour with no stranded
  client.

## Refuse

| Temptation | Why not |
|---|---|
| Worktree path in the URL | the one thing ADR-0022 refuses; a ref names the checkout and git decides where it lives |
| Statusing prunable/bare entries | git itself says the checkout may be missing; the call can only fail or lie |
| Auto-refresh for sibling dirtiness | the poll is gone (0030 again); a per-worktree token would be three execs × N worktrees on a 5 s clock for one surface |
| Create/prune/clean worktree actions | inherited verbatim from 0022: nothing interlocks a write against an agent mid-turn |
| Worktree rows as a separate list beside the graph | the row *is* the review unit (Conductor); a side list would split the picture and double the chrome |

## Alternatives considered

| Alternative | Why not |
|---|---|
| Keep one uncommitted row, add a per-worktree dropdown to it | the rows are positional history; a dropdown hides the very state the graph exists to show |
| `git status --porcelain` piped per worktree lazily on click only | the row must carry its count to be visible at all; a "check each worktree" click sequence is the current pain, not the cure |
| Count via `for-each-ref %(worktreepath)` alone | gives the branch→worktree link (already derivable from `worktree list`), says nothing about dirtiness |
