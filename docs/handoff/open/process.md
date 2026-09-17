# Process and repository

## Next

- [x] The handoff board sat at its cap and failed `make close` until a paid debt was pruned — superseded by ADR-0145 (bounded view: 2 bullets per topic, 7-day notes, open debts only, over target warns).

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

## Stalled worktrees discarded (2026-09-15, owner's instruction)

Four trees idle 1.3–7.2 days were the handoff board's overflow — six in-flight
rows pushed it past the cap and **every** branch's `make close` died on it.
Discarded with their branches: `herdr-validation` (2 commits), `picode-video-pilot`
(5), `desktop-entry-fix` (3, including `web: one boot path for both entries —
/desktop/ was shipping without styles`), `tmux-orphans` (no commits, two dirty
files under `internal/tmux/`). Ten commits are gone; the reflog is the only
place they survive. Board back to 113 lines / 11944 bytes.

`make ci` now renders the board too, so the next overflow fails at CI time
instead of in the closing rite of whoever happens to run it next.
