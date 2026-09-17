# ADR-0145: The handoff board is a bounded view

- **Status**: accepted
- **Date**: 2026-09-16
- **Boundary**: process — what the board renders, and what over-target means
- **Refines**: ADR-0123 (derived, budgeted, never hand-written), ADR-0131
  (index; an invisible topic is an error) and ADR-0140 (note sections expire).
  The derivation, the git-ignored file, the hooks and the two headings stay as
  decided; the budget's *meaning* changes.

## Context

ADR-0123 gave the board a hard cap (120 lines / 12 KB) enforced by making
`make handoff` fail, with the failure as a prerequisite of `make close` and
`make ci`. It assumed the sources are trimmed as work lands. They are not:
they are append-only by design — every session writes one note (ADR-0105 §1),
every topic keeps its open items, and notes stayed on the board for 30 days
(ADR-0140).

Measured 2026-09-16, four agents on the repo: 25 topics, 379 notes, and every
session in one day hit the cap at its *closing* step — the point of peak
context. The only available fix was to delete bullets from other sessions'
notes: four prune rounds in one day, one of them a line that had stopped being
true ("the options menu's page slide is not animated" — the slide was removed
hours earlier), and one that deleted a bullet whose author had not been asked.
Two further costs showed up in the same window: the byte shopping produced
nothing for the product, and hand-pruning another agent's prose is exactly the
edit a multi-agent repo cannot verify.

The arithmetic is the point: 25 topics × (one next + one debt line) already
fills the cap before a single session note contributes. A complete ledger and a
fixed 12 KB view are incompatible; the cap was never a pruning pressure, it was
a scheduled conflict.

## Decision

The board is a **bounded view** and truncates by construction:

- **Per topic**, at most 2 `## Next` bullets, then `…+N more in
  docs/handoff/open/<topic>.md`. Debts stay one line per topic: the **open**
  count, the file, the plan.
- **Per session note**, at most 1 bullet per section, and only while the note is
  ≤ **7 days** old (was 30): a note is a handoff to the next session, not a
  ledger. At most 8 note bullets per section reach the board; the rest are
  named by one pointer.
- **Debts carry state in the topic file**: `- [ ]` (or a plain bullet) is open,
  `- [x]` is paid. Paid debts leave the board and stay in the file — paying a
  debt is flipping a box in the file that owns it, never deleting another
  session's line.
- **Over target warns; it never blocks.** The two targets (120 lines / 12 KB)
  stay as the readability bar, and over them the generator names the largest
  sources and *still writes the board*. `make handoff` fails only when a source
  cannot be read — a topic file whose every bullet sits outside the two headings
  (ADR-0131 A) — because that is a defect in the file, not a size.
- The durable home of a live item stays the topic file; a session note's bullets
  are a courtesy that expires.

## Consequences

Easier: `make close` and `make ci` cannot fail on how much the team wrote down;
a session never edits (or deletes) another session's prose to fit a budget; the
board is small enough to actually read, and the pointer line tells the reader
where the rest is. Paying a debt leaves a visible record instead of a hole.

Harder / accepted as cost: the board is no longer a complete list of next steps
— a topic with five open items shows two. The reader who needs all five opens
the topic file, which the pointer names. A session that wants its item on the
board must put it in the topic file (already the rule) and keep it among the
first two (a real constraint: order becomes meaning — the top bullet is the
one being worked).

Failure mode to watch: if a topic's top-two bullets go stale, the board lies by
selection. The mitigation is the same as before — pay the item and move the
next one up — and it is visible in one file instead of across 379 notes.

## Alternatives considered

- **Raise the cap.** Refused: it buys one or two more agents' worth of days and
  keeps the same failure (fixed budget, append-only sources), and every raise
  makes the view less readable — the thing it exists for.
- **Trim notes at `make close` mechanically into the topic file** (this is the
  next step, a separate change) — accepted as the *follow-up*, not as the fix:
  it removes the note contribution entirely, and it needs its own migration for
  379 existing notes. The quota above bounds the board whether or not that
  lands.
- **Drop note bullets from the board now.** Refused for now: for a single
  session's handoff the note is where the next agent looks first, and the 7-day
  1-bullet quota costs little.
- **Keep failing over budget, but prune automatically by dropping the oldest
  bullets.** Refused: silent deletion of prose is the failure this ADR exists to
  end, and "oldest" is not "paid".
