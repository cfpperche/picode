# ADR-0140: Session-note debts expire from the board after 30 days

- **Status**: accepted
- **Date**: 2026-09-15
- **Boundary**: process — what the handoff board renders, superseding ADR-0131's "session-note debts still render inline"
- **Supersedes**: ADR-0131 (one line: note debts no longer render unbounded)

## Context

ADR-0131 decided session-note debts "still render inline (they are few and
they age out of relevance with their branch)". Both halves of that
parenthesis failed against measurement:

| Assumption | Measurement, 2026-09-15 |
|---|---|
| Note debts are few | 20 note-debt bullets render, from 18 notes |
| They age out | `render()` filters notes' Next by 30 days but maps **all** notes' Debts — a note's debts render forever |

This is the second over-budget incident in three days (0131's own context
records the first, 2026-09-13): the board hit 118 lines / 13 128 bytes
against 120 / 12 288 caps, failing `make close` on every branch until a
session spent its effort pruning other topics. The Next-side expiry works
— the Debts side is the same leak one section down.

Notes are transient by design (325 exist; the durable home for a debt is
`docs/handoff/open/<topic>.md`). Expiry costs nothing at merge time: all
18 Debts-bearing notes are fresh today, so the board renders byte-identical
minus nothing — the change only stops future accumulation.

## Decision

Session-note `## Debts` bullets render on the board only while their note
is fresh (≤ 30 days, the same window as `## Next up`). A debt that still
matters past that belongs in `docs/handoff/open/<topic>.md`, and the
writer conventions in that directory's README say so: one home per item,
name the branch/test/commit that pays it, reconcile echoes in the paying
branch, never bullet-ize a generator failure.

## Consequences

- The board stops growing with note debt: the unbounded side is now only
  the per-topic count lines, which is what 0131 intended.
- Sessions promoting a durable debt must move it to a topic file before
  the note ages out, or it leaves the board silently — the README rule
  makes this explicit, and the note's text stays in git either way.
- If this is wrong, inputs are untouched: every bullet stays in its note
  file, and reverting one filter in `scripts/handoff-board.mjs` restores
  the inline form. Re-measure the 12 KB cap only after this lands.

## Alternatives considered

- **Keep rendering all note debts and prune harder.** That is the incident
  being decided: two sessions in three days paid it, and the backlog side
  grows forever, so pruning is a treadmill with a 30-day delay.
- **Raise the byte cap.** Refused twice already (0123's alternative,
  0131's re-measurement): it moves the cliff without shrinking the board.
- **Expire topics' debts too.** Topics are the ledger — their debts render
  as count lines, which do not grow the board per bullet. Nothing to gain.
