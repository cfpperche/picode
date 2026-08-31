# ADR-0038: Git graph v2 — inline commit detail, uncommitted row, search, cheap auto-refresh

- **Status**: accepted (amends 0022; supersedes 0030 on manual refresh, for the graph only)
- **Date**: 2026-08-31

## Context

The graph ADR-0022 shipped is complete but ergonomically behind the tool
its lane allocator was ported from (mhutchie's Git Graph):

1. **The detail pane is a fixed bottom split.** Selecting a commit gives
   the graph 45% of the height and the diff 55% (`.gg-split-open`), which
   moves the row you clicked and cannot be resized. Git Graph instead
   expands the detail *between* the rows, directly under the click.
2. **The port already carries the hooks for that.** `branchPath(lines,
   {expandAt, expandY})` (`web/src/lib/gitgraph.js:297`) shifts every
   line below an expanded row — written for exactly this, never called.
   Likewise the original's uncommitted-changes support was deliberately
   dropped in the port (header note, `gitgraph.js:9-11`).
3. **Per-file stats are derived from the patch.** The browser downloads
   a commit's whole diff (up to 2 MiB) to draw `+N −M` per file, and a
   truncated patch yields wrong counts. `git show --numstat` answers the
   same question for the price of one cheap exec.
4. **`parents` reaches the browser and dies there.** The JSON carries
   them; `CommitDetail` never renders them.
5. **Refresh is fully manual** (ADR-0030 cites the graph as the
   precedent). Meanwhile the repo already polls in three places —
   tui-working 3s, status 15s, device 15s — and `gitinfo.Inspect`
   already pays `git status --porcelain -uall` per workspace on every
   sidebar refresh.
6. **Remote branch pills render but disappear.** `--remotes` is in the
   log and `.gg-ref-remote` exists; the pill is plain grey and reads as
   noise next to the accented local pills.
7. Measured baselines (handoff archive): layout 14ms at 10k commits,
   `?limit=2000` in 0.12s/408KB, 2,000 rows fine in the DOM. The window
   model is not the bottleneck; ergonomics are.

## Decision

The graph stays read-only and window-loaded; v2 changes how it is read,
not what it can do. Six pieces, additive JSON only:

1. **Inline detail.** Selecting a commit opens `CommitDetail` in a row
   inserted directly below the clicked one; the SVG lines detour around
   it via the existing `expandAt`/`expandY` hooks, and the circles and
   SVG height gain the same offset. The pane is resizable by a bottom
   drag handle, height persisted in `localStorage` (the file tree's
   sizer pattern). The 45/55 split dies.
2. **Uncommitted row.** When the working tree is dirty, the graph gains
   a first row "Uncommitted Changes (N)" — a hollow dot joined to HEAD
   by a dashed line, as the original draws it. The count travels in the
   graph payload (`uncommitted: {count}`) because the row must be atomic
   with `head`; the *content* stays on the existing `gitstatus`/`gitdiff`
   endpoints, fetched only on click, rendered inline like a commit.
   Layout-wise the row is a pseudo-commit prepended before `layout()` —
   the allocator treats it as the original did; no special lanes.
3. **Search.** A header input matches subject, author, and hash prefix
   over the loaded window, client-side. Matches highlight and Enter
   walks them; non-matches dim. **Rows are never hidden** — the layout
   is positional and a filtered graph would be a lie.
4. **Numstat.** `Show` runs a second exec, `git show --numstat -M -m
   --first-parent`, and `files[]` gains `add`/`del`. The browser prefers
   them over counting patch lines, which also fixes stats on truncated
   patches.
5. **Parents.** `CommitDetail` renders parent hashes as buttons that
   select the parent commit; a parent outside the loaded window doubles
   the limit once, then says so.
6. **Auto-refresh.** A new cheap endpoint `GET .../git/head` returns
   `{key, token, uncommitted}` where `token = sha256(HEAD ‖ for-each-ref
   ‖ dirty count)` — three execs, no log. The surface polls it every 5s
   while mounted and visible (`document.hidden` guard), and refetches
   the graph only when the token changes. The Refresh button stays as
   the explicit gesture and the valve. This supersedes ADR-0030's
   "manual Refresh, like the git graph" *for the graph*; the file tree
   remains manual.

Remote pills get an icon and a distinct colour, the `origin/` prefix set
off from the branch name — visual only.

## Consequences

- **Easier**: reading a commit no longer costs your place in the graph;
  the dirty state and the history finally share one picture; a stale
  graph stops lying within 5s for the price of ~3 cheap execs.
- **Harder**: `expandAt` runs for the first time since the port — it is
  vendored code with zero coverage, so tests for it land *before* the
  JSX that calls it. The uncommitted row shifts every row index by one;
  the component builds a single `rows` array once instead of scattering
  `+1`.
- **Shared surface**: `.gg-detail*` is reused by the file tree's
  WorkingDiff (ADR-0032). The inline pane wraps it in a new `.gg-inline`
  container; the shared rules do not change. Only `.gg-split*` — used by
  the graph alone — is deleted.
- **If wrong**: every piece is a small branch; the JSON changes are
  additive, so reverting any of them strands no client.

## Refuse

| Temptation | Why not |
|---|---|
| fsnotify for refresh | a new dependency and a watcher per tab — the exact unmeasured cost 0030 refused; the token poll is three execs |
| SSE / WebSocket for git | new infra with one consumer; the repo already answers "did it change?" by polling in three places |
| Server-side search (`--grep`) | a second query mode for a window the client already holds; revisit if 2,000 proves short |
| A filter that hides rows | the lane layout is positional; hiding rows redraws a graph that no longer tells the truth |
| Show Remote Branches toggle / branch dropdown | remote refs already load and render; the gap was contrast, not control |
| Write actions from the uncommitted row (stage, discard, commit) | inherited verbatim from 0022 — nothing interlocks a write against an agent mid-turn |
| Following dirty state live via the graph payload | the 5s token already carries the dirty count; two sources of truth would drift |

## Alternatives considered

| Alternative | Why not |
|---|---|
| Keep the bottom split, add a sizer | the row you clicked still moves; inline is what the reference tool does and the hooks are already ported |
| `uncommitted` as a separate request | the row must agree with `head` from the same load; a second fetch can straddle a commit |
| Reusing `WorkingDiff.jsx` for the uncommitted row | it is a full-surface single-file viewer; the row needs per-file cards, so it reuses the diff primitives (`hunksFromDiff`, `.gg-file*`) instead |
| Numstat parsed from the patch we already have | truncation breaks it today; numstat is exact and does not scale with diff size |
