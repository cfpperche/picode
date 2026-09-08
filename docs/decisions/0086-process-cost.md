# ADR-0086: The rite around a change costs less than the change

- **Status**: accepted
- **Date**: 2026-09-06
- **Amends**: ADR-0018 (deploy), ADR-0084/0085 (restart forensics), the
  docs-harness parity principle (`docs/benchmarks/2026-09-03-docs-harness.md`)

## Context

An adversarial review of the planning → development → testing → review →
deploy loop (2026-09-06, 15 days and 1159 commits into the repository)
measured where tokens, provider quota and wall-clock went:

| Metric | Value |
|---|---|
| Commits touching `docs/handoff.md` | 678 of 1159 (58%); 143 touched nothing else |
| Merges with hand-resolved conflicts in handoff / CHANGELOG / architecture | 103 / 83 / 34 of 181 |
| Branches with exactly one commit | 98 of 137 (72%); median 5 min from first commit to merge |
| Docs commits after each merge | ~3.8 per merge; median 24 min, p75 118 min |
| Service restarts per day | 26–63 (journal); every restart ended the managed CLI/agent panes until ADR-0085 |
| `make ci` on a worktree | ~4 min, 84% in `go test ./...`, run for CSS-only diffs |
| GitHub CI on main | 142 failures in 157 runs — two macOS-only test assumptions, never fixed |
| `docs/handoff.md` | 485 lines against a stated ~150 cap; five "previous deployment" paragraphs |
| Mandatory reading before the first edit | AGENTS + handoff + skills + benchmarks ≈ 17k tokens; architecture.md 22k more; `docs/benchmarks/` 37k more |
| Stale worktrees | 8 of 10 belonged to merged branches, 1.4 GB each; each new tree paid `npm ci` twice |

Three mechanisms dominated: a deploy per merged branch on the instance the
other agents work in; a closing rite of twenty-odd mechanical turns run at
peak context, each re-sending the whole conversation; and one prose file
that every session reads, rewrites and hand-merges. The owner approved
eight changes on 2026-09-06.

## Decision

1. **Deploy leaves the per-branch rite.** `picode deploy` asks the running
   daemon `GET /api/deploy/readiness` (loopback callers need no session)
   and refuses, exit 2, while any agent or terminal is mid-turn; `--force`
   / `PICODE_DEPLOY_FORCE=1` is a deliberate one-off. `main` ships in
   batches: `make deploy-batch`, run by the owner or by the
   `picode-deploy.timer` at 12:00, 18:00 and 23:00, which also recaptures
   stale public images first. A merged branch is done when `main` can
   fast-forward, not when production restarted.
2. **The mechanical close is a script.** `make close` runs the scoped
   gates, regenerates the artifacts the diff invalidated (OpenAPI,
   llms.txt, captures when `web/` changed), checks that `main` can
   fast-forward and prints `make close-summary`. The closing docs are
   written from that summary — by a fresh session, or after compaction —
   never from a context that has read the repository.
3. **The handoff is capped and split.** `docs/handoff.md` is at most 100
   lines (pre-commit refuses more) and holds only current state, in
   flight, next up and debts. Each session leaves one file in
   `docs/handoff/<date>-<branch>.md` (≤ 25 lines); deployment history is
   `var/deploy-log.jsonl` and `git log`. Two sessions never edit the same
   lines.
4. **Capture parity is advisory.** `make docs-check` warns on a stale
   fingerprint and fails only on a hand-edited image or a stale generated
   file; `--strict` keeps the old behaviour for `make close` and the
   deploy batch, which recapture when it says so. The public site may lag
   the UI by hours.
5. **Gates match the diff.** `make ci-scoped` classifies the diff against
   `main` (`scripts/ci-scope.mjs --local`): Go changes test the changed
   packages and their importers; `web/` runs test-js and the embedded
   build; `docs-site/` (née `www/`) runs the site and Vale; metadata runs the fast gates.
   `make ci` remains the whole matrix, run once for the merge on `main`.
6. **GitHub CI must be green or smaller.** The macOS failures were test
   assumptions (unresolved `/var` symlinks, a 503 asked before request
   validation) and are fixed; a red job that nobody reads is deleted, not
   tolerated.
7. **Reading follows the change; ADRs follow boundaries.** AGENTS.md
   carries a table of what to read for a CSS fix, a component, a handler
   or a protocol change. An ADR is written for protocol, persistence,
   security-model and process boundaries — never for a UI refinement.
   The default pi role thinks at `medium`; `max` is asked for.
8. **Hygiene is a command.** `make worktree NAME=x` creates a tree with
   hardlinked `node_modules` (one second, not `npm ci`); `make
   worktree-gc` removes merged, clean, idle trees. Visual-review evidence
   stays in `var/screenshots/`; `docs/screenshots/` is frozen. A single
   QA recipe and the known tooling traps live in the agent-browser skill.

## Consequences

Easier: a branch closes in one command and one short file; a CSS fix pays
seconds of gates, not minutes; the handoff is readable in one screen and
cannot regrow; restarts happen three times a day at known hours, and
never mid-turn; a green GitHub run means something again.

Harder: production lags `main` by up to a few hours (the owner can run
`make deploy-batch` any time; the guard still applies); the public site's
images can trail the UI until the next batch; a session that wants the
full matrix in its worktree runs `make ci` explicitly; the readiness route
is one more loopback surface (it names terminals and reasons, never
content, and is guarded off-loopback like every other API).

If wrong: a deploy that must not wait uses `--force`; `DOCS_STRICT=1
make docs-check` restores the blocking gate; the cap is one number in
`.githooks/pre-commit`. Nothing here touches the product's data plane.

With ADR-0085's `trap '' HUP` the observed restart on 2026-09-06 18:50 left
all nine sessions alive, so the guard now protects the remaining costs of
a restart — interrupted managed turns, dropped terminal streams, the
reconnect every viewer pays — and the batch keeps those to three known
moments a day.

## Decision table — `picode deploy`

| Daemon answer | `--force` | Action |
|---|---|---|
| no `server.json`, nothing listening, or 404 (older daemon) | any | deploy (nothing to protect) |
| 200, `busy` empty | any | deploy |
| 200, `busy` non-empty | no | refuse before copying the binary; exit 2; name each owner and why |
| 200, `busy` non-empty | yes | deploy |
| answer unreadable | any | refuse with the parse error |

Coverage: `TestDeployRefusesWhileAgentsWork`, `TestReadinessDecisionTable`
(internal/install), `TestDeployReadinessListsEveryBusyOwner`
(internal/server), the auth table rows for loopback vs remote, and the
hook self-test rows for the 100-line cap.

## Alternatives considered

- **Keep deploying per merge, make resume cheaper** (ADR-0084's answer):
  rejected — it pays for the interruption instead of avoiding it, and
  does nothing for the twenty-odd restarts a day.
- **A separate staging instance for agents**: deferred — two daemons and
  two data dirs double the dogfood surface; batching plus the guard removes
  the harm with no new topology.
- **Generate the handoff entirely from git**: rejected — in-flight state,
  debts and next steps are judgement, not history. Only history was moved
  out.
- **`t.Parallel()` across `internal/server`**: rejected for now — its
  tests swap package-level probe functions; parallelism would race them.
  Scoping avoids the suite for non-Go diffs, which is where the minutes
  went.
- **Rewrite git history to drop the 339 committed UI bundles (567 MB)**:
  owner's call, outside this ADR.
