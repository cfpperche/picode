# ADR-0131: The handoff board is an index, and an invisible topic file is an error

- **Status**: accepted
- **Date**: 2026-09-13
- **Boundary**: process — what the board carries, and what happens when a topic file cannot be read
- **Refines**: ADR-0123 (the board is derived; budgeted; never hand-written). Its rendering rule and its budget are what change here; the derivation, the git-ignored file and the hook stay as decided.

## Context

Two defects measured 2026-09-13, one session after ADR-0123's board went live (numbers in `docs/plans/handoff-board.md`):

| Defect | Measurement |
|---|---|
| A topic file could be invisible | `docs/handoff/open/snippets.md` listed six debts under an `#` H1 with no `## Next` / `## Debts` heading. The generator reads only bullets under those headings, so **none** of them reached the board — for every session that read it. It was found by writing a session note, not by any check. |
| The two caps contradict each other | Board at 97 of 120 lines but 12 232 of 12 288 characters. The average bullet is 101 characters and renders at ~126 with its attribution, while the caps assume ~102 per line, so the byte cap binds about 20 lines early. One session filing one honest debt could fail a gate every topic shares — that session spent its effort pruning other topics to fit. |

The growth is structural: **next steps are bounded** (a topic has one or two), **debts are not** (they accumulate until someone pays them). Rendering every debt bullet inline makes the board scale with the backlog, which is exactly the failure ADR-0105 and ADR-0123 were written to end.

## Decision

The board renders two different things, because the two sections behave differently:

- **`## Next up`** keeps its bullets inline, with attribution — it is short, and it is the board's actionable part.
- **`## Debts / open questions`** carries **one line per topic**: the count, the topic file, and the topic's `Plan:` path when the file declares one. The bullets stay in `docs/handoff/open/<topic>.md`, which remains the ledger and the place a paid debt is deleted. Session-note debts still render inline (they are few and they age out of relevance with their branch).
- `make handoff` **fails, naming the file**, when a topic file lists bullets but none under a recognized heading — the invisible-file class is refused instead of silently dropped. A file with its own extra sections (`Traps`, `Slice 3`) is unaffected: only the two headings are the board's business.

## Consequences

- The board stops growing with the backlog: 12 232 characters became ~6 300, and 48 debt bullets became ~20 topic lines. The budget stays as the pruning pressure, now with room to absorb a normal session's work.
- A reader pays one hop for detail — open the topic named on the line. In exchange the board answers *which topic has debt and where its plan is*, which is what a session needs to decide whether to look.
- The silent failure is gone: a mis-shaped topic file is now a failed gate in the branch that wrote it, not a debt nobody sees.
- Accepted costs: the board no longer shows debt text at a glance, so a session that wants the words opens files; a topic line is a count, and counts can be stale in *meaning* even when the file is current (a count of one can be a crisis, a count of seven can be trivia). Both are the price of not rendering an unbounded set.
- If this is wrong, the inputs are untouched: every bullet is still in its topic file, and reverting one function in `scripts/handoff-board.mjs` restores the inline form.

## Alternatives considered

- **Keep rendering every debt and prune harder** (the branch that preceded this decision did exactly that). It worked once, by deleting other topics' verified-shipped bullets — and left ~150 characters of headroom, so the next session pays again. It treats a structural mismatch as a discipline problem.
- **Raise the byte cap** (ADR-0123's alternative, re-measured). It makes the contradiction disappear without making the board smaller; the backlog grows forever, so the cap only moves the cliff.
- **Drop the board's next-up section too**, leaving only counts. Next steps are bounded and are what a session reads the board for; compressing them saves little and costs the section's whole purpose.
- **A warning instead of a failure** for the invisible file. A warning in a command whose output nobody reads is how the six debts stayed invisible for a day; the gate is the only reader that always sees it.
