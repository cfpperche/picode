# ADR template

`make adr NAME=<short-title>` copies the part below the rule into
`docs/decisions/NNNN-<short-title>.md` with the next free number across every
worktree, and appends the row to the index in `docs/decisions/README.md`.
ADRs are immutable once accepted — to change a decision, write a new ADR that
supersedes or amends the old one and say so in both index rows.

**Boundary first.** An ADR records a decision that crosses a boundary:
protocol, persistence, security model, or process. If the Boundary line
below has nothing honest to say, this is not an ADR — write the paragraph in
`docs/architecture/<subsystem>.md` or a note in `docs/plans/` instead.

---

# ADR-NNNN: Title

- **Status**: proposed | accepted | superseded by ADR-XXXX
- **Date**: YYYY-MM-DD
- **Boundary**: protocol | persistence | security model | process — which one, and what crosses it

## Context

What forces are in play? What problem demands a decision? Facts, options,
constraints — not opinions.

## Decision

What we will do, stated in the present tense, one paragraph, no hedging.

## Consequences

What becomes easier, what becomes harder, what we accept as cost.
Include the "who/what breaks if we're wrong" analysis.

## Alternatives considered

Each alternative and the specific reason it lost.
