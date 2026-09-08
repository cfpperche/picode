# 2026-09-08 — feat/git-graph-review: adversarial review of the graph's write phases

The owner asked for a hostile review of ADR-0096's four phases; this branch
fixes everything it found. The findings are recorded as an amendment in the
ADR itself. In order of damage:

1. **Undo of a commit composed `reset --hard`** — it would delete the work
   just committed. Inverses are per action now: `--soft` for a commit, each
   reset by its own kind, `--keep` (new `reset-keep`, refuses rather than lose
   a change) for merge/rebase/pull/cherry-pick/revert. `--hard` only undoes a
   `--hard`. Test: no undo of those ever composes a hard reset.
2. **A failed delivery read as "Sent".** `typeIntoTerminal`/`askAgentGit` take
   `{quiet}`: the form keeps the failure, the graph never watches for it.
3. **Focus escaped a fieldless dialog** (merge, fetch…). Only steered when
   there is a field; verified live — focus lands on the door select.
4. **A worktree created from a worktree nested inside it.** The composer names
   `<root>/.worktrees/<name>` absolutely (quoted; root from the common dir),
   and a worktree folder is one segment (`ErrSlug`). Verified live from inside
   a worktree: `git worktree add '/home/goat/picode/.worktrees/x' main`.
5. **A second remote's branches were "not a branch".** The remote comes from
   the target's prefix, longest known name wins (`ErrRemoteBranch` otherwise).
6. **The ADR claimed `--end-of-options`**; the code refuses instead. Amended.

Fragilities closed: the pending watch compares against a `git/head` token read
*just before* delivery (not the last load's), does not spend its budget while
the tab is hidden, and reloads the graph when it gives up; the undo position
comes from `compose` (fresh `head`), and the recorded ref is matched by kind
as well as name; `checkout-remote` opens with the local name filled; a
detached checkout's dirty row offers no `commit-push`; the catalog is
re-requested on every menu until it arrives, and the menu says when it has
failed; one report per failure (no toast + inline); ask-door copy no longer
mentions a terminal; the `ClearLine` test waits for a prompt and fails loud;
its comment names what was verified (bash, zsh, fish, tty canonical) and what
was not (nu, pwsh), and the CHANGELOG states the cost — an unsubmitted line is
discarded, where before it was corrupted.

Correction to the previous note: the terminal menu's `Text size` submenu
**works** (14px → 15px through its leaf, verified live). The phase-2 claim
that submenu leaves "never dispatched" was my harness timing, not Radix; the
graph's flat sections stay because they are the better design, not because
submenus are broken.

Not browser-proven, reasoned only: the "Sent over a failure" path needs tmux
to be unavailable or terminal creation to fail, neither forceable on the
scratch. The ask door is still unexercised live.
