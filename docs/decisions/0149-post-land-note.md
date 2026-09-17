# ADR-0149: A post-merge correction to an existing handoff note lands on main

- **Status**: accepted
- **Date**: 2026-09-17
- **Boundary**: process — where a fact learned *after* a branch merged is written down
- **Amends**: ADR-0105 (decision 4, and the `main`-in-root rows of its pre-commit decision table)

## Context

ADR-0105 made handoff state reach `main` only inside a fast-forwarded branch,
and ADR-0086 before it made shipped work `git log` + the changelog, with the
handoff note a **session** record. Both are right, and both leave one case with
no door: the fact that cannot exist before the merge.

Two of them, the same day (2026-09-17):

- The desktop shell's wordmark click was fixed on a branch (`c7e2c58d`). Its
  note said, true when written, that the owner's Windows click was the last
  check. The owner confirmed it an hour later. Correcting that sentence — one
  line, in the file a reader opens before redoing the verification — would have
  cost a worktree, a branch, a scoped gate run and the landing `make ci`: the
  rite ADR-0105 measured at ~10 minutes of gates per branch.
- A one-line debt paid in the field ("Owner-confirmed 2026-09-17: tab reorder
  works in the restarted shell") did pay it — `feat/pay-dnd-debt`, one changed
  line in `docs/handoff/open/desktop-shell-dnd.md`, through the whole
  branch-and-land machinery (`main`'s reflog: `merge feat/pay-dnd-debt:
  Fast-forward`). A durable item discovered *after* the merge inherits the same
  cost, because a new `docs/handoff/open/<topic>.md` is refused on `main` too.

The ban ADR-0105 wrote was aimed at the opposite kind of commit: 65 direct
`main` commits in three days, about half "record the merge/deploy" prose that
recorded a ritual and corrected nothing. A correction exists because someone
read the note and found it wrong — or confirmed it right. That is the one kind
of direct commit the ban should not cover.

## Decision

A commit on `main` in the root checkout may **modify** anything already tracked
under `docs/handoff/` (a session note or a topic file) and may **add** a file
under `docs/handoff/open/`. Everything else stays as ADR-0105 left it: a
changelog fragment, a *new* session note, and `docs/handoff.md` still reach
`main` only by fast-forwarding a branch, and `make handoff` remains the only
writer of the board. The door is one `git commit`, no worktree, branch or gate
run, because the fact it carries cannot be produced earlier.

## Consequences

Easier: a field confirmation, a debt paid by observation, or a correction to a
claim a note made is one commit beside the claim; nobody redoes a verification
because the note was left saying "not done yet".

Harder, and accepted: `main`'s history gains small `docs:` commits. The door is
bounded by what it can touch — no code, no fragment, no new note — so it cannot
become the pile ADR-0105 banned, but it is a wider surface than "refuse
everything". Prose can now change without a gate ever seeing it; the gates
never covered prose.

If wrong: the hook's rule is one condition — restore ADR-0105's grep and the
door closes again. Nothing in the product or the data plane is involved.

## Decision table — the `main`-in-root rows of `.githooks/pre-commit`

| Staged | Action |
|---|---|
| a `docs/changelog.d/*` file | refuse (unchanged: a fragment travels with its branch) |
| a new `docs/handoff/<date>-<branch>.md` | refuse (a note is born in the branch it records) |
| a modification of a tracked `docs/handoff/**` file | allow (the correction lands next to the claim) |
| a new `docs/handoff/open/<topic>.md` | allow (a durable item's home, ADR-0145) |
| `docs/handoff.md` | refuse (unchanged, ADR-0123: generated) |
| anything else | unchanged from ADR-0105 |

Coverage: `scripts/hooks-selftest.sh` (37 rows, run by `make hooks-check` and
`make ci`).

## Alternatives considered

- **Write the confirmation to the deploy log** (`~/.picode/var/deploy-log.jsonl`
  already records `at` + `0.3.1+<sha>` per deploy): no policy change, but the
  record would sit outside git while the claim it corrects sits inside it, and a
  reader of the note would never see it. Rejected — the door has to be where the
  sentence is.
- **Allow any docs-only commit on `main`**: reopens the 65-commit pile ADR-0105
  measured, and lets fragments ride along.
- **Do nothing, and stop writing lines that expire** (notes state method and
  blind spot, never "awaiting check"): removes most of the need and is now a
  rule of `/skill:handoff-update` — but a fact learned later still has nowhere
  to go, and the branch-per-`- [x]` case above is the proof.
