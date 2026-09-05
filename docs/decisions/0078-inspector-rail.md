# ADR-0078: An Inspector rail follows the selected tab's owner and opens content in the center

- **Status**: proposed (the owner approved the plan on 2026-09-05: docs and
  the read-only rail ship now; the PR and Commit phases below are designed,
  not approved)
- **Date**: 2026-09-05
- **Amends**: ADR-0074's placement of the inspector inside the folder tab;
  the visual anatomy's "one center surface, one dock"
  ([design/benchmark-visual-anatomy.md](../design/benchmark-visual-anatomy.md))
- **Study**: [2026-09-05 — the right-hand inspector rail](../benchmarks/2026-09-05-inspector-rail.md)
- **Number**: 0076 is taken on `main` and by the unmerged `feat/pi-diff`
  branch; this file takes 0078 and may be renumbered at merge (precedent:
  `4502305f`).

## Context

Paseo, Orca and t3code put Files / Changes / PR in a persistent rail to the
right of the conversation, and open what the rail selects in the center.
The Cursor bar recorded that shape as adopted — "left nav + central work
area + right inspector — all resizable and collapsible; panels remember
state" — and ADR-0074 applied it *inside* the folder tab: a resizable
navigation rail with a local detail pane. The diff-editor roadmap parked
"File tree as a sidebar tab" under *Later*. The owner asked for parity with
the benchmarks and for better decisions than theirs.

Every read the rail needs exists: `/browse`, `/gitstatus`, `/gitdiff`, the
`root` precondition (ADR-0074), the file tab and its `FilePane`, the working
diff, the fleet `git.updated` feed. What did not exist was the rail itself,
per-file line counts on `gitstatus`, and a Diff view for the file tab.

Three documented refusals stand near this surface. "Explorer as the home"
(ADR-0015, 0019, 0022, 0030, 0041) refuses a surface that annexes the
product's identity. "Card or tab, not a third hatch"
(design/file-preview-roadmap.md) refuses a third host for the same file.
"Following the terminal's live cwd" (ADR-0022, 0030, 0074) refuses an
unasked retarget. None of them refuses a navigation rail that opens the
existing center tabs and pins its root.

## Decision

The desktop shell gains a third flex child after `<main>`: `<aside
id="inspector">`, a navigation and review rail with two tabs, **Changes**
(default) and **Files**. It anchors to the owner of the selected tab —
agent, terminal, or workspace, resolved from the tab id the way ADR-0022
and ADR-0030 resolve theirs — and keeps its last anchor beside tabs that
have no folder (apps). An owner that disappears clears it. The rail is
hidden on the Home dashboard and on preference panes.

The rail hosts no editor. Clicking a file opens the existing `f:` tab in
the center; clicking a change opens the same tab in a **Diff** view
(`WorkingDiff`), and the tab's "View diff" / "Open file" pair — ADR-0074's
own — swaps the two in place. The view is per-tab viewer state held by the
app, not part of the tab id or the hash.

Every read pins the anchor's folder as the `root` precondition. The first
read and an explicit **Refresh** go without it and adopt whatever folder the
owner is in now; a background read that meets a 409 becomes one blocked
line — "This terminal moved to `<cwd>`. **Follow**" — with the previous
lists kept. Live updates come from `git.updated` for the pinned root, from
the feed's open/reset, from focus/visibility, and, for terminals, from the
live cwd and dirty facts on the terminal row. No interval.

`gitstatus` grows: each change carries `add`, `del` and `binary` (one
`git diff HEAD --numstat -z -M` for tracked files; untracked files are
counted in Go with a 4 MiB cap), and the page carries `branch`, `worktree`
and `totals` over the re-anchored slice. Changes is a folder-grouped tree
whose folders sum their descendants (`+1.9k −684`), with an
`Uncommitted · +N −M` line. Beside an agent, an `All | This agent` scope
intersects the working tree with the paths that session's `edit`/`write`
tools named.

