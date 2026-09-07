# ADR-0092: An absent checklist renders as silence

- **Status**: accepted
- **Date**: 2026-09-06
- **Amends**: ADR-0055 (internal checklist), ADR-0081 (terminal checklists)

## Context

ADR-0055 made an unmet checklist obligation visible: when a required plan
was missing (a turn ended without one under `always`, or a mutating tool
call was refused), the shells rendered a discrete amber **No checklist**
line on the agent's card — "accusing an agent that could not write a plan"
was deliberately avoided, but accusing one that *could* and *did not* was
the point. ADR-0081 carried the same line to terminal cards.

Dogfooding showed the line reads as noise in practice: it appears during
ordinary read-only work, sits on cards for sessions that will never write
a plan, and its amber draws the eye to a state the owner cannot act on.
The owner's call (2026-09-06): if there is no checklist, show no line at
all — no counter, no "No checklist".

## Decision

The absent marker keeps its meaning in the **data plane** — pi-checklist
still publishes `absent`/`blocked`, the store still records it, the feed
still announces it — but the **presentation** renders it as silence, the
same as "nothing known". The one-line projection `{kind:"absent"}` stays
in the shared domain helpers; the components render `null` for it (desktop
sidebar cards, mobile agent rows). The blue accent on
the `(x/n)` counter goes with it: the operator line is one muted line, not
a status signal.

## Consequences

A met-or-unmet obligation is no longer visible anywhere; only a real list
shows. The gate's refusal (the actual enforcement, ADR-0055) is unchanged
and remains the moment a missing plan becomes visible to the *agent*. The
cost is honesty in the other direction — a card says nothing about whether
the contract applies — accepted because the owner cannot act on the
information and the amber implied urgency nothing had.

## Alternatives considered

- Keep "No checklist" but mute it to plain text. Rejected by the owner:
  any line is noise when it cannot be acted on.
- Show the line only while the turn that missed the plan is live. Rejected:
  more state to keep honest for no actionable gain.
