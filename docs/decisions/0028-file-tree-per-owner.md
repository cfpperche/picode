# ADR-0028: The file tree belongs to a folder, is read-only, and highlights what changed

- **Status**: accepted
- **Date**: 2026-08-31

## Context

The owner's ask: *see the files an agent edited without leaving PiCode*.
The ADE has no file tree; the benchmarks do. What the benchmarks actually
ship, though, is narrower than an explorer (researched 2026-08-31):

1. **Agent orchestrators are diff-first.** Conductor reviews per-worktree
   diffs; Vibe Kanban pairs a board with a diff viewer; Sculptor reviews
   through a merge UI; Happy's experimental browser lists *only* changed
   files. Crystal/Nimbalyst is the closest to us: a tree per session
   worktree with the changed files highlighted. A full IDE tree appears
   only where an editor is embedded (OpenHands embeds VS Code Web).
2. **Three prior refusals stand in the way.** ADR-0015 ("LSP,
   explorer-as-home … stay refused"), ADR-0019 ("no explorer, no LSP, no
   MIME zoo"), and the diff-editor roadmap ("Explorer as the product …
   Browse API already exists; it is not the home", with "File tree as a
   sidebar tab" parked under *Later*). What they refuse is the explorer as
   the product's home. None of them refuses a read-only surface opened
   from an owner's row — ADR-0022 shipped exactly that shape for git.
3. **Most of the machinery exists.** `browseAgentDir` +
   `relUnderCwd` (ADR-0015) list one directory level confined to a cwd;
   `openFileTab` → FilePane is a complete viewer/editor; the git graph
   already solved owner-vs-identity tabs, provisional ids, and
   reload re-authorisation. Terminals lacked `/browse`; workspaces lacked
   any file route at all — and ADR-0027 means a workspace can have
   nobody in it to borrow a route from.
4. **tmux/git facts, measured here:** `git status --porcelain -z` reports
   paths relative to the repo toplevel (not the asking cwd), collapses an
   untracked directory into one `dir/` record unless `-uall`, and encodes
   a rename as two NUL fields — the parser must consume both.

## Decision

One tree per **folder**, identified by the canonical root the server
answers with, offered for all three owner kinds. Route by owner, identify
the tab by folder — ADR-0022's two identities, verbatim:

```
hash    #/tree/<w|t|a>/<id>   the owner that asked (this is what authorizes)
tab id  d:<canonical root>    what it is (this is what deduplicates)
```

The workspace becomes a file-reading owner in its own right:
`/api/workspaces/{id}/{browse,text,blob,file,gitstatus}` confine to the
registered folder (`ws_free` refused — its path is a sentinel), and file
tabs learn the `w` owner. Terminals gain `/browse` at the live pane cwd.

The tree is **read-only navigation**: expanding a folder is a lazy
one-level `/browse`; clicking a file opens the existing file tab. What
answers the owner's actual question is the decoration:
`GET …/gitstatus` returns the working-tree changes re-anchored to the
owner's cwd, the surface lists them flat in a **Changes** section on top,
and the tree marks changed files and their ancestor folders — collapsed
or not, since the change list arrives whole. No repository is a state,
not an error: `200 {"git": false}` and the tree stands undecorated.

A tree opened from a terminal pins its folder at open time; a manual
Refresh through an owner whose cwd moved renames the tab to the new root
(merging if it already exists) — never on its own.

## Consequences

- **Easier**: "what did the agent touch" is one click from the sidebar,
  and the answer opens in the editor that already exists. The surface is
  ~2 new components; the server work is wrappers around ADR-0015 helpers
  plus one 60-line status reader.
- **Easier**: empty workspaces (ADR-0027) finally have a use before the
  first agent: browse the folder you just added.
- **Harder**: three owner kinds ride every file route now; the `w` kind
  touches routes.js, FilePane, and the boot/hashchange/writeback chain in
  App.jsx. A reader who assumes two kinds will be wrong.
- **Cost accepted**: `skipFileDir` hides node_modules and friends from
  the tree with no reveal. A changed file inside a hidden folder still
  surfaces — the Changes section lists it and the file routes serve it.
- **If wrong**: the entry points are three call sites, the tab one route
  family. Removing them leaves the workspace file routes, which stand on
  their own.

## Refuse

| Temptation | Why not |
|---|---|
| Create / rename / delete / move / upload | nothing interlocks a write against an agent mid-turn in that cwd; ADR-0022 refused writes for the same reason |
| A fifth sidebar tab ("Files" as a home) | ADR-0026 sized the header for four; an entry point is not a home (ADR-0022) |
| fsnotify / polling | manual Refresh, like the git graph; a watcher per tab is unmeasured cost |
| A recursive tree endpoint | N one-level browses suffice, and the whole gitstatus already decorates collapsed folders |
| `?all=1` to reveal node_modules / .git | consistency with browse and the @-mention search; recorded as a known limit instead |
| Following the terminal's live cwd | the tab would retarget unasked (ADR-0022); Refresh is the deliberate gesture |
| Working-tree diffs in the tree | that is the diff-editor roadmap's track; here a change is a dot, not a patch |
| staged vs unstaged as separate kinds | one kind per file decorates; the distinction belongs to a future diff surface |
| Persisting expanded folders | cheap to recreate; localStorage already carries three tab maps |
| Mobile | the git graph is desktop-only too; same mold, same debt |

## Alternatives considered

| Alternative | Why not |
|---|---|
| Changed-files-only browser (Happy's shape) | half the ask; the owner also wants to *look around* — the tree costs little once Changes exists |
| Open workspace files through an agent/terminal of the workspace | breaks exactly where ADR-0027 begins: the empty workspace |
| Key the tab by owner | two agents in one folder would carry two copies of the same tree |
| Embed an editor with its own explorer (OpenHands' shape) | we are an ADE, not a worse IDE — the refusal that survives this ADR |
