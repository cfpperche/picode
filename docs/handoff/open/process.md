# Process and repository

## Next

## Debts

- `.pi/compact.json` `atPercent 0.5` never fires for large-window models (peaks 379 K); capture tolerance is 128 px.
- `.git` is ~530 MB (UI bundles, MP4s); a history rewrite is the owner's call. Branch protection and CODEOWNERS need the owner.
- Tutorial video freshness audits are stale after source relocation.
- On merging `feat/herdr-validation` / `feat/picode-video-pilot` (opened before ADR-0105): their prose moves to `docs/changelog.d/` fragments and the session note, not into the board.
- Hook edits cannot be exercised from a worktree (it runs the root's hooks); a refused commit needs `git -c core.hooksPath=$PWD/.githooks commit`.
- `make ci` failed once (2026-09-12) after listing the Go packages ok, then passed on the identical tree — cause unknown; `var/ci-last.log` keeps the next run diagnosable (retries would hide it).
- A branch that edited `docs/handoff.md` before ADR-0123 meets one `modify/delete` conflict at merge: resolve by `git rm -f docs/handoff.md`; the hook refuses it staged as modified, on purpose.
