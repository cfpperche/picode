# Process and repository

## Next

- Runbook step 6 (0.2.0, tag `v0.2.0`): watch the owner's window; a regression becomes a patch tag, never a rewritten one.

## Debts

- `.pi/compact.json` `atPercent 0.5` never fires for large-window models (peaks 379 K); capture tolerance is 128 px.
- `.git` is ~530 MB (UI bundles, MP4s); a history rewrite is the owner's call. Branch protection and CODEOWNERS need the owner.
- Tutorial video freshness audits are stale after source relocation.
- Three worktrees sat idle 22 h–4 d with no commit; `make worktree-status` reports them, and their fate is the owner's.
- On merging `feat/herdr-validation` / `feat/picode-video-pilot` (opened before ADR-0105): their prose moves to `docs/changelog.d/` fragments and the session note, not into the board.
- Hook edits cannot be exercised from the making worktree: every worktree runs the root checkout's hooks — a commit those hooks would refuse needs `git -c core.hooksPath=$PWD/.githooks commit`.
- `make ci` failed once (2026-09-12) after listing the Go packages ok, then passed on the identical tree — cause unknown; `var/ci-last.log` keeps the next run diagnosable (retries would hide it).
- A branch that edited `docs/handoff.md` before ADR-0123 meets one `modify/delete` conflict at merge: resolve by `git rm -f docs/handoff.md` — the hook refuses it staged as a modification, on purpose.
