# Process and repository

## Next

- [x] The handoff board sat at its cap and failed `make close` until a paid debt was pruned — superseded by ADR-0145 (bounded view: 2 bullets per topic, 7-day notes, open debts only, over target warns).

## Debts

- [ ] **`internal/server` in one `-race` process on a two-core runner has
  outgrown its timeout.** Measured 2026-09-23: 1691 s on the 0.5.0-era suite
  (94% of the 30m ceiling), then `FAIL internal/server 1800.070s` — a red gate
  with no failing test to read — once ADR-0184's launch tests, which spawn real
  terminals, landed. The ceiling is 50m now; that buys time, not speed, and
  every push to `main` pays ~35 minutes for that job. Candidates, cheapest
  first: give the heavy packages (`internal/server`, `internal/store`) a CI job
  of their own, so their ceiling is theirs and the rest of the job finishes
  early; revisit `GO_TEST_SHARDS=1` — the comment in `.github/workflows/ci.yml`
  turned sharding off because four shards each started their own tmux server,
  and the tmux tests now keep to their own socket and server, so that reason
  may be addressable; and trim the suite, which is 365 serial tests by design
  (they swap package-level probes, ADR-0086).
  **Re-measured 2026-09-23** (`feat/ci-sharding`, branch not landed): a
  dispatched run with the script's four shards came back red, but not for the
  reason the decision was made for — macOS failed on two test defects of its
  own (below) and ubuntu on `TestPreviewEventsStream`, seen once under shard
  load and unattributed. So the verdict on sharding is *open*, not negative,
  and the honest next step is the first candidate here — the heavy packages in
  a CI job of their own — because it removes the ceiling question without
  betting the gate on how four processes share a two-core runner.
  **Landed 2026-09-24** (`feat/ci-heavy-job`, owner approved on request): the
  heavy pair runs in a `go-heavy` job on the same platform list and with the
  same Windows skip, the rest of the suite keeps a `-timeout 25m` ceiling of
  its own (`GO_TEST_EXCLUDE_HEAVY=1` in `scripts/go-test.sh`), and the two jobs
  run in parallel. The ceiling question is answered by construction. Whether
  the split buys wall clock is not assumed: two cores are still two cores, and
  the 275-385 s shard timings above came from a 16-core machine, not a runner.

- [x] **The macOS Go job is red on tests nobody runs.** Paid 2026-09-23, in two
  steps: the two defects are fixed (`feat/macos-tests`) and the leg now runs
  where it matters — a push whose diff touches a path macOS has actually broken
  on before joins it to the matrix (`macosRelevant` in `scripts/ci-scope.mjs`:
  `internal/tmux`, `internal/server`, `internal/clicreds`, `internal/clipkgs`,
  the workflow and that script). ADR-0105's file and its index row carry the
  amendment, because the reason the leg was cut — red for infrastructure
  reasons, unread for days — was a fact that had stopped being true.

