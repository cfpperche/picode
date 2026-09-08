# ADR-0096: The git graph acts on what you point at, through the doors that already write

- **Status**: accepted (the owner approved the direction on 2026-09-07 —
  phases 1–2 first, tier C reachable by run-when-idle behind the opt-in plus
  a typed confirmation, and the command composer moving to the server)
- **Date**: 2026-09-07
- **Amends**: the *Refuse* tables of ADR-0022 (graph is read-only),
  ADR-0032 (hunk stage/discard), ADR-0038 (writes from the uncommitted row)
  and ADR-0073 (worktree writes) — the same sentence ADR-0078 already
  answered for the Inspector rail
- **Extends**: ADR-0078 (the three delivery doors and the interlock)
- **Study**: [2026-09-07 — write actions on a commit graph](../benchmarks/2026-09-07-git-graph-write-actions.md)
- **Plan**: [docs/plans/git-graph-actions.md](../plans/git-graph-actions.md)
- **Number**: 0095 is the highest on `main` and in `.worktrees/*/docs/decisions`
  at the moment of writing; this file takes 0096 and may be renumbered at
  merge (precedent: `4502305f`).

## Context

The graph shipped complete and inert. ADR-0022 refused every write with one
sentence — *nothing interlocks a write against an agent mid-turn* — and
ADR-0032, 0038 and 0073 inherited it verbatim. ADR-0078 answered that
sentence on 2026-09-05 and shipped three doors that write git safely:
**prepare** (`POST …/terminals/{id}/type`, literal keystrokes, no Enter),
**run when idle** (`…/run`, types and submits behind the `repoBusy`
interlock) and **ask the agent** (`POST …/agents/{id}/ask`, through the
channel that already carries prompts). Mobile took the same three doors
(ADR-0095). The graph took none of them.

The gap is structural. The rail's six actions — fetch, pull `--ff-only`,
push, commit, commit+push, `gh pr create` — all act on *the checkout you are
in*; none takes a target. The graph's whole value is that you **point at
something**: a commit, a branch pill, a remote pill, a tag, a sibling
worktree's dirty row. Every tool in the study binds its write vocabulary to
exactly those targets — mhutchie's Git Graph, the tool this repository's lane
allocator was ported from, has eight context menus and about thirty-five
items. PiCode draws the targets and attaches no verb to any of them, not even
*copy hash* (verified: no clipboard call in `GitGraph.jsx`, `CommitDetail.jsx`
or `GitGraphSurface.jsx`).

Facts that decide the design:

1. **Git in the service process is the one thing that cannot work.**
   ADR-0078's own comparison table measured it: the systemd service has no
   ssh-agent socket, no pinentry and no credential helper, so a server-side
   `push` "is the first thing to fail here" — precisely where the user's
   terminal succeeds.
2. **Git already refuses the dangerous multi-worktree cases**, and PiCode can
   predict them. Measured here with git 2.53.0: `switch` and `branch -D` on a
   branch checked out in a sibling worktree fail naming that worktree;
   `worktree remove` on a dirty checkout demands `--force`; a held
   `.git/index.lock` fails with exit 128 and a distinctive message.
3. **The state the menu needs is free.** Extending the graph's existing
   `for-each-ref` call from `%(objectname) %(refname)` to add
   `%(upstream:short) %(upstream:track) %(worktreepath)` runs in the same
   9–22 ms on this repository (measured) and yields, per branch, the upstream,
   the ahead/behind counts and the checkout that holds it.
4. **`git/head` exists and is orphaned.** ADR-0038 built the three-exec token
   endpoint; ADR-0073 stopped using it. There is no caller in `web/`
   (verified). It is exactly the completion signal a fire-and-watch action
   needs.
5. **The command composer is already duplicated.** `shellQuote`, `refArg`,
   `gitActionCommand`, `gitActions`, `branchChip`, `askGitPrompt` and
   `askedNote` exist twice — `web/desktop/src/lib/inspector.js:339` and
   `web/mobile/src/lib/git/actions.js` — differing by one blank line
   (measured). The graph would be the third copy.
