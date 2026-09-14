# Open topics

One file per open subject — `## Next` for work not started, `## Debts` for what
is accepted but not paid. `make handoff` (ADR-0123, ADR-0131) reads every file
here into `docs/handoff.md`; edit the topic file, never the board.

Bullets are the unit: one line, no sub-lists, and `Plan: docs/plans/<x>.md`
when a plan carries the detail. Delete a bullet in the branch that paid it —
that deletion is the record.

**Two headings the board reads, and the file must use them.** The board renders
your `## Next` bullets inline, and your debts as one line — the count, this
file's path and its `Plan:` — so a file whose bullets sit under any other
heading (or under none) contributes nothing. `make handoff` fails and names the
file when *every* bullet in it is invisible; other headings (`Traps`, `Slice 3`)
are fine as long as the file has its two.

Write the debt where the reader will look for it, not where it is cheapest:
a count without a `Plan:` sends the reader hunting, and the `Plan:` line is
what makes the count actionable.
