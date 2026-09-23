# Missions (ADRs 0199–0200)

Missions preserve outcomes independently of agent activity and native sessions.
The [plan](../plans/missions.md) defines the acceptance scenarios; the store
owns lifecycle, history, idempotency and reservations. Server observations own
Git identity, revision, file digests, session identity and detected activity.

## Contract

| Layer | Responsibility |
|---|---|
| `internal/store/missions*.go` | Atomic state/history/receipt/event writes; unique agent reservation; owner-only acceptance |
| `internal/server/missions.go` | Owner routes, attributed agent tool route, filesystem/Git observations, serialized submission |
| `cmd/picode/mission.go`, `internal/mcptool/mission.go` | One reporting contract over inherited launch identity |
| `web/shared/domain/missions.js`, `client/useMissions.js` | Route/draft/action model and injected hook lifecycle, with no React dependency |
| Browser and mobile Missions views | Independent presentation and navigation using shared contracts |
| `internal/apps/inbox.go` | Record correlated mission answers through the store; no implicit terminal delivery |

States are ready, in-progress, blocked, in-review, paused, completed and
cancelled. The assignment independently records prepared, unconfirmed or
acknowledged delivery and whether it still reserves the agent. Pause/cancel
retain reservations. A stopped agent is opened/started through its existing
surface; Missions does not introduce a launcher or background scheduler.

All mutations require `requestId`, and updates require `expectedVersion`.
Agent writes also require assignment `generation` and a known matching native
session. Replaying the same payload returns the original receipt. Reusing a
key with another payload conflicts. A per-mission server mutex serializes
owner operations with sends; the store transaction makes competing agent
reservations exclusive. External runtime and filesystem activity remains
outside that transaction and must be observed and owner-confirmed.

## API and recovery

- `GET /api/missions?workspace=…&before=…`: descending sequence, 101 records
  to detect another 100-record page.
- `GET /api/missions/{id}?before=…`: current state, deterministic context,
  history page and fresh observations. Observation failure preserves history.
- `POST /api/missions`: owner creation.
- `POST /api/missions/{id}/actions`: versioned owner action.
- `POST /api/missions/tool`: attributed read/report contract used by CLI/MCP.

Restoring a missing workspace is an explicit `relink` action after releasing
the executor. Repository identity must match; plain folders require owner
confirmation of prepared files. The assignment counter stays monotonic and
restoring a workspace starts a new evidence scope. Agent action menus link
to their currently reserved mission.

No prompt retry follows a timeout, restart, reconnect or feed reset. The
unconfirmed receipt remains visible. The owner opens the target, confirms
receipt or stops it before assigning again. Agent reports cannot automatically
resume paused/cancelled work. Context carries objective, criterion IDs,
decisions, latest checkpoint, blocker, next action, generation and working
folder, with a digest, up to eight recent reported attempts and bounded evidence
references. Native transcripts, file contents and credentials are omitted.
Evidence remains accessible through `show`. Browser retries retain the exact
original request payload, including its version, across form reloads.

## Review and retention

Evidence is a reporter's observation, a confined file reference (regular,
at most 4 MiB, hashed through `os.Root`), or an existing Delivery ID in the
same repository and candidate revision. A link never transfers Delivery author
authority. Evidence capture in a Git repository requires a clean committed candidate.
Initializing Git in a previously plain folder binds the new repository identity
and invalidates evidence from the earlier scope. Every criterion needs
current-scope passing evidence. Pending review
checks scope, clean HEAD and current file/Delivery references again at accept.
Historical acceptance survives subsequent code changes; new work requires
reopening. Requested changes invalidate the old evidence scope.

Limits: 2,000 unarchived missions, 2,000 work updates per mission, 32 criteria (1,000 bytes
each), 128 evidence entries (8,192 bytes each), 200-byte title, 8,192-byte
objective/checkpoint, 32,768-byte context and 2,000-byte next action. Requests
are limited to 80 KiB. Cancel, release and archive remain available at the
update limit so capacity cannot trap an executor. Archive hides terminal missions without deleting their
history. There is no automatic retention purge. Use PiCode's database backup
to retain the full ledger; context copy is a continuation packet, not a full
backup. Full snapshots and receipts increase storage with mission history.

## UI adaptation

Workspace menus and the browser command palette open a compact list/detail
flow. Mobile additionally exposes Missions under More. Cursor's artifact
review informs criterion/evidence pairing; t3code's persistent result links
inform Delivery references; Paseo's explicit workspace boundary informs the
transfer preview. See the benchmark references in the approved plan. There
is no new dashboard or shared presentation component between apps.
