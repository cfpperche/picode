# ADR-0032: A change dot expands into its diff, and the tree reaches the host file manager

- **Status**: accepted
- **Date**: 2026-08-31

## Context

ADR-0030 shipped the file tree with git-status decoration and closed on a
dot: you see *that* a file changed, not *what* changed. Its Refuse table
parked "working-tree diffs in the tree" for the diff-editor track and
answered staged-vs-unstaged with "the distinction belongs to a future
diff surface". The owner, after using V1, asked for that surface — plus a
header button that opens the folder in the operating system's file
manager (Linux, Windows+WSL, mac).

Facts that shaped the design:

1. **The diff plumbing existed.** `gitgraph.splitPatch`, `maxPatchBytes`,
   `hunksFromDiff`, `DiffLine` and the commit pane's classes render any
   unified diff; only the *question* (working tree vs HEAD, one file) was
   new.
2. **`git diff --no-index` answers with exit code 1** exactly when the
   sides differ, and the house `git()` helper treats non-zero as "no
   output" — the untracked-file case needed a looser runner. And an
   empty `git diff HEAD` is ambiguous (clean-tracked or untracked): only
   `ls-files`-empty paths may take the /dev/null fallback, or a clean
   file answers with its whole body.
3. **Opening a path in the OS was already solved** in
   `internal/backup/reveal.go` — WSL detection, explorer.exe with a
   converted or `\\wsl.localhost` UNC path, `open` on darwin, `xdg-open`
   elsewhere. It was a backup detail only by accident of history.
4. **Workspaces had no `/git*` route** — the diff had to cover all three
   owner kinds or the workspace tree's Changes section would be
   click-dead.

## Decision

Four moves, all owner-scoped like everything else in the tree:

- `GET /api/{agents|terminals|workspaces}/{id}/gitdiff?path=` answers one
  file's working-tree-vs-HEAD patch (`gitgraph.WorkingDiff`): confined by
  `relUnderCwd`, path passed after `--`, untracked files diffed against
  /dev/null (whole file as additions — also the shape for a repository
  with no commits yet), binary and truncation flags as in the commit
  pane. Clicking a Changes row opens this in a pane beside the tree; the
  tree narrows to a rail; "Open file" jumps to the editor.
- `POST …/reveal` opens the owner's folder (or a confined subpath) in the
  host file manager via the new `internal/osopen` package — reveal.go
  moved out of `internal/backup`, WSL detection deduplicated.
- The surface refreshes on `visibilitychange`/window focus — the moment
  the working tree most likely moved is the moment you come back.
- `gitinfo.Info` gains `Dirty` (porcelain `-uall` line count) and the
  sidebar's branch pill wears it as a badge.

## Consequences

- **Easier**: the V1 loop closes — dot → diff → edit, without leaving the
  tree. Staged vs unstaged needs no kind of its own: the patch shows it.
- **Cost accepted**: the badge adds one `git status` subprocess per
  agent/terminal row per list response (2s timeout, personal machine).
  If it ever hurts, the recorded fallback is an opt-in `?dirty=1`.
- **Cost accepted**: Reveal is host-local. A browser on another machine
  (tailnet phone, remote desktop) opens a window on the *server's*
  desktop. The mobile shell has no tree; the desktop remote case is
  documented here rather than gated.
- **If wrong**: the diff pane is one component and one route family; the
  tree stands without it exactly as V1 shipped.

## Refuse (V2)

| Temptation | Why not |
|---|---|
| Create / rename / delete / upload | still no interlock against an agent mid-turn (ADR-0030) |
| fsnotify / interval polling | focus-refresh covers the human loop; a watcher per tab is still unmeasured |
| Staged vs unstaged as separate kinds | the patch renders the distinction; a second kind would only re-encode it |
| Hunk-level stage/discard from the pane | that is a write; see the first row |
| `?all=1`, mobile, persisted expansion | unchanged from ADR-0030 |
| Gating Reveal to local sessions | trust boundary is the personal machine/tailnet (ADR-0007); a gate needs a "session locality" notion PiCode does not have |

## Alternatives considered

| Alternative | Why not |
|---|---|
| Reuse the commit route with a synthetic hash | working tree has no hash; the overload would poison `isHash`'s guarantee |
| Synthesize untracked patches by hand | `--no-index` produces canonical output; hand-built hunks drift from what `hunksFromDiff` expects |
| Badge via a separate polled endpoint | the gitinfo object already rides every list response the sidebar renders |
