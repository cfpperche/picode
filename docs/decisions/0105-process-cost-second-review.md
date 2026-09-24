# ADR-0105: The rite runs in a fresh context, the living docs stop conflicting, deploy is the owner's call

- **Status**: accepted
- **Date**: 2026-09-09
- **Boundary**: process — how a branch closes, how `main` ships, how the changelog and handoff are written, how ADRs are minted
- **Amends**: ADR-0086 (batched deploys and the deploy timer, captures in `make close`, the line-only handoff cap, CHANGELOG edits per branch)

## Context

A second adversarial review of the loop, three days and 394 commits after
ADR-0086, measured the same things again — this time from the session
transcripts themselves (17 Claude Code sessions and 37 pi sessions on this
repository, 2026-09-06 → 09-09), the reflog of `main`, and the deploy log.

| Metric | Value |
|---|---|
| Context tokens sent by the 17 Claude Code sessions | 1.47 billion over 3,997 calls |
| Median peak context per session | 348 K; 14 of 17 sessions above 150 K; one at 965 K |
| Share of all context tokens spent in calls above 400 K | 62% |
| Branches touched by one session | up to 16 |
| Closing-rite windows (`make close` → merge/deploy) | 30; median 6 calls and **2.7 M tokens** each; 11 needed two to four `make close` |
| Catch-up merges since 09-06 | 75; **45 had hand-resolved hunks** in CHANGELOG (31), handoff (26) or architecture (5) — 94% of all conflict resolutions |
| Commits since 09-06 | 394, **52% `docs:`**; 85 "refresh public captures", 10 of 33 compared pairs spurious (PNGs differing by a few bytes) |
| Direct commits on `main` | 65; about half "record the merge/deploy" prose that ADR-0086 had banned |
| Deploys | 50 from 20 agent terminals, median gap 37 min; the 12/18/23 timer found "already deployed" every time; 10 agent deploys used `--force` |
| `docs/handoff.md` | 99 lines but 12.4 KB — lines up to 581 bytes; the cap was gamed |
| `docs/architecture.md` | 128 KB; one section of 66 KB; the reading table pointed at "the section", which did not exist as a unit |
| ADRs since 09-06 | 17 (six a day); four were UI or route refinements |
| Gates wall-clock in the Claude Code sessions | 9.7 h in three days, ~10 min per branch, against a median 17 min from first commit to merge |
| `internal/server` tests | 365 serial tests, 75–90 s; **22 s** split across four processes, all green |
| Go test cache | keyed by `PICODE_TERM_ID` (read by `internal/install`), so every PiCode terminal ran cold; proven with one variable changed |
| GitHub CI | red since 09-06 (a Windows toolchain download, then a whitespace check that failed the whole run); `origin` 20 commits behind |

Three mechanisms dominated: sessions that keep working through many
branches and run the mechanical rite at 400–900 K of context; two prose
files that every branch edits on the same lines; and a deploy policy that
nobody followed, so its timer, its text and its `--force` hatch were pure
overhead. The owner reviewed the findings on 2026-09-09 and approved the
changes below, with two explicit refusals recorded under *Alternatives*.

## Decision

