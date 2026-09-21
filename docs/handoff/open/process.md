# Process and repository

## Next

- [x] The handoff board sat at its cap and failed `make close` until a paid debt was pruned — superseded by ADR-0145 (bounded view: 2 bullets per topic, 7-day notes, open debts only, over target warns).

## Debts
- [ ] **`TestStopIdleFencesConversationAndCommands` fails on the GitHub
  ubuntu runner and nobody can say why.** It reports "agent has an action in
  progress" at the one assertion where `StopIdle` must *succeed*
  (`internal/rpc/runtime_test.go`), after 3.57s — four times the 0.85s it
  takes locally. `StopIdle` gates on `commandMu.TryLock()`, so "idle" is an
  instantaneous condition, and the obvious theory is a background holder on
  a loaded runner. **That theory is not supported**: 36 rounds under three
  parallel loops are green here, no production code calls `send`, `ReplyUI`,
  `GetState` or `Interrupt` from a goroutine, and the test starts none of its
  own (all checked 2026-09-21, feat/ci-env-tests). It was deliberately left
  alone — a bounded retry would turn the gate green without anyone
  understanding it, which is the failure the other four fixes in that branch
  were about. Whoever reproduces it owns the fix.

- [ ] **Should `/pair` be guarded-and-exempt instead of unguarded?** Today
  `guarded()` covers `/api/`, `/ws/` and `/mcp/communication` only, so the
  pairing page and form never reach the Host or Origin checks. An unreachable
  `case p == "/pair"` in `exempt()` made it read as a deliberate pass; the
  dead line is gone (feat/audit-fixes, 2026-09-21) and the question is not.
  Guarding it would apply `HostAllowed` to the one route someone uses when
  they cannot reach PiCode yet, so it is the owner's call, not a tidy-up.

- [ ] **ADR-0154 (picode-mcp) and ADR-0156 (computer foreground guard) are
  still `proposed` in both the file and the index, and both shipped.** The
  2026-09-21 audit synced the eight ADRs whose own file already said
  accepted; these two need the owner to say the word, because flipping a
  status is recording a decision, not fixing an index. `make docs-check` now
  refuses the mismatch in the other direction (accepted file, proposed
  index), so this cannot drift further.

- [ ] **A conflicted merge plus `git add -A` committed conflict markers into
  `WebTab.jsx`, and main's web build broke for ~10 minutes (2026-09-18).** The
  landing loop hid the merge output, so a stopped-with-conflicts merge looked
  like "main moved again", and the next `git add -A` swept the markers into the
  feature commit. Recovery: `PICODE_ALLOW_MAIN_REWIND=1 git reset --hard` to the
  last green tip, work preserved on `feat/annot-pick`. Candidates: the landing
  loop must abort on a non-zero merge exit (never `> /dev/null`), `make close`
  should fail on a file containing conflict markers, and a ff must be refused
  when `ci-scoped` did not pass.

- [ ] **A dirty root checkout is silently allowed until a landing refuses.**
  On 2026-09-18 my own uncommitted one-line edit to a handoff topic sat in the
  root and blocked `git merge --ff-only` for five retries while I blamed other
  sessions' traffic — the refusal message was going to `> /dev/null`. The hook
  stops feature *commits* in the root; nothing warns about *dirty tracked
  files*. Candidate: `make land`/`make close` prints the root's dirty files
  before attempting the fast-forward, and a blocked ff always prints its
  reason.


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
