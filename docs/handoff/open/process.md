# Process and repository

## Next

- Runbook step 6 (0.2.0, tag `v0.2.0`): watch the owner's window; a regression becomes a patch tag, never a rewritten one.

## Debts

- `.pi/compact.json` `atPercent 0.5` never fires for large-window models (peaks 379 K); capture tolerance is 128 px.
- `.git` is ~530 MB (UI bundles, MP4s); a history rewrite is the owner's call. Branch protection and CODEOWNERS need the owner.
- Tutorial video freshness audits are stale after source relocation.
- Three worktrees sat idle up to 4 d with no commit; `make worktree-status` reports them, their fate is the owner's.
- Merging `feat/herdr-validation` / `feat/picode-video-pilot` (pre-ADR-0105): their prose moves to `docs/changelog.d/` fragments and the session note, not into the board.
- Hook edits cannot be exercised from the making worktree: every worktree runs the root checkout's hooks; a commit the branch's own hook allows needs `git -c core.hooksPath=$PWD/.githooks commit`.
- A full-matrix `make ci` failed once on 2026-09-12 and passed on the identical tree next run — cause unknown; the run is kept in `var/ci-last.log` (retries would only hide it).
- A branch that edited `docs/handoff.md` before ADR-0123 conflicts once (`modify/delete`) at merge: resolve by `git rm -f docs/handoff.md` — the hook refuses it staged as a modification, on purpose.