Width (260–560, default 320), open state and tab are per-viewer
preferences in localStorage. A viewer who never toggled the rail gets it
open on windows of 1440 px or wider. The rail shrinks before it hides and
hides only when even its minimum width would push the conversation column
under 640 px; the wish to have it open survives the squeeze. `Ctrl+.`
(`Cmd+.`), a tab-strip button and a palette action toggle it. The ≤767 px
column shell never shows it; mobile is untouched.

The rail is the host for later right-hand panels — PR, the browser
preview's last capture, search, tasks. One rail, never two asides. With
four or more tabs it switches to the sidebar's icon idiom.

## Decision table and acceptance

| Conditions | Action | Evidence |
|---|---|---|
| Select an agent tab | Anchor = that agent; root from `/browse`; Changes shown | `inspector.test.js anchorFor`; browser QA |
| Select a terminal tab | Anchor = that terminal; root pinned at the first read | browser QA |
| Select a file, git graph or folder tab | Anchor = that tab's owner | `anchorFor` test |
| Select an app tab, Home or a pane | Keep the last anchor; hidden on Home and panes | `anchorFor` test; browser QA |
| No anchor yet | "Open an agent or terminal to inspect its files." | browser QA |
| Anchored owner removed | Anchor cleared; empty state | `anchorFor` test |
| Background refresh, same root | Lists replaced; expansion, scroll and filter kept | browser QA |
| Background refresh answers 409 | Blocked line with Follow; lists kept; no retarget | browser QA (terminal `cd`) |
| Follow or Refresh | Read without `root`; new root pinned; expansion reset | browser QA |
| `git.updated` for the pinned root | Coalesced reload | browser QA |
| Folder without git | Files works; Changes tab absent; "Not a git repository" | `TestGitStatusOnAPlainFolderIsAState`; browser QA |
| Zero changes | "No changes. View files"; no badge | browser QA |
| Click a change | Center file tab in Diff view; a second click is idempotent | browser QA |
| Click a file | Center file tab in Editor view | browser QA |
| Diff ↔ Editor | "Open file" / "View diff" on the same tab id | browser QA |
| This agent scope | Working tree ∩ session paths; empty → "No files from this agent yet. Show all" | `scopeChanges`, `normalizeTouched` tests; browser QA |
| Terminal or workspace anchor | Scope chips absent | browser QA |
| Folder sums | Descendant totals; binary files count 0 | `groupChanges`, `changeTotals` tests |
| Per-file counts and totals | `gitstatus` add/del/binary/branch/totals over the visible slice | `TestStatusWithStats*`, `TestGitStatusReAnchorsToTheOwnerCwd` |
| Toggle by button, `Ctrl+.` or palette | Open state flipped and persisted | browser QA |
| Window leaves less than 640 px for the center | Rail shrinks, then hides; wish kept | `inspectorLayout` tests; browser QA |
| ≤767 px shell | Rail hidden | browser QA |
| Drag or arrow keys on the sizer | Width clamped to 260–max and persisted on release | `resizeEdge.test.js`; browser QA |
| Reload | Open state, width and tab restored | browser QA |
| Overlay audit, aligned rows, both themes | `ok: true`; screenshots read | browser QA |

Browser acceptance runs against an isolated fixture whose picode workspace
is a seeded dirty repository. Evidence lands in `docs/screenshots/`
(`inspector-*`) and the observed results in
[docs/plans/inspector.md](../plans/inspector.md).

## Consequences

Review is one glance away from every conversation and terminal without
leaving them, and the file tab gains a Diff view. No dependency is added;
the rail reuses the tree rows, the tab pair, the dropdown and the file
tab. `gitstatus` runs two to three more git commands per call; calls are
user- or event-driven, never periodic, and the fleet watcher is unchanged.

The folder tab (`d:`) stays as the deep-review host with its editor and
draft guards; the rail links to it ("Open as tab"). A viewer now has two
places to read the same working tree; they agree because both read the
same routes with the same precondition.

Debts named rather than hidden: a complete filename search for all three
owner kinds (`git ls-files`), a watch lease for terminals if free terminals
feel stale, the sizer idiom still copied in Sidebar and FileTreeSurface,
and the per-turn `+N −M` footer beside the conversation.

