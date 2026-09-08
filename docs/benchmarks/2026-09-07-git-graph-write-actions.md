# Study: write actions on a commit graph (Git Graph, GitLens, GitKraken, Tower, GitButler, lazygit, Conductor)

- **Date:** 2026-09-07
- **Sources:** public docs and open repositories, cited per fact. Nothing was
  cloned; closed-source claims are marked *inf.* Local receipts were measured
  in this repository with git 2.53.0 and are marked **measured**.
- **Scope:** what a graph lets you *do* to the thing you clicked, how the
  field tiers risk, and what an undo means. The in-house counterparts are
  `web/desktop/src/components/GitGraphSurface.jsx` (read-only graph) and the
  Inspector's Git menu (`web/desktop/src/lib/inspector.js:352`, six actions,
  three delivery doors — ADR-0078).

## Why now

The owner's screenshot (2026-09-07) shows the graph doing everything it was
designed to do — 250+ commits, per-worktree uncommitted rows, branch pills,
agents named on their checkouts — and nothing else. Every documented refusal
of a write (ADR-0022, 0032, 0038, 0073) carries the same sentence: *nothing
interlocks a write against an agent mid-turn.* ADR-0078 answered that
sentence in 2026-09-05 and shipped three doors that write git safely. The
graph never got them.

| Fact | Value |
|---|---|
| Actions reachable from the graph today | 0 (copy hash is not offered either) |
| Actions in the Inspector's Git menu | 6 — fetch, pull `--ff-only`, push, commit, commit+push, `gh pr create` |
| Delivery doors that already exist | 3 — prepare (`POST …/terminals/{id}/type`), run-when-idle (`…/run` + interlock), ask the agent (`POST …/agents/{id}/ask`) |
| Actions in the reference tool (mhutchie) | ~35 across 8 context menus |
| Where the six actions live | `web/desktop/src/lib/inspector.js:339-461` **and** `web/mobile/src/lib/git/actions.js` — byte-identical apart from one blank line (**measured**, `diff` of the non-comment lines is a single blank) |
| `git/head` token endpoint (ADR-0038) | shipped, then left unused by ADR-0073 — no caller in `web/` |

## What the field does

### 1. The graph is a menu surface, not a picture

mhutchie's Git Graph — the tool this repository's lane allocator was ported
from (ADR-0022) — binds a distinct context menu to **eight** targets: commit,
uncommitted-changes row, local branch (checked out and not), remote branch,
tag, stash, a file inside the commit detail, and a link. The commit menu
alone offers *Add Tag, Create Branch, Checkout, Cherry Pick, Revert, Drop,
Merge into current branch, Rebase current branch on this Commit, Reset
current branch to this Commit,* plus two clipboard entries; the local-branch
menu adds *Rename, Delete, Push Branch, Create Pull Request, Create Archive*;
the remote-branch menu adds *Delete Remote Branch, Fetch into local branch,
Pull into current branch*; the stash menu is *Apply, Pop, Drop, Create Branch
from Stash* ([Context Menus wiki]). Items disappear rather than fail: *Drop*
appears "only if topologically possible", *Push Branch* "only if remotes
exist", *Fetch into local branch* only when a matching unchecked-out local
branch exists. Every destructive item is spelled with an ellipsis — it opens
a dialog, it does not act on click.

GitLens' Commit Graph reaches the same place from the other side: right-click
targets are commit, branch, tag, author and column header; the documented
actions are cherry-pick, revert, create branch, merge, rebase, and comparison
against a common base ([GitLens Commit Graph], [GitKraken blog 2026]). The
docs do not put push/pull/fetch in the graph's own menus — those stay on the
branch UI elsewhere in the extension.

**What we take:** the taxonomy — action follows target — and the *hide, don't
disable* rule for items git would refuse anyway.

### 2. Risk is tiered by the dialog, and force is a separate word