1. **One branch, one session.** The session that did the work ends at the
   fast-forward; the next task starts in a new terminal. The closing docs
   are written from `make close-summary` in a subagent or a fresh session,
   never at the peak context of the working session. Screenshots for
   visual-review are read in a subagent. No harness configuration changes
   (owner's decision): this is a rule in AGENTS.md, CONTRIBUTING.md and the
   skills, enforced by review.
2. **Deploy is the owner's call.** The deploy timer and `make deploy-batch`
   are removed. The owner runs `make deploy` from the root whenever they want
   `main` live; a branch session never deploys on its own. `picode deploy`
   keeps ADR-0086's mid-turn guard; `--force` stays the owner's one-off.
   Public captures refresh inside `make deploy` when the fingerprint says so,
   once per deploy instead of once per branch, and `scripts/docs-shots.mjs`
   keeps the committed image when a recapture differs by at most 0.05% of
   its pixels.
3. **The changelog is assembled.** A branch writes one fragment,
   `docs/changelog.d/<branch-slug>.md`, with Keep a Changelog sections;
   `make changelog` folds every fragment into `[Unreleased]` on `main`
   before a release. The pre-commit hook refuses a `CHANGELOG.md` edit that
   is not that assembly.
4. **The handoff holds no shipped work.** `docs/handoff.md` keeps in flight,
   next up and debts only; shipped work is `git log`, the ADR index and the
   changelog. The hook caps it at 100 lines **and 8 KB**, and refuses
   `docs/handoff.md`, `docs/handoff/` or `docs/changelog.d/` committed
   directly on `main` in the root checkout — they travel with the branch.
5. **Whitespace errors stop at the commit.** `git diff --cached --check`
   runs in the pre-commit hook; the GitHub step that failed whole runs is
   gone. The Go matrix runs on Ubuntu for every push and adds macOS and
   Windows only for a tag or a manual run; the tmux build is cached.
   **Amended 2026-09-23:** macOS joins the per-push matrix when the diff
   touches a path where it has actually broken before — sockets and process
   inspection in `internal/tmux` and `internal/server`, path resolution in
   `internal/clicreds` and `internal/clipkgs`, the workflow and
   `scripts/ci-scope.mjs` itself (`macosRelevant`). The legs were cut down
   because they were red for infrastructure reasons and unread; those defects
   are fixed, and the last one (two macOS test bugs) was found only by a
   manual run, which is the late discovery the paragraph above calls the cost.
   Windows stays on tags and manual runs: it compiles rather than executes the
   daemon (ADR-0020), so a per-push leg would buy little.

   **Amended 2026-09-24:** the two packages that take ~26 minutes under `-race`
   on a two-core runner (`internal/server`, `internal/store`) run in a job of
   their own (`go-heavy`), on the same platform list and with the same Windows
   skip; the rest of the suite keeps a ceiling and a verdict of its own. A
   package this slow had already killed a run with no failing test to read
   (`FAIL internal/server 1800.070s`, 2026-09-22) while sharing the ceiling
   with ~300 others.
6. **Gates.** `scripts/go-test.sh` runs `internal/server` across four
   processes (package globals are per process, so the probe swapping that
   rules out `t.Parallel` is untouched) and unsets `PICODE_TERM_ID` so every
   terminal shares the test cache. `make ci` runs its gates four at a time.
   `make web` is a stamp: no rebuild when nothing under `web/` changed.
   `make close` reuses a green `ci-scoped` when the tree hash has not
   changed since.
7. **ADRs are minted, and need a boundary.** `make adr NAME=x` allocates the
   number across every worktree and appends the index row. The template
   carries a mandatory Boundary line; a change with no protocol,
   persistence, security-model or process boundary is not an ADR.
8. **Architecture is one file per subsystem.** `docs/architecture.md` is an
   index; the content lives in `docs/architecture/<subsystem>.md` so the
   reading table in AGENTS.md names a file that exists.

## Consequences

Easier: a branch's closing rite costs a fresh context instead of a peak one;
two branches never edit the same changelog or handoff lines, so catch-up
merges stop asking for hand resolution; `main` restarts only when the owner
asks; the heaviest gate takes a quarter of the time and its cache is shared
across terminals; a CSS-only merge on `main` no longer rebuilds the UI
three times; an ADR costs one command, not a 19 KB read and a number race.

Harder: the public site's images and the production instance lag `main`
until the owner deploys; a release needs `make changelog` first; the split
architecture loses in-page anchors that pointed into the old file (none
existed); a branch that was mid-flight on 2026-09-09 with a `CHANGELOG.md`
edit merges as it is, but its next changelog line must be a fragment; the
GitHub macOS/Windows legs run rarely, so an OS-specific regression surfaces
at the next tag, not the next push.

If wrong: `GO_TEST_SHARDS=1` restores the serial suite; `make ci-gates`
runs the gates serially; deleting `var/web.built` forces a UI rebuild;
`PIXEL_TOLERANCE=0` in `docs-shots.mjs` restores byte-exact captures;
`PICODE_ALLOW_SWITCH=1` bypasses every hook rule for a deliberate one-off.
Nothing here touches the product's data plane.

## Decision table — `.githooks/pre-commit`

| Tree | Staged | Action |
|---|---|---|
| any | `CHANGELOG.md` or `docs/handoff.md` without its first-line header | refuse (clobbered by a parallel session) |
| any | `docs/handoff.md` over 100 lines or 8 KB | refuse (cap) |
| any, merge/rebase/cherry-pick in progress | anything | allow (the merge commits on its own terms) |
| any | a diff with trailing blanks or space-before-tab | refuse (whitespace) |
| any | `CHANGELOG.md` without a deleted `docs/changelog.d/*.md` alongside | refuse (write a fragment) |
| root on `main` | `docs/handoff.md`, `docs/handoff/*` or a `docs/changelog.d/*` addition | refuse (travels with the branch) |
| root on `main` | anything else | allow |
| root off `main` | anything | refuse (feature work belongs to a worktree) |
| linked worktree | anything else | allow |

Coverage: `scripts/hooks-selftest.sh` (29 rows, run by `make hooks-check`
and `make ci`), `scripts/changelog-assemble.test.mjs`.

## Alternatives considered

- **Deploy timer every 15 minutes** (automatic, coalescing, no agent ever
  deploys): rejected by the owner — no scheduled deploy at all; they ask
  for one when they want it.
- **`autoCompactWindow` / `CLAUDE_CODE_AUTO_COMPACT_WINDOW` in the repo's
  Claude Code settings, and `atTokens` in `.pi/compact.json`**: rejected by
  the owner — no custom harness configuration. The session-length rule is
  therefore behavioural; the measurement (62% of tokens above 400 K, pi
  peaks at 379 K with compaction firing ten times in 37 sessions) stays here
  as the trigger for a re-measure.
- **`t.Parallel()` in `internal/server`**: still rejected (ADR-0086); the
  process shard gets the same speed without touching the tests.
- **Keep captures in `make close` with the pixel tolerance only**: rejected —
  the tolerance removes the spurious commits, not the minute of fixture and
  browser time paid by every UI branch.
- **A separate `docs/changelog.d` per Keep a Changelog section**: rejected —
  one file per branch is what the branch already owns.
- **Rewriting git history to drop the committed UI bundles**: still the
  owner's call, outside this ADR.

## Amendment (2026-09-24): the owner decides when a session ends

§1 said the next task starts in a new terminal. The owner overrode it in
practice — on 2026-09-23/24 one session carried the AGENTS.md study, the
Instructions page, ADR-0204 and five more branches at the owner's request —
and a rule nobody keeps teaches agents to discount the others. The session
may now continue past the fast-forward when the owner asks; the owner, not
the agent, says when to switch. What stays is the part that bought the
savings: one branch at a time, the closing docs written from `make
close-summary` in a subagent, and screenshots read in a subagent, so the
peak context never carries the rite or the images. The measurement in
Context remains the trigger for a re-measure if long sessions grow costly
again.