## Refuse

| Temptation | Why not |
|---|---|
| An editor or diff pane inside the rail | Duplicates ADR-0074's pane and its document lifecycle; card or tab, not a third hatch |
| Following the terminal's live cwd | The rail would retarget unasked (ADR-0022, 0030, 0074); Follow is the deliberate gesture |
| A branch selector in the rail | Switching a branch is a write (ADR-0022); the branch is shown as a fact |
| Stage, unstage, commit, push | Nothing interlocks a write against an agent mid-turn (ADR-0022, 0032, 0038, 0073); Phase 3 below is the owner's call |
| PR and GitHub in this version | ADR-0034 refuses tokens in the UI; Phase 2 below reads through the host's `gh`, and is the owner's call |
| A Search tab | `/files?q=` is agent-only and stops after 200 files; the Files filter says "loaded files" honestly, and a complete search is named debt |
| Icon tabs | Two tabs need no legend; the sidebar's icon idiom returns at four or more |
| Persisting expanded folders, filter or scope | Cheap to recreate (ADR-0030) |
| Terminals in the fleet git watcher | `PaneCwd` plus four git commands per terminal every 3 s whether or not anyone looks; the terminal row's live facts drive refetches instead |
| A second aside for the browser preview | The rail is the host; a capture panel is a tab of it |
| Virtualized lists | Bounded by `-uall` status size; revisit past ~2k changes |
| Mobile | ADR-0044/0072: supervision surfaces only |
| A hash route for the rail | Per-viewer state, like the sidebar width and `termView` |

## Later phases, designed and pending the owner's decision

**Phase 2 — PR, read-only through the host's `gh`.**
`GET /api/{agents|terminals|workspaces}/{id}/pr?root=` resolves the cwd
through the owner, checks the root, then: no `gh` on PATH →
`{status:"blocked", reason:"gh-missing"}`; `gh auth status` failing →
`{status:"blocked", reason:"gh-unauth"}`; `gh pr view --json …` in that
cwd → `{status:"ok", pr}` or `{status:"none", branch}`. States are 200, as
`gitstatus` treats "no repository". A 60 s cache per repository and branch;
no background poller against GitHub. The tab reads `PR #n`; the empty state
offers "Create in terminal", which types `gh pr create --fill` into the
owner's terminal without pressing Enter (a literal `tmux send-keys -l`
route, new). PiCode never stores a token and never creates a PR itself
(door, not cage).

**Phase 3 — Commit / Commit & Push.** Two designs. (a) The rail types
`git add -A && git commit -m '…'` (and `&& git push`) into the owner's
terminal, unsubmitted: the human presses Enter where they can see it, PiCode
writes no git, and the four ADRs' sentence stays true. (b) A server-side
commit behind an interlock: refused with 409 while any agent whose cwd
resolves to the same repository is mid-turn (runtime streaming, TUI
working, automation run) or a terminal there is working; then `git add -A`,
`git commit`, optional push behind a second confirmation, a `git.updated`
publish and a toast naming the exact command. Honest limit: the interlock
is advisory over the commit's own duration and sees only PiCode-known
agents. The mechanics are chosen after the first dogfood.

## Alternatives considered

| Alternative | Reason not selected |
|---|---|
| The rail hosts the editor and the diff | Duplicates ADR-0074's document lifecycle; the conversation stops being the hero at 320 px |
| Fold the folder tab into the rail | Loses deep review — editor, drafts, guards — that needs the center's width |
| An icon rail like the sidebar | Legend problem at two tabs; adopted as the growth rule instead |
| Follow the terminal's live cwd | Refused three times already; an unasked retarget |
| Per-file counts through lazy `/gitdiff` calls | N commands instead of one; rows would flicker in and the total would arrive last |
| Hide the rail instead of shrinking it under pressure | Loses the rail on ordinary laptops; shrinking keeps it until 640 px is at stake |
| Server-side preferences | The sidebar's width is localStorage too; the rail is per-viewer state |
