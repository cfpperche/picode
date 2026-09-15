# Process and repository

## Next

- The handoff board sits at its 12288-byte cap: `make close` fails at `make handoff` until a paid debt is pruned (2026-09-15).

## Debts

- `.pi/compact.json` `atPercent 0.5` never fires for large-window models (peaks 379 K); capture tolerance is 128 px.
- `.git` is ~530 MB (UI bundles, MP4s); a history rewrite is the owner's call. Branch protection and CODEOWNERS need the owner.
- Tutorial video freshness audits are stale after source relocation.
- On merging `feat/herdr-validation`/`feat/picode-video-pilot` (pre-ADR-0105): their prose goes to changelog fragments + the session note, not the board.

## Notes

- Hook edits cannot be exercised from a worktree (it runs the root's hooks); a refused commit needs `git -c core.hooksPath=$PWD/.githooks commit`.
- `make ci` failed once (2026-09-12) after Go packages ok, passed on the identical tree — cause unknown; `var/ci-last.log` keeps it diagnosable (retries hide it).
- A branch that edited `docs/handoff.md` pre-ADR-0123 hits one `modify/delete` conflict: resolve with `git rm -f docs/handoff.md` (the hook refuses it staged, on purpose).
- **FIXED 2026-09-15** (`feat/tmuxcwd`): `TestCreateTerminalInWorkspaceUsesItsFolder` was reading the *daemon's* cwd. Not test interference: the live `#{pane_current_path}` read taken in the same instant as `new-session` races the pane's own process, and tmux answers with the *server's* directory (the daemon's cwd) until the pane's command has spawned — 12 of 30 creations in a loop, and the empty server restarting between creations is what opens the window. The creation response is now the folder the session was created in (`freshTermView`), the next poll reads live again, and `TestCreatedTerminalAnswersWithItsOwnFolder` fails without the fix. Residual hazard, not observed to fail: `TestTmuxServerWatchIntegration` sets a per-test `TMUX_TMPDIR` on a process-wide variable and then dismantles that server, so any test or background goroutine reaching tmux in that window lands on it.
