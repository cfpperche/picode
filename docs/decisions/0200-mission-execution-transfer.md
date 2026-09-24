# ADR-0200: Mission assignment and controlled transfer

- **Status**: accepted
- **Date**: 2026-09-23
- **Boundary**: protocol and process — assignment authority, acknowledgements and recovery across native CLI sessions.

Amended by [ADR-0213](0213-mission-native-session-binding.md) for Pi's first
managed-session attribution when assignment precedes session creation.

## Context

The approved mission plan requires one responsible executor while preserving
PiCode's native CLI lifecycle (ADRs 0091 and 0160). Prompt submission can time
out after the native process received it. Terminal idle signals cannot prove
that a child process stopped writing.

## Decision

The owner binds a mission to an existing workspace agent, native session,
working folder and generation. Assignment requires explicit target readiness;
transfer additionally requires confirmation that the source and its children
stopped writing. Detected activity blocks responsibility changes. Different
working folders require explicit confirmation that files are prepared, a clean
source and the same candidate commit when using Git; files
are never silently copied. A native session rollover requires a new binding.

Persist `unconfirmed` before prompt submission. Reuse managed Pi `SendTurn`, Pi
interactive delivery, or the existing guest-terminal prompt door. Never retry
an uncertain send automatically. Acknowledgement is a separate attributed
operation. Owner confirmation is labelled as such. Pause/cancel block new
mission sends, request managed interruption where available, and retain the
reservation until the owner confirms quiescence. Other CLIs stop through their
existing terminal controls. Daemon restart never starts new execution.

`picode mission` and the optional `mission` MCP family inherit launch identity.
They may read their assigned mission and acknowledge, report, block, attach
evidence or request review. They cannot assign, transfer or accept. Generation,
version and observed native session must agree. This reuses PiCode's same-user
launch attribution, **not** a sandbox against a malicious CLI with the daemon
owner token (ADR-0096). No new peer-contact grant is created.

Inbox questions capture assignment generation, native session and scope.
An answer to the current blocker updates mission context. An old answer is
retained as history and never reactivates another assignment. Answers are not
implicitly pasted into terminals. Generic Inbox triage never accepts a mission.

## Consequences

Recovery stays explicit and preserves uncertain writer reservations. Manual
continuation remains possible through a deterministic context packet. Live
vendor behavior must be measured separately from fixtures. This change does
not add managed mode to other CLIs, automatic agent launch, a reviewer role,
or unattended scheduling. Those remain the later plan milestones.

## Alternatives considered

- Release on pause/idle: can create a second writer while a child still runs.
- Retry a timed-out paste: can execute the objective twice.
- Share native transcripts: unnecessary provider and privacy coupling.
- Kill processes automatically on transfer: risks unrelated work and drafts.