6. **PiCode is the only tool in the study that knows who else is writing the
   tree.** Conductor, Crystal, GitButler and every desktop client have a
   trivial answer — one worktree per task, nobody else. `repoBusy`
   (`internal/server/git_run.go:118`) computes the real answer already, and
   spends it on a 409 *after* the act.

## Decision

The graph gains a **target-bound action vocabulary, delivered only through
ADR-0078's three doors**. Git runs in the user's shell or inside an agent's
turn, never in the service process; no fourth door is created. Five parts:

1. **Action follows target.** Every row and pill the graph already draws
   carries a menu: commit, local branch, remote branch, tag, uncommitted row
   (per worktree), worktree row, repository header. The vocabulary is
   mhutchie's taxonomy trimmed to what PiCode can honour, and an item that
   git would refuse is **hidden, not disabled** — the payload knows enough to
   decide.

2. **Risk is a tier, not a warning colour.** Four tiers govern which doors
   are offered and what gate stands in front:

   | Tier | Actions | Doors | Gate |
   |---|---|---|---|
   | **0 — no git** | copy hash / subject / branch name; open a worktree's tab; open a terminal there | — | none |
   | **A — additive, local, reflog-recoverable** | fetch `--prune` · create branch · create tag · create worktree · checkout a clean tree · stash push | prepare · run-when-idle · ask | the command preview |
   | **B — moves HEAD or history, recoverable** | pull `--ff-only` · merge `--no-ff` · rebase onto · cherry-pick · revert · reset `--soft`/`--mixed` · stash pop/apply · delete a **merged** local branch | prepare · run-when-idle · ask | preview, with the target row held highlighted while the form is open |
   | **C — publishes, or destroys work** | push · push `-u` · push `--force-with-lease` · delete an **unmerged** local branch · delete a remote branch · delete a tag · reset `--hard` · discard · clean `-fd` · stash drop · worktree remove `--force` | prepare (default) · run-when-idle **only** with the existing `picode-inspector-run` opt-in **and** a typed confirmation · ask | typed confirmation of the ref name, or the word `delete` |
   | **Never** | plain `push --force` · `filter-branch` · `gc --prune=now` · `reset --hard` aimed at a *sibling* worktree | — | — |

   `--force-with-lease` is offered; plain `--force` is not. No automatic
   background fetch is added on any timer: an automatic fetch voids the lease
   ([sublimehq/sublime_merge#1846]), and ADR-0030 and 0073 already refused a
   standing poll.

3. **The menu names who else is in the repository before the click, and
   offers the move git would allow.** `repoBusy`'s answer becomes a line in
   the menu header — *"Atlas is mid-turn in this repository"* — and
   pre-selects *prepare* over *run*. A *Checkout* of a branch that lives in a
   sibling worktree is replaced by **Open that worktree**, a tab PiCode
   already has, rather than a command git will reject.

4. **An action reports back by watching, not by returning.** Delivering a
   command yields "typed" or "submitted", never "merged". The target row
   enters a pending state naming the action and the door; the surface polls
   `GET …/git/head` once a second for at most 30 s — **only while an action
   is pending and the tab is visible**, never on a standing timer — refetches
   when the token changes, and after 30 s says *"Still running. Watch it in
   Terminal <name>."* An **ask** reports through the agent's own tab, and the
   toast says which channel took it (`askedNote`).

5. **One composer, on the server, in phase 2.** The duplicated client helpers
   move first to `web/shared/domain/gitCommands.js`, gaining a validated
   `target` (a ref matching the existing `refArg` pattern, or a full 40-hex
   hash), with `--end-of-options` wherever a ref could be read as a flag —
   as `loadCommits` already does. Then `POST …/git/deliver` validates target
   and action server-side and forwards to `type` / `run` / `ask`: the server
   already owns `repoBusy` and `WorktreeOfRef`, and two clients composing
   shell strings for ~35 actions is two places to get quoting wrong.

Payload additions are additive JSON: per-branch `upstream`, `ahead`,
`behind`, `worktree` and `merged`; a `remotes[]` list. Stashes are out of the
first cut (see *Still open*).

Delivery order, approved by the owner: **phase 1** — refs' new fields,
menus with tier 0 only, the "who else is here" line, a command preview with
no send; **phase 2** — tiers A and B through all three doors, pending rows,
the `git/head` watch, and the server composer; **phase 3** — tier C behind
its typed confirmation; **phase 4** — create a worktree *and* the agent that
lives in it from a commit or branch, and an undo that **prepares** the
inverse command (`git reset --hard <recorded>`, `git branch <name>
<recorded>`) from a HEAD and reflog position recorded before every tier B/C
delivery. The undo prepares; it never runs, and the copy says so.

## Consequences

- **Easier**: the surface that shows every worktree, every branch and every
  agent finally lets you act on them, without leaving it and without a raw
  terminal. The three doors mean the same action is available to a human at a
  prompt, to a human who trusts the interlock, and to the agent that owns the
  tree.
- **Easier, and unique**: a menu that says who else is writing before you
  click, and that answers "checkout this branch" with the worktree that
  already holds it. No other client in the study can do either.
- **Harder**: the graph gains an interaction model it never had — menus,
  forms, confirmations, pending rows — and the first UI in this repository
  where a wrong click destroys work. Every tier C row needs a test *and* a
  screenshot of its confirmation before it ships.
- **Cost accepted**: the interlock stays **advisory**. It sees only what
  PiCode knows, only at the moment it looks, and a race of milliseconds
  remains — ADR-0078's own words. Extending run-when-idle to tier C widens
  what that race can cost. The owner weighed this against the alternative and
  chose it: refusing outright makes `git push` one-click and `push
  --force-with-lease` unreachable, which pushes people into a raw terminal
  where there is no interlock at all.
- **Cost accepted**: an action's result is observed, not returned. A command
  that hangs at a credential prompt shows as pending for 30 s and then points
  at the terminal. This is the price of keeping git in the user's shell, and
  it is the same price the Inspector rail already pays.
- **Cost accepted**: the undo prepares an inverse command instead of
  guaranteeing a restore. GitButler snapshots the object database because it
  owns the working directory; PiCode does not, and must not claim the
  guarantee.
- **If wrong**: every payload field is additive and every menu item is a row
  in one pure module (`graphActions`). Removing the module leaves the graph
  exactly as ADR-0073 left it, with no stranded client — and `git/deliver` is
  a thin forwarder over routes that already exist.

## Still open

Recorded rather than hidden. The recommendation stands until the owner says
otherwise, and none of them blocks phases 1–2.

| # | Question | Recommendation |
|---|---|---|
| Q4 | **Stash: in or out?** No stash row exists today; it costs a payload list and five menu items | Out of phases 1–3; revisit after. It is the one benchmark menu with no PiCode precedent |
| Q5 | **`askGitPrompt` says "never force-push"**, and tier C offers `--force-with-lease` | Amend the wording in phase 3 to "never plain `--force`; `--force-with-lease` only when I ask for it by name" |
| Q6 | **Is "start an agent here" a graph action or a workspace action?** | Graph action, phase 4. The commit or branch under the cursor *is* the argument, which is the whole point |

## Refuse

| Temptation | Why not |
|---|---|
| Running git in the service process | Measured gap (ADR-0078): no ssh-agent, no pinentry, no credential helper. Push fails exactly where the terminal succeeds, and a second execution surface is a second security model for one act |
| Drag-and-drop merge and rebase (GitKraken, GitButler) | A drag is not a confirmation, and the target of a rebase is the one thing that must be read before it happens |
| An automatic background fetch | Voids a `--force-with-lease` ([sublimehq/sublime_merge#1846]) and puts a timer against a remote for a surface nobody may be looking at |
| Interactive rebase inside the graph | Needs an editor loop the graph cannot host; the terminal has one, and an agent resolves conflicts better than a dialog can |
| A standing `git/head` poll | Refused by ADR-0030 and 0073; the watch here is bounded to a pending action and a visible tab |
| Hiding rows after an action, or filtering the graph to the branch you acted on | ADR-0038: the lane layout is positional, and a filtered graph is a lie |
| Plain `--force`, `filter-branch`, `gc --prune=now` | No tier holds them, no undo covers them, and nobody asked |
| A third copy of the command composer | It is already duplicated twice (measured); the graph must not be the third |
| Disabling menu items git would refuse | mhutchie hides them; a disabled item that never explains itself is worse than an absent one — and PiCode can offer the move that *does* work |

## Alternatives considered

| Alternative | Why not |
|---|---|
| `POST …/git/{action}` running git in the service process | The environment gap above; also duplicates the interlock, the ref resolution and the credential story that the terminal doors already own |
| Keep the graph read-only and grow the Inspector's Git menu instead | The rail's actions are HEAD-scoped by construction; the six of them cannot express "rebase *this branch* onto *that commit*". The target is the graph's whole contribution |
| One "Git actions" button in the graph header, like the rail's | Same limitation, and it wastes the rows already on screen — the field's answer is a menu per target, unanimously |
| Ship tier C first (delete branch/worktree is what the owner named) | Tier C is where an undo matters, and the undo needs the recorded positions phase 2 introduces. Phases 1–2 also cover the create side the request implies |
| A confirmation *colour* or a "danger" section instead of tiers | A colour does not decide which doors are offered; the tier does, and it is checkable in a test |
| Server-side snapshot before every action (GitButler's oplog) | PiCode does not own the working tree; a snapshot it cannot restore atomically is a promise it cannot keep |

[sublimehq/sublime_merge#1846]: https://github.com/sublimehq/sublime_merge/issues/1846

## Amendment (2026-09-08): what the composer really does, and what a review found

§5 said the composer uses `--end-of-options` wherever a ref could be read as a
flag. It does not; it **refuses** any ref that could be — a leading dash,
`..`, `@{`, a traversal, a shell metacharacter — so nothing reaching argv needs
a terminator. The guard is the validation, and the sentence is corrected here
rather than the code made to match a sentence.

An adversarial review the same day found, and this amendment records:

- **Undo composed `reset --hard` for a commit**, which deletes the work that
  was just committed. The inverses are now honest per action: `--soft` for a
  commit, each reset by its own kind, and `--keep` — which refuses rather than
  lose a local change — for merge, rebase, pull, cherry-pick and revert.
  `--hard` is offered only to undo a `--hard`.
- A delivery that failed (409 busy, terminal moved) was announced as *Sent*.
  The form now keeps the failure and the graph never watches for it.
- A worktree command was relative to the terminal's folder, so one read from
  a sibling worktree nested a checkout inside it. It names
  `<root>/.worktrees/<name>` absolutely now, and a worktree folder is one
  segment.
- A second remote's branches were refused as "not a branch": the remote is
  read from the target's own prefix.
- The undo position is read at composition time, not from the graph's last
  load, and the pending watch compares against a token read just before
  delivery — not against a baseline an earlier pending action may have moved.

The typed confirmation is a browser gate and the catalog is a correctness
contract: the `type` / `run` routes accept what they always accepted. Nothing
here makes the doors safer against a hostile client than ADR-0078 left them;
what changed is that an honest client can no longer get the quoting or the
tier wrong.

## Amendment (2026-09-08, later): the ask door reaches pi terminals

The review above still listed the ask door as unexercised. Trying it on the
owner's own instance showed why: the owner runs pi as an Agent CLI terminal,
and the door reached only agents of the store. ADR-0089's amendment of the
same day opens it, narrowly, to pi terminals with a live receiver; the graph
lists them as occupants and asks them through `POST /api/terminals/{id}/ask`.
