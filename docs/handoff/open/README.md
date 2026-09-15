# Open topics

One file per open subject — `## Next` for work not started, `## Debts` for what
is accepted but not paid. `make handoff` (ADR-0123, ADR-0131, ADR-0140) reads every file
here into `docs/handoff.md`; edit the topic file, never the board.

Bullets are the unit: one line, no sub-lists, and `Plan: docs/plans/<x>.md`
when a plan carries the detail. Bullets carry their proof: name the branch,
test, or command that pays the item, so a future prune is a check, not
archaeology. Delete a bullet in the branch that paid it — in this file *and*
in any older session note still echoing it; that deletion is the record.

**One home per item.** A live item lives in a session note (transient — the
board drops a note's sections 30 days after its date) or in a topic file
(durable), never both. Promoting an item here deletes the note's echo; a
debt that outlives its note must move here before it ages out, or it leaves
the board silently (the note keeps its text in git either way).

**Never bullet-ize a generator failure.** When `make handoff` fails over
budget it already names the files; prune them instead of filing the failure
as a bullet — a meta bullet spends the bytes it complains about.

**Two headings the board reads, and the file must use them.** The board renders
your `## Next` bullets inline, and your debts as one line — the count, this
file's path and its `Plan:` — so a file whose bullets sit under any other
heading (or under none) contributes nothing. `make handoff` fails and names the
file when *every* bullet in it is invisible; other headings (`Traps`, `Slice 3`)
are fine as long as the file has its two.

Write the debt where the reader will look for it, not where it is cheapest:
a count without a `Plan:` sends the reader hunting, and the `Plan:` line is
what makes the count actionable.
