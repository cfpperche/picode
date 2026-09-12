# Process and repository

## Next

- Runbook step 6 (0.2.0, tag `v0.2.0`): watch the owner's window; a regression becomes a patch tag, never a rewritten one.

## Debts

- `.pi/compact.json` `atPercent 0.5` never fires for large-window models (peaks 379 K); capture tolerance is 128 px.
- `.git` is ~530 MB (UI bundles, MP4s); a history rewrite is the owner's call. Branch protection and CODEOWNERS need the owner.
- Tutorial video freshness audits are stale after source relocation.
- Three worktrees sat idle 22 h–4 d with no commit in the 2026-09-12 measurement; `make worktree-status` reports them, and their fate is the owner's.
- On merging `feat/herdr-validation` / `feat/picode-video-pilot` (opened before ADR-0105): their prose moves to `docs/changelog.d/` fragments and the session note, not into the board.
- Hook edits cannot be exercised from the worktree that makes them: `core.hooksPath` is absolute (`<root>/.githooks`), so every worktree runs the *root* checkout's hooks — `make hooks-check` on a throwaway repo is the local proof, and a commit the branch's own hook allows (deleting `docs/handoff.md`) needs `git -c core.hooksPath=$PWD/.githooks commit`.
- A full-matrix `make ci` failed once on 2026-09-12 (Go packages listed ok, then FAIL) and passed on the identical tree on the next run — cause still unknown; the run is now kept in `var/ci-last.log`, so the next one is diagnosable (retries would only hide it).
- A branch that edited `docs/handoff.md` before ADR-0123 conflicts on it once (`modify/delete`) when it merges the generated board in: resolve by `git rm -f docs/handoff.md` — the hook refuses the commit while the file is staged as a modification, on purpose.
