# 2026-09-08 — feat/git-graph-deliver: the graph writes (ADR-0096 phases 2-4)

34 actions bound to the row or pill you point at, through ADR-0078's three
doors. The service process still never runs git.

**One composer, in Go** (`internal/gitcmd`). `POST …/git/compose` returns the
exact command, its tier, the verb and the agent prompt; the browser builds no
shell fragment, and the preview is the string the terminal receives. Refs are
*validated*, never quoted — a dash, `..`, `@{`, a traversal or a metacharacter
is refused. Tiers come from `GET /api/git/actions`: with no catalog the graph
offers no write action at all (fails closed). The branch a push publishes is
read from the checkout, not from the caller.

**Gates by who presses Enter.** Prepare needs none — the human reads the
command in their own prompt. Tier C through the run door asks for a typed
phrase, always something already on the dialog.

Phase 4: **worktree + agent in one gesture** (proven live — `.worktrees/qa-tree`
on disk and agent `qa-tree` living in it), and **undo where an honest inverse
exists** — it composes `git reset --hard <recorded>` and sends it through the
same door with the same gate. Push, discard and clean get no undo, because
there is none to give.

Four defects found in the browser, not in review, each fixed with a test:

1. `.um-popover` capped its height against the viewport, so a menu opened low
   on a list ran off the bottom without scrolling (542px at y=183 in a 649px
   window, `ok:false`). It follows Radix's available-height now — every menu
   in the app inherits the fix.
2. Submenu leaves never dispatched under the harness, leaving grouped rows
   unprovable. They are labelled sections now: one click per row, no second
   layer. (The terminal menu's own `text` submenu shares the shape and was
   never exercised — worth a hand check.)
3. **A prepared command swallowed the next one.** An unsubmitted command stayed
   on the prompt and the next arrived glued to it (`fatal: only one reference
   expected`). `tmux.ClearLine` runs before every type, for the Inspector rail
   too. Regression test in `internal/tmux`.
4. `git/head` had no workspace route (missing since ADR-0038), so a graph
   opened through a workspace watched forever; and the token ignored the
   worktree list, so `git worktree add` was reported "still running" after it
   had finished. Both fixed, both tested.

visual-review: PASS (gg2-menu-sections, gg2-dialog-create-branch,
gg3-tierc-typed-confirm, gg4-worktree-and-agent, gg4-undo-offer; overlayAudit
ok on every overlay, no row wraps; card 5/5).

Debts: the **ask** door is unexercised live (it needs a running agent —
`askableAgents` filters to running ones); mobile has no sheet for the new
vocabulary yet, though the module is shared; the Inspector rail still composes
its own six commands client-side rather than through `git/compose`.
