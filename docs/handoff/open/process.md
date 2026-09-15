# Process and repository

## Next

- Handoff board is over budget on main (116 lines / 12670 bytes vs 120/12288 caps, 2026-09-15), so every branch's `make close` fails at `make handoff` until paid debts are pruned from the largest open topics.

## Debts

- `.pi/compact.json` `atPercent 0.5` never fires for large-window models (peaks 379 K); capture tolerance is 128 px.
- `.git` is ~530 MB (UI bundles, MP4s); a history rewrite is the owner's call. Branch protection and CODEOWNERS need the owner.
- Tutorial video freshness audits are stale after source relocation.
- On merging `feat/herdr-validation`/`feat/picode-video-pilot` (pre-ADR-0105): their prose goes to changelog fragments + the session note, not the board.

## Notes

- Hook edits cannot be exercised from a worktree (it runs the root's hooks); a refused commit needs `git -c core.hooksPath=$PWD/.githooks commit`.
- `make ci` failed once (2026-09-12) after Go packages ok, passed on the identical tree — cause unknown; `var/ci-last.log` keeps it diagnosable (retries hide it).
- A branch that edited `docs/handoff.md` pre-ADR-0123 hits one `modify/delete` conflict: resolve with `git rm -f docs/handoff.md` (the hook refuses it staged, on purpose).
- The Go shard runner flakes on `internal/server` under load (2026-09-15): `TestCreateTerminalInWorkspaceUsesItsFolder` reads the pane's cwd as the *test's* own directory, and the same run logs `TestTmuxServerWatchIntegration`: `tmux: server on /tmp/… is gone`. Suspect cross-test interference after the tmux isolation/reap work (ADR-0141). Passes in isolation (0.5 s) and on retry — `make ci-scoped`/`make close` failed twice and passed on the next run, so a red `internal/server` in that shard is not proof the diff broke something.
