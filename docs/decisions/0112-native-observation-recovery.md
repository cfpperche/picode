# ADR-0112: Recover native observations across daemon restarts

- **Status**: accepted (owner approval, 2026-09-10)
- **Date**: 2026-09-10
- **Boundary**: persistence and protocol — keep the latest native terminal observation outside the daemon, then reconstruct the live registries only after process and pane validation.

## Context

After deployment, tmux preserves the six native TUIs while PiCode loses their
in-memory identities and activity. Presence recovery proves a process, not its
conversation. HTTP-only reporters also lose events during daemon downtime.
The Messages view incorrectly classified the pending identification as an error.
The owner approved recovery and independent activity/connection presentation.

## Decision

Amend ADR-0056/0062's ephemeral-only observation choice and ADR-0107's restart
recovery. The existing native hook saves one private, versioned JSON observation
per terminal before HTTP delivery. It contains CLI, wrapper run/PID/start token,
OS boot ID, native session/path, event sequence, activity and Codex source precedence;
never credentials, prompts or message bodies. Atomic replacement under a local
file lock orders concurrent writers. The lock file retains a private sequence,
source and end fence before checkpoint publication; recovery requires both records
to agree. Failed publication preserves that fence, so a delayed Idle cannot replace
the newer failed event. If even updating the fence fails, its incarnation is blocked
until a fresh wrapper start; ordinary events cannot reset it. Runtime start/end prevents another incarnation
from inheriting the record. Native events continue recording while PiCode is down.

The existing presence watcher validates the record against the current owner,
CLI, process incarnation and exact pane ancestry, then republishes identity and
activity together. The original event age still bounds Working. Corrupt, absent,
foreign or stale observations cannot become Idle. Automatic attention additionally
checks the current record before its existing live-session, draft and permission
guards. The mailbox, opt-in, revocation, ACK and uncertain-attempt rules do not change.
There is no new daemon, runtime engine, dependency or SQLite migration.

Pi's existing terminal receiver heartbeat runs every five seconds and reconciles
the exact wrapper before accepting its connection report. It proves receiver
presence only, never activity. Desktop and mobile distinguish activity, connection
preparation and historical test proof; pending identification is not an error.
Enabled unavailable participants remain visible with their connection reason.

Hermes' native background review can overwrite the process-wide session variable.
Its adapter selects only an explicit root CLI/TUI pre-LLM session/turn pair and
accepts completion/approval events for that pair. Helper turns cannot mark the
root Idle. The native pre-tool hook binds one plain `picode messages` invocation
to that same pair with a command-local environment assignment; compound commands
and other tools remain untouched. This uses Hermes' supported argument-modification
contract before its normal approval checks, never a saved-session fallback or
process-global environment repair. Event sequences are captured before hook I/O.

## Consequences

Linux/WSL gains restart recovery from native events without a new model turn or
CLI restart. Existing common hooks adopt recording on their next actual event;
older processes cannot acquire observations retroactively. A missing observation
still requires native evidence, not the saved session or newest file. Other OSes
retain live reporting and unknown state after restart until their process identity
mechanism is validated. The native adapter event coverage remains the trust limit;
this is local capability authorization, not hostile-process attestation.
If storage refuses every write, invalidation and permission change while still
serving old bytes, no local recorder can persist the failure; power-loss durability
and such total storage faults are outside this daemon-restart guarantee.

## Alternatives considered

- Persist only the server map: misses events while the daemon is unavailable.
- Restore last-session pins as Idle: confuses history with current native state.
- Poll or restart every CLI: adds vendor-specific control and risks drafts.
- Label every pending state Needs attention: falsely implies a native approval.

Validation: [decision table and acceptance](../plans/communication-recovery.md).
