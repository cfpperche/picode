# ADR-0188: A terminal identity keeps the browser's session drive

- **Status**: accepted (owner session, 2026-09-22)
- **Date**: 2026-09-22
- **Boundary**: security model — what a caller identified only by a
  terminal with no agent (`term:<id>`: a shell, a sign-in) may do with the
  work browser.
- **Amends**: ADR-0184 (its decision-table row "Grant request as
  `term:<shell or sign-in>` → refuse", for the browser only).

## Context

ADR-0184 made grants an agent's alone and its table says a request as
`term:<shell or sign-in>` is refused. The shipped code refuses stored grants
for a terminal key everywhere (computer off, browser default, delivery 403,
grant edits 400) but kept one thing: ADR-0172's session drive, which lets an
identified caller act on any http(s) URL in *its own* browser split. An
adversarial review (2026-09-22) flagged the gap between the row and the
code. The owner chose to keep the behaviour and correct the record.

## Decision

A caller identified only by a terminal id with no agent keeps the work
browser's session drive (ADR-0172) in its own split, and nothing else: no
stored browser or computer grant, no raw protocol beyond the default tier,
no delivery. Every other row of ADR-0184 stands.

## Consequences

A CLI typed into a plain PiCode shell can still drive the browser beside its
own session, as it could before ADR-0184 — the split is the human's to see.
It cannot be granted anything more without becoming an agent (Make agent).
If this is wrong, the exposure is a shell's own browser split, not the
human's tabs or desktop.

## Alternatives considered

- **Refuse the session drive for terminal identities (ADR-0184 as
  written).** Lost: the owner kept it — a shell running a CLI loses the
  browser beside it for no gain the split does not already show.
