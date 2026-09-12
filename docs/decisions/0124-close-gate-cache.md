# ADR-0124: `make close` reuses a green run when the merge could not have changed it

- **Status**: accepted
- **Date**: 2026-09-12
- **Boundary**: process — what the closing rite guarantees before a fast-forward, and when a gate may be skipped
- **Amends**: ADR-0105 (the per-tree stamp reused by `make close`)

## Context

`git log --since='10 days ago' --merges --grep="^Merge branch 'main'"`
returns **151 catch-up merges** — the ritual of pulling a moved `main` into a
branch before it can fast-forward. Each one is followed by `make close`, and
`make close` re-runs the scoped gates unless the *tree hash* is unchanged
(ADR-0105's stamp). A merge always rewrites the tree, so the reuse never fires
on the case it was built for. The session paid the gates twice per branch: once
while iterating, once after the merge that touched nothing it tested.

`make close` cannot simply trust the diff either: after `git merge main`, the
path set of a branch is unchanged, while the *content* of a file in it may
have changed (main's edits merged in) — the shape that would skip the gates on
code that was never tested.

## Decision

`make ci-scoped` records what its run actually covered — the base commit, the
tree, the changed paths and each path's blob hash — in
`<git-dir>/picode-ci-scoped.json`, and `make close` reuses that run when
recomputation shows the result still holds: the tree is identical, or the merge
brought no change to any recorded path (compared by blob hash, not by name) and
nothing in the recorded set was amended. Any other case re-runs the scoped
gates. The decision lives in `scripts/ci-scope-reuse.mjs`, whose pure function
is tested row by row in `scripts/ci-scope-reuse.test.mjs`; the shell scripts
only gather git state and call it.

## Consequences

- The common case — a merge that brings unrelated work — stops paying 2–4
  minutes of gates per catch-up, 151 times in ten days.
- The gate can be skipped only when the content it tested is provably still
  there: a hash per path, not a name. A rebase, an amend, a conflict resolution
  or a touch of any covered file re-runs.
- If this is wrong, the wrongness is visible: `make close` prints the reason
  ("ci-scoped is green for these 17 paths — the merge did not touch them"), and
  a human reading a green run six commits old has the file list in front of
  them.

## Alternatives considered

- **Always re-run the scoped gates at close** (status quo). 151 merges in ten
  days, each re-testing a tree whose tested content is unchanged.
- **Reuse on "the merge touched no changed path" alone.** An amend of a file I
  already changed keeps the path set and changes the content — the gates would
  be skipped on untested code. The blob comparison is what makes the shortcut
  honest.
- **Hash the tree minus main's own changes.** Equivalent in effect, harder to
  explain and to test than a per-path hash list.
