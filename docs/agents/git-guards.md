# The git guards

> Reference for [AGENTS.md](../../AGENTS.md) non-negotiable 5 (isolated
> worktrees). AGENTS.md keeps the rules; this file keeps how git enforces them
> and the incidents behind them.

This is **enforced by git, not by trust**: `.githooks/reference-transaction`
aborts any `git switch`/`git checkout` that would move the root checkout
off `main` (switching back to `main` is always allowed), refuses any
rewind of `main` behind its current tip — including a fast-forward from
a stale ref onto a divergent tip, which silently erased merged work once
(rolling back is a deliberate `PICODE_ALLOW_MAIN_REWIND=1` one-off) — and
`.githooks/pre-commit` refuses feature commits made there, clobbered
living docs, a committed handoff board, whitespace errors, conflict
markers in the staged diff, direct
`CHANGELOG.md` edits, changelog fragments and *new* session notes on
`main` (ADR-0105); amending a handoff note that already landed, or adding
`docs/handoff/open/<topic>.md`, is allowed there — one commit, no branch
(ADR-0149). `make hooks` (implied by
`make dev` and `make ci`) points git at them; `make hooks-check` proves
the whole policy on a throwaway repo. A clone that never ran make has no
guard. Deliberate one-off: `PICODE_ALLOW_SWITCH=1 git switch <branch>`.
**Never `git add -A` or `git add .`** — stage explicit paths. A
conflicted merge resolved by wholesale staging shipped conflict markers
and broke main's web build twice on 2026-09-18 (one of them on main
itself); the pre-commit now refuses added markers, and wholesale staging
is how unrelated files ride into a commit unnoticed.
**Never run `git clean -fdx` (or `-fdX`) in the primary checkout:**
`.worktrees/` is git-ignored, so clean deletes every agent's working
tree in one stroke.