- [x] **`TestStopIdleFencesConversationAndCommands` fails on the GitHub
  runners and nobody can say why.** Paid 2026-09-22 (`feat/ci-vendor-tests`) —
  reproduced and explained. The capture bridge resolves its session file in a
  goroutine at spawn (`refreshCaptureSessionFile`, ADR-0082) and **holds
  `commandMu` for the whole `get_state`** while it waits for the writer's reply,
  so the test's wait ("probe the lock until it is free") proved only that the
  lock was momentarily free — the query could start right after it, and the
  `StopIdle` that must succeed then landed inside it. The 2026-09-21 audit's
  premise was wrong on one file: `resolveCaptureSessionFile` calls `GetState`
  **from that goroutine**, so "no production code calls `GetState` from a
  goroutine" was never true. Evidence: `go test ./internal/rpc -run
  TestStopIdleFencesConversationAndCommands -race -count=10` fails on the old
  tree (30.5s) and passes on the fixed one (30.1s); papering only one of the two
  gates moved the message from "queued work in progress" to "an action in
  progress", which is what pointed at the lock rather than at queued work. Fix:
  the bridge marks `resolved` when its first `get_state` returns (success or
  not) and the test waits for that — no retry, the single assertion stays.

- [x] **Nothing stops the next server test from depending on a CLI installed on
  the machine.** Paid 2026-09-23 (`feat/hermetic-gate`): `ci-scoped`'s Go stage
  now runs the scoped packages with `PATH=<toolchain>:/usr/bin:/bin` — no agent
  CLI exists there, so a test that reaches a vendor-locating path without
  stubbing fails at the gate instead of on a runner. Three did on 2026-09-22
  (`opencode`, `omp`, `pi`; the third landed an hour after the first two were
  fixed) and main went red for hours over tests this PATH catches in one run.
  The whole suite was measured under it before adopting it: one failure, which
  was not a CLI dependency at all — see the terminal-creation race fixed in the
  same branch.

- [x] **Should `/pair` be guarded-and-exempt instead of unguarded?** Today
  `guarded()` covers `/api/`, `/ws/` and `/mcp/communication` only, so the
  pairing page and form never reach the Host or Origin checks. An unreachable
  `case p == "/pair"` in `exempt()` made it read as a deliberate pass; the
  dead line is gone (feat/audit-fixes, 2026-09-21) and the question is not.
  Guarding it would apply `HostAllowed` to the one route someone uses when
  they cannot reach PiCode yet, so it is the owner's call, not a tidy-up.
  **Decided 2026-09-22:** the cross-site check applies to a `/pair`
  submission; the `Host` check deliberately does not. That closes the
  tab-in-another-window case with no way to lock anyone out of pairing.

- [x] **ADR-0154 (picode-mcp) and ADR-0156 (computer foreground guard) are
  still `proposed` in both the file and the index, and both shipped.**
  Accepted by the owner 2026-09-22; file and index both say so. The
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
- [x] **`make ci-scoped` never runs `docs-check` for a change under `docs/`.**
  `scripts/ci-scope.mjs` classifies every `docs/` path as `metadata` (fmt,
  vet, hooks), but `docs-check` resolves every relative Markdown link inside
  `docs/`. On 2026-09-23 `feat/agents-md-study` closed green and then failed
  `make ci` on `main` at 65ba8571: the link regex read Antigravity's include
  syntax inside inline code as a broken link (fixed by `feat/agents-md-link`).
  Candidate: the `metadata` scope runs the link pass of `docs-check` alone.
  — paid by `feat/ci-docs-links`: `scripts/docs-living.mjs` (ADR index,
  architecture index, relative links; code spans are not links) runs in
  `ci-scoped` for docs/-only changes and on every `make close`.


- `.pi/compact.json` `atPercent 0.5` never fires for large-window models (peaks 379 K); capture tolerance is 128 px.
- `.git` is ~530 MB (UI bundles, MP4s); a history rewrite is the owner's call. Branch protection and CODEOWNERS need the owner.
- Tutorial video freshness audits are stale after source relocation.
- On merging `feat/herdr-validation`/`feat/picode-video-pilot` (pre-ADR-0105): their prose goes to changelog fragments + the session note, not the board.

## Notes

- **A `CI gate` failure whose output says `GO: cancelled` is not a failure.**
  A push to `main` cancels the run in progress (observed 2026-09-22 22:10: the
  gate read `failure` only because `GO` read `cancelled`, and the very next
  push had already started a fresh run). Read the *newest* run for the tree
  before diagnosing. And `gh run watch … | tail` reports `tail`'s status, not
  the run's — redirect and read `$?`, or a green-looking `exit=0` will cover
  two real failures.
- **`git stash` is shared across every session in a checkout.** A peek at the
  list on 2026-09-23 found `stash@{0}` mine and `stash@{1}` another branch's
  ("emitter checkpoint before managed-stop retest", `feat/browser-capture-emitter`)
  — the same stack, one `stash clear` or a wrong-index `pop` away from deleting
  someone else's work. Read the list before touching it, pop by index, and know
  that a worktree's stash lives in the primary checkout's git dir.
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