Tower shows a confirmation dialog for force push specifically because it is
"potentially destructive", and pairs it with an undo that covers "nearly any
Git operation … including operations that would normally require a carefully
constructed git reflog rescue mission" ([Tower — Force Push in Git]).
GitKraken requires selecting Force Push and then confirming Force Push a
second time ([GitKraken — git push force]). Sublime Merge is the cautionary
tale: an open issue argues `push --force-with-lease` should not be offered
while Automatic Fetching is on, because the lease silently stops protecting
anything once a background fetch has updated the remote-tracking ref
([sublimehq/sublime_merge#1846]).

**What we take:** three things. Destructive is a *tier*, not a warning
colour. `--force-with-lease` over `--force`, never plain force. And a lease
is void if anything fetches behind the user's back — which is an argument
against ever putting an automatic fetch on a timer.

### 3. Undo is a snapshot log, not a reflog lookup

GitKraken Desktop's toolbar Undo covers checkout, commit, discard, delete
branch, remove remote, reset branch and several rebase operations, and undoes
**only the most recent action** ([GitKraken — Undo & Redo]).

GitButler goes further and is the most interesting design in the field:
before *any* major action it snapshots the whole state — virtual branch
state, uncommitted work, conflict state — into the git object database, and
every operation lands in an Undo Timeline; a restore is itself recorded as a
new oplog entry, so undo is undoable ([GitButler — Operations History],
[Operations Log]). Its CLI exposes `undo` and `oplog` as first-class
subcommands, and undo "acquires exclusive worktree access" before restoring
([gitbutler#10058]).

**What we take:** the snapshot-before-the-action idea, and the honesty that
comes with it — an undo you can *offer* is one you recorded a position for
beforehand. Not the object-database snapshot machinery: PiCode does not
own the working tree.

### 4. The worktree is a first-class row with its own lifecycle

lazygit has a Worktrees panel: `n` creates, `space` switches, `o` opens in
the editor, `d` removes the worktree together with its metadata — and `w` on
a *branch, commit or stash* creates a worktree from that item
([lazygit keybindings], [lazygit features]). VS Code needs an extension for
any of this; the popular ones wrap exactly create / list-and-switch / remove
([vscode-git-worktrees], [Git Worktree Manager]).

Conductor productizes the same three verbs around an agent: creating a
workspace "creates a Git worktree, checks out a branch in it, optionally
copies gitignored files and runs setup scripts", the branch "becomes the unit
you review, push, and turn into a pull request", and the end state is
"archiving the workspace when the task is done" ([Conductor — Git
Worktrees]). Crystal/Nimbalyst and Claude Squad take the same shape: one
worktree per agent session, diffs reviewed across worktrees in one window
([Nimbalyst], [orchestrator round-ups 2026]).

**What we take:** create-worktree-from-this-row as a graph action, and the
observation that PiCode already has both halves — `make worktree` and the
agent store — with nothing joining them in the UI.

### 5. Where the field's safety actually comes from

Every one of these tools has a trivial answer to *"who else is writing this
tree right now?"* — nobody. Conductor's own docs concede the isolation is
"development isolation, not a security boundary"; the agent runs as your
user. One worktree per task means a one-click app-side commit is safe by
construction. ADR-0078 already recorded this asymmetry: **PiCode's unit is an
agent that *has* a cwd, and several agents and terminals may share one
folder.** That is the whole reason four ADRs refused writes.

## What git itself enforces (measured, git 2.53.0)

The benchmarks lean on git's own refusals more than their docs admit. These
are free interlocks PiCode inherits — and, more usefully, states PiCode can
*predict* from the graph payload it already loads, so the menu can explain
instead of letting a command fail in a terminal.

| Attempted | git's answer | exit |
|---|---|---|
| `git add` while another process holds `.git/index.lock` | `fatal: Unable to create '…/index.lock': File exists.` + "Another git process seems to be running in this repository" | 128 |
| `git switch feat-x` when `feat-x` is checked out in a sibling worktree | `fatal: 'feat-x' is already used by worktree at '…'` | non-zero |
| `git branch -D feat-x`, same condition | `error: cannot delete branch 'feat-x' used by worktree at '…'` | non-zero |
| `git worktree remove <dir>` with modified or untracked files | `fatal: '…' contains modified or untracked files, use --force to delete it` | 128 |

Two more receipts that shape the design:

- **Per-branch upstream and worktree cost nothing.** Extending the graph's
  existing `for-each-ref` call from `%(objectname) %(refname)` to add
  `%(upstream:short) %(upstream:track) %(worktreepath)` runs in the same
  9–22 ms on this repository (**measured**) and yields, per branch, the
  upstream name, `[ahead 172]`-style tracking, and the checkout that holds
  it. Today the graph asks for none of it and the branch chip gets the same
  facts from a separate `rev-list` in `gitstatus`.
- **A push, unlike everything else, needs a credential the service does not
  have.** ADR-0078's own comparison table already recorded it: the systemd
  service environment has no ssh-agent socket and no pinentry, so a
  server-side push "is the first thing to fail here". The terminal doors
  exist because of this, not merely out of caution.

## Comparison

| Tool | Targets with menus | Destructive gating | Undo | Worktrees | Knows who else is writing |
|---|---|---|---|---|---|
| mhutchie Git Graph | 8 | dialog per item; items hidden when impossible | no | no concept | no |
| GitLens Commit Graph | 5 | VS Code dialogs *(inf.)* | no | via extensions | no |
| GitKraken Desktop | commit/branch/tag/stash | two-step for force push | last action only | listed as supported | no |
| Tower | commit/branch/tag/stash *(inf.)* | dialog for force push | "nearly any operation" | yes *(inf.)* | no |
| GitButler | commit/branch/stack | snapshot before every major action | full oplog timeline, undo-able undo | yes | no (single working dir by design) |
| lazygit | every panel | typed confirm on some *(inf.)* | no | dedicated panel, `w` from branch/commit/stash | no |
| Conductor / Crystal | workspace row | archive flow | no | **the unit of the product** | not needed — one agent per tree |
| **PiCode today** | **0** | — | — | rows for every worktree, agents named | **yes — and it is the only one** |

## What PiCode adopts

1. **Action follows target.** The eight-menu taxonomy, trimmed to what
   PiCode can honour, bound to the rows the graph already draws: commit,
   branch pill, remote pill, tag pill, uncommitted row, worktree row.
2. **Hide what git would refuse.** The payload already knows which worktree
   holds a branch; a *Checkout* that git will reject with "already used by
   worktree at …" is replaced by *Open that worktree* before the click, not
   by a 409 after it.
3. **Risk tiers, not warning colours.** Additive / history-moving /
   publishing-or-destroying, with the doors and the confirmations attached to
   the tier — the repository already ships typed confirmation for a
   destructive act (CLI uninstall, ADR-0093).
4. **`--force-with-lease`, never `--force`**, and no background fetch on a
   timer that would void the lease (sublime_merge#1846).
5. **Snapshot-before-action, honestly scoped.** Record HEAD and the reflog
   position before a tier B/C delivery, and offer an undo that *prepares* the
   inverse command in the terminal. PiCode does not own the tree; it must not
   claim GitButler's guarantee.
6. **Worktree lifecycle as graph rows**, with the one move no benchmark can
   make: create a worktree *and* the agent that lives in it, from the commit
   or branch under the cursor.

## What PiCode refuses

| Temptation | Why not |
|---|---|
| Running git in the service process | ADR-0078's measured gap: no ssh-agent, no pinentry, no credential helper. Push fails exactly where the terminal succeeds |
| Drag-and-drop merge/rebase (GitKraken, GitButler) | A drag is not a confirmation; the target of a rebase is the one thing that must be read before it happens |
| An automatic background fetch | Voids a `--force-with-lease` (sublime_merge#1846) and adds a timer against a remote for a surface nobody may be looking at |
| Interactive rebase | Needs an editor loop the graph cannot host; the terminal already has one, and an agent handles conflicts better than a dialog |
| `git filter-branch`, `gc --prune=now`, plain `--force` push | No tier, no undo, no owner asked for them |
| A second execution surface for "safe" commands | Two paths to write git is two security models; the doors of ADR-0078 are the doors |

## Sources

- [Context Menus · mhutchie/vscode-git-graph Wiki](https://github.com/mhutchie/vscode-git-graph/wiki/Context-Menus)
- [mhutchie/vscode-git-graph](https://github.com/mhutchie/vscode-git-graph)
- [GitLens Commit Graph](https://help.gitkraken.com/gitlens/gl-commit-graph/)
- [GitLens vs VS Code Git Graph — use cases (2026)](https://gitkraken.com/blog/gitlens-vs-vs-code-git-graph-use-cases-from-reddit-2026)
- [GitKraken Desktop — Undo & Redo](https://help.gitkraken.com/gitkraken-desktop/undo-and-redo/)
- [GitKraken — How to Git Push Force](https://gitkraken.com/learn/git/problems/git-push-force)
- [Tower — Force Push in Git](https://www.git-tower.com/blog/force-push-in-git)
- [sublimehq/sublime_merge#1846 — force-with-lease with automatic fetching](https://github.com/sublimehq/sublime_merge/issues/1846)
- [GitButler — Operations History](https://docs.gitbutler.com/features/timeline)
- [GitButler — Operations Log (CLI)](https://docs.gitbutler.com/cli-guides/cli-tutorial/operations-log)
- [gitbutlerapp/gitbutler#10058 — v1 of many gitbutler cli commands](https://github.com/gitbutlerapp/gitbutler/pull/10058)
- [lazygit — Keybindings](https://github.com/jesseduffield/lazygit/blob/master/docs/keybindings/Keybindings_en.md)
- [lazygit — Features](https://lazygit.dev/features/)
- [alexiszamanidis/vscode-git-worktrees](https://github.com/alexiszamanidis/vscode-git-worktrees)
- [Git Worktree Manager (VS Code Marketplace)](https://marketplace.visualstudio.com/items?itemName=jackiotyu.git-worktree-manager)
- [Conductor — Git worktrees](https://conductor.build/docs/concepts/git-worktrees)
- [Nimbalyst — best agent management tools 2026](https://nimbalyst.com/blog/best-agent-management-tools-2026/)
- [Augment Code — open-source agent orchestrators (2026)](https://www.augmentcode.com/tools/open-source-agent-orchestrators)
