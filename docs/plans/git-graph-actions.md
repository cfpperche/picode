# Git graph — the write generation

> **Status: all four phases shipped (2026-09-08); reviewed adversarially the same day — see ADR-0096's amendment.**
> The owner chose **phases 1–2 first**, **tier C reachable by run-when-idle
> behind the opt-in plus a typed confirmation**, and **the command composer
> moving to the server in phase 2** (§10). The boundary decision is recorded
> in **ADR-0096**, which amends the *Refuse* tables of ADR-0022, 0032, 0038
> and 0073.
>
> **Naming.** "Git graph v2" is already taken: ADR-0038 shipped under that
> name (inline detail, uncommitted row, search, numstat). What the owner
> asked for on 2026-09-07 — *"expandir as capacidades … operações que fazem
> modificações"* — is the generation after it. This document calls it **the
> write generation**; the ADR will be v3 if the owner prefers a number.
>
> **Study:** [2026-09-07 — write actions on a commit graph](../benchmarks/2026-09-07-git-graph-write-actions.md).

## 1. Where we actually are

The graph is not missing a write *engine*. It is missing the wiring to one
that shipped two days ago and never reached it.

| Piece | State |
|---|---|
| Graph payload: commits, refs (head/remote/**tag**), worktrees + agents, per-worktree uncommitted | shipped (ADR-0022, 0038, 0073) |
| Three delivery doors — **prepare** (`POST …/terminals/{id}/type`), **run when idle** (`…/run` + `repoBusy` interlock), **ask the agent** (`POST …/agents/{id}/ask`) | shipped (ADR-0078 stages 1–3), used by the Inspector rail and by mobile (ADR-0095) |
| The six actions those doors carry | fetch · pull `--ff-only` · push · commit · commit+push · `gh pr create` — all **HEAD-and-upstream**; none takes a target |
| Graph actions | **zero.** Not even *copy hash* |
| `GET …/git/head` cheap-token endpoint | shipped in ADR-0038, orphaned by ADR-0073, **no caller in `web/`** |
| `gitActionCommand` / `gitActions` / `askGitPrompt` | **duplicated verbatim** in `web/desktop/src/lib/inspector.js:339` and `web/mobile/src/lib/git/actions.js` (measured: the two blocks differ by one blank line) |

The gap is structural, not incidental. The rail's actions act on *the
checkout you are in*. The graph's entire value is that you **point at
something** — a commit, a branch pill, a remote pill, a sibling worktree's
dirty row. Every benchmark binds its write vocabulary to exactly those
targets (mhutchie: 8 context menus, ~35 items). PiCode has the targets drawn
on screen and no verbs attached to them.

## 2. The one decision that unblocks the rest

**Git keeps running only in the user's shell or an agent's turn. No fourth
door, and never in the service process.**

This is not caution, it is a measurement ADR-0078 already recorded: the
systemd service has no ssh-agent socket, no pinentry and no credential
helper, so a server-side `push` fails precisely where the terminal succeeds.
Adding a second execution surface would also mean a second security model for
the same act. Everything below composes a command string client-side — as
`gitActionCommand` does today — and hands it to a door that already exists.

Consequence to accept up front: **the graph learns the result by watching,
not by return value.** §6 says how.

## 3. Risk tiers

The field gates by dialog per item (mhutchie), by two-step confirm
(GitKraken), or by undo (Tower, GitButler). PiCode should gate by an explicit
tier, because the tier — not a warning colour — decides which doors are even
offered. The repository already ships typed confirmation for a destructive
act (CLI uninstall, ADR-0093).

| Tier | Actions | Doors offered | Gate |
|---|---|---|---|
| **0 — no git** | copy hash / subject / branch name; open a worktree's tab; reveal a file | — | none |
| **A — additive, local, reflog-recoverable** | fetch `--prune` · create branch · create tag · create worktree · stash push · checkout a clean tree | prepare · run-if-idle · ask | the command preview in the form |
| **B — moves HEAD or history, still recoverable** | pull `--ff-only` · merge `--no-ff` · rebase onto · cherry-pick · revert · reset `--soft`/`--mixed` · stash pop/apply · delete a **merged** local branch | prepare · run-if-idle · ask | preview **+ the target row stays highlighted while the form is open** |
| **C — publishes, or destroys work** | push · push `-u` · push `--force-with-lease` · delete an **unmerged** local branch · delete a **remote** branch · delete a tag · reset `--hard` · discard / clean `-fd` · stash drop · worktree remove `--force` | **prepare by default**; run-if-idle only with the existing opt-in **and** a typed confirmation; ask is always allowed (the agent is a reviewer) | typed confirmation of the ref or `delete` |
| **Never** | plain `push --force` · `filter-branch` · `gc --prune=now` · `reset --hard` aimed at a *sibling* worktree | — | — |

Two rules from the study travel with the tiers:

- **`--force-with-lease`, never `--force`.** And no automatic background
  fetch on a timer: an automatic fetch voids the lease
  ([sublime_merge#1846]). The graph's refresh stays user-driven, as ADR-0030
  and 0073 already have it.
- **Hide what git would refuse; do not disable it.** mhutchie hides *Drop*
  when it is not topologically possible and *Push Branch* when there is no
  remote. PiCode can go one better — see §5.

## 4. Target → action

Adopted from mhutchie's taxonomy, trimmed to what PiCode can honour, and
bound to the rows the graph already draws. PiCode-only rows are marked ★.

| Target (row / pill) | Tier 0 | Tier A | Tier B | Tier C |
|---|---|---|---|---|
| **Commit** | copy hash, copy subject | create branch here · create tag · checkout (detached) · ★ create worktree here · ★ start an agent here | cherry-pick onto current · revert · merge into current · rebase current onto · reset current (soft/mixed) | reset current `--hard` |
| **Local branch pill** | copy name | checkout · ★ create worktree for this branch | rename · merge into current · rebase current on · delete (merged) | push · push `-u` · push `--force-with-lease` · delete (unmerged) |
| **Remote branch pill** | copy name | fetch into local · checkout as local | pull into current (`--ff-only`) | delete remote branch |
| **Tag pill** | copy name | — | — | push tag · delete tag |
| **Uncommitted row** (any worktree) | open its files (shipped) | stash push | commit · commit + push | discard all · clean `-fd` · stash drop |
| **★ Worktree row** | open terminal here · open its Files tab | create worktree · ★ create worktree **and** the agent that lives in it | prune | remove · remove `--force` |
| **★ Repository header** | — | fetch `--prune` | pull `--ff-only` · create branch | — |

Stashes get actions only if the payload gains a `stashes[]` list (§7); they
are the one mhutchie menu with no counterpart in the graph today.

## 5. The four moves no benchmark can make

This is the argument for building it here rather than telling people to open
GitKraken.

1. **The menu names who else is in the repository, before the click.**
   `repoBusy` (`internal/server/git_run.go:118`) already enumerates every
   agent mid-turn and every busy terminal in the git common dir. Today it
   surfaces as a 409 *after* you act. In the graph it becomes a line in the
   menu header — *"Atlas is mid-turn in this repository"* — and it pre-selects
   *prepare* over *run*. No other git client knows the answer at all.

2. **A checkout git will refuse is replaced by the thing you meant.**
   Measured: `git switch feat-x` when `feat-x` lives in a sibling worktree
   fails with *"already used by worktree at …"*; `git branch -D` the same. The
   payload already carries `worktrees[]` with their branches, so the menu
   offers **"Open that worktree"** (a tab PiCode already has) instead of a
   command that cannot work. `%(worktreepath)` on the refs makes this exact
   rather than inferred.

3. **Judgment calls go to an agent, not a dialog.** Merge, rebase and
   cherry-pick have one honest failure mode: conflicts. ADR-0078 stage 3
   already sends a prompt through the agent's own channel, and the agent
   resolves conflicts with the repository's rules in context. `askGitPrompt`
   gains target-bound wording — *"rebase feat/x onto main; if it conflicts,
   stop and tell me which files"*. A dialog cannot do this.

4. **Worktree + agent in one gesture.** Conductor, Crystal and Claude Squad
   are entire products built on *one worktree per task*, and PiCode owns both
   halves already (`make worktree`, the agent store) with nothing joining
   them in the UI. "★ Start an agent here" on a commit or branch row is the
   single highest-value item in §4, and it is a *write* the owner explicitly
   asked for (*"delete branch/worktree"* implies the create side too).

## 6. How an action reports back

Delivering a command through a terminal returns "typed" or "submitted", not
"merged". AGENTS.md requires motion and optimistic UI for state that takes
time; a static flash then "all done" is FAIL.

Revive **`GET …/git/head`** — built by ADR-0038, orphaned by ADR-0073, still
three cheap execs returning `{key, token, uncommitted}`. After a delivery:

1. the target row enters a **pending** state naming the action and the door;
2. the surface polls `git/head` every second for up to 30 s — *only while an
   action is pending, only while the tab is visible*, never on a standing
   timer (ADR-0030's refusal stands);
3. the token changes → refetch the graph, land the row, toast the result;
4. 30 s with no change → *"Still running. Watch it in Terminal <name>."*
   with a link. Honest, not a spinner forever.

For an **ask**, the agent's own tab is the progress view; the toast says
which channel took it (`askedNote` already does this).

**Undo (phase 4).** Before any tier B/C delivery, record HEAD and the current
reflog position of the target worktree in the delivery record. Undo *prepares*
the inverse command in the terminal (`git reset --hard <recorded>`, `git
branch <name> <recorded>`) — it never runs it, and the copy says so. This is
GitButler's snapshot-first idea scoped honestly: PiCode does not own the
working tree and must not claim GitButler's guarantee.

## 7. Server changes (all additive)

| Change | Cost | Why |
|---|---|---|
| `loadRefs`: extend the existing `for-each-ref` format with `%(upstream:short) %(upstream:track) %(worktreepath)` | **zero measured** — 9–22 ms, same single call | Per-branch upstream, ahead/behind and the checkout that holds it. Feeds tiering, hiding, and move #2 |
| `remotes[]` from one `git remote` | one exec, only when the menu opens | *push -u origin* needs a remote name; "no remotes" hides push entirely (mhutchie's rule) |
| `merged` flag per local branch (`for-each-ref --merged HEAD`) | one exec | Decides B (delete merged) vs C (delete unmerged) |
| `stashes[]` (`git stash list`) — **only if stash actions are adopted** | one exec | The one mhutchie menu with no row today |
| `POST …/git/deliver` — **optional, see open question Q3** | — | A thin composer that validates target + action server-side and forwards to `type`/`run`/`ask`, so desktop and mobile stop composing shell strings independently |

No new execution route. No new dependency. `git/head` and the three doors are
reused as they are.

## 8. Client changes

- **Kill the duplication first.** Move `shellQuote`, `refArg`,
  `gitActionCommand`, `gitActions`, `branchChip`, `askGitPrompt`, `askedNote`
  into `web/shared/domain/gitCommands.js`, imported by desktop, mobile and
  the graph. They are identical today (measured); a third copy for the graph
  would be the moment this becomes unmaintainable. This is a prerequisite,
  not a nice-to-have.
- Extend the composer with a **target**: `gitActionCommand(action, {target,
  branch, upstream, message})`, where a target is a validated ref name (the
  existing `refArg` pattern) or a full 40-hex hash (`isHash`, already in
  `internal/gitgraph/gitgraph.go:305`, needs a JS twin). `--end-of-options`
  where a ref could be read as a flag, as `loadCommits` already does.
- One `GitActionMenu.jsx` used by every target, driven by a pure
  `graphActions(target, graph, ctx)` under `node --test`. Radix dropdown +
  the shipped `InspectorCommitDialog` pattern for forms; no new widget.
- **Mobile is a long-press sheet over the same pure module** (ADR-0095's
  independence holds: shared logic, its own presentation).

## 9. Phases

Each phase ends green and shippable on its own.

| Phase | Content | Risk | Ends when |
|---|---|---|---|
| **0 — ADR** | ADR-0096: amends the four Refuse tables, records the tiers, the doors, `--force-with-lease`, the decision table | none | owner accepts |
| **1 — targets and truth** *(no writes)* | refs gain upstream/track/worktree/merged; remotes; context menus with **tier 0 only** (copy, open worktree, open terminal); the "who else is here" header line; command *preview* with no send button | none — read-only | menus open on every target; `git/head` still unused |
| **2 — tier A + B** | create branch/tag/worktree · checkout · fetch/pull · merge/rebase/cherry-pick/revert · reset soft/mixed · commit and commit+push from the uncommitted row — through all three doors; pending rows + `git/head` watch | medium | decision table rows for A and B are covered by tests + browser QA on a scratch fixture |
| **3 — tier C** | delete local/remote branch · delete tag · discard · clean · stash drop · worktree remove · push and push `--force-with-lease`, behind typed confirmation; run-if-idle requires the opt-in *and* the typed confirm | high | every C row has a test **and** a screenshot of its confirmation |
| **4 — the PiCode moves** | ★ create worktree **and** its agent from a commit/branch · undo-prepares-the-inverse · target-bound `askGitPrompt` wording for conflict handoff | medium | the two ★ rows are the demo |

Phase 1 alone already answers part of the screenshot's complaint (the graph
is inert) at zero risk, and is the substrate everything else needs.

## 10. Decisions and what is still open

Answered by the owner, 2026-09-07:

| # | Question | Answer |
|---|---|---|
| **Q1** | Scope of the first cut | **Phases 1–2 now.** Tier C is where an undo matters, and undo is phase 4 |
| **Q2** | Does run-when-idle extend to tier C? | **Yes — with the existing opt-in *and* a typed confirmation.** Refusing outright would make `git push` one-click and `push --force-with-lease` unreachable, pushing people into a raw terminal with no interlock at all. ADR-0078's interlock stays advisory and the ADR says so |
| **Q3** | Does the command composer move to the server? | **Yes, in phase 2** — `POST …/git/deliver` validates target + action and forwards to `type` / `run` / `ask`. Two clients composing shell strings for ~35 actions is two places to get quoting wrong; the server already owns `repoBusy` and `WorktreeOfRef` |

Still open, with the recommendation standing until the owner says otherwise:

| # | Question | Recommendation |
|---|---|---|
| **Q4** | **Stash: in or out?** No stash row exists today; it costs a payload list and five menu items | Out of phases 1–3; revisit after. It is the one benchmark menu with no PiCode precedent |
| **Q5** | **Does `askGitPrompt`'s "never force-push" wording change?** Tier C offers `--force-with-lease` | Amend it to "never plain `--force`; `--force-with-lease` only when I ask for it by name". Needed in phase 3, not before |
| **Q6** | **Is "start an agent here" a graph action or a workspace action?** It creates an agent, which is the sidebar's job today | Graph action, phase 4. The commit or branch under the cursor *is* the argument, and that is the whole point |

## 11. Refuse (carried into the ADR)

| Temptation | Why not |
|---|---|
| Git in the service process | Measured gap: no ssh-agent, no pinentry, no credential helper (ADR-0078). Push fails exactly where the terminal succeeds |
| Drag-and-drop merge/rebase | A drag is not a confirmation, and the target of a rebase is the one thing that must be read before it happens |
| Automatic background fetch | Voids `--force-with-lease` ([sublime_merge#1846]) and adds a timer against a remote for a surface nobody may be looking at |
| Interactive rebase in the graph | Needs an editor loop the graph cannot host; the terminal has one and an agent handles conflicts better than a dialog |
| A standing `git/head` poll | ADR-0030 and 0073 refused it; the watch here is bounded to a pending action and a visible tab |
| A filter that hides rows after an action | ADR-0038: the lane layout is positional; a filtered graph is a lie |
| Plain `--force`, `filter-branch`, `gc --prune=now` | No tier, no undo, nobody asked |
| A third copy of the command composer | §8 — it is already duplicated twice; the graph must not be the third |

[sublime_merge#1846]: https://github.com/sublimehq/sublime_merge/issues/1846
