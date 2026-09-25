# Missions (ADRs 0199–0200)

Missions preserve outcomes independently of agent activity and native sessions.
The [plan](../plans/missions.md) defines the acceptance scenarios; the store
owns lifecycle, history, idempotency and reservations. Server observations own
Git identity, revision, file digests, session identity and detected activity.
Pi's first-session attribution amends [ADR-0200](../decisions/0200-mission-execution-transfer.md)
through [ADR-0213](../decisions/0213-mission-native-session-binding.md).

## Contract

| Layer | Responsibility |
|---|---|
| `internal/store/missions*.go` | Atomic state/history/receipt/event writes; unique agent reservation; owner-only acceptance |
| `internal/server/missions.go` | Owner routes, attributed agent tool route, filesystem/Git observations, serialized submission |
| `cmd/picode/mission.go`, `internal/mcptool/mission.go`, `internal/pimission` | One reporting contract over inherited launch identity; Pi also receives an embedded native extension in managed and interactive launches |
| `web/shared/domain/missions.js`, `client/useMissions.js` | Route/draft/action model and injected hook lifecycle, with no React dependency |
| Browser and mobile Missions views | Independent presentation and navigation using shared contracts |
| `internal/inboxview/inbox.go` | Record correlated mission answers through the store; no implicit prompt send to a terminal |

States are ready, in-progress, blocked, in-review, paused, completed and
cancelled. The assignment independently records a prompt receipt as prepared,
unconfirmed or acknowledged, and whether it still reserves the agent. The wire
field `assignment.delivery` names that receipt, not the Delivery feature. In
Missions prose and UI, call it a *receipt* or *handover*; reserve **Delivery**
for the artifact and integration feature. Pause/cancel retain reservations. A
stopped agent is opened/started through its existing surface; Missions does not
introduce a launcher or background scheduler.

All mutations require `requestId`, and updates require `expectedVersion`.
Agent writes also require assignment `generation` and a known matching native
session. The CLI/MCP boundary validates these three fields before posting a
mutation, with a field-specific message; `show` and `context` remain available
without mutation fields. The daemon remains authoritative for current-version,
assignment and session checks, including refusing stale generations. Replaying
the same payload returns the original receipt. Reusing a key with another
payload conflicts. A per-mission server mutex serializes
owner operations with sends; the store transaction makes competing agent
reservations exclusive. External runtime and filesystem activity remains
outside that transaction and must be observed and owner-confirmed.

Pi's native `mission` tool is embedded in PiCode and injected with `-e` into
managed RPC and interactive agent launches. It uses only Node built-ins and
Pi's extension registration API; it does not require `pi-mcp-adapter`, a
workspace package installation, or `.pi/settings.json`. The tool sends the
same `/api/missions/tool` payload as the CLI/MCP implementation and stamps the
inherited agent or terminal identity; it sends no client-provided native
session field. The server resolves the current native session from PiCode's
managed pending-session ledger, terminal runtime, or pinned last session for
writes. Assigned `show` and `context` reads need no mutation fields or session
binding. If a mission was assigned before the first session existed, its first
acknowledgement binds that observed session through the existing store rule. A
later mismatch is refused. The extension is available
only in PiCode-launched Pi processes; it is not a general Pi MCP server or a
new authority path.

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
Browser and mobile detail views place a **Read evidence** action beside the
review controls. It opens the criterion reports for the review revision;
individual historical evidence records can also be expanded in History. Notes
may name local files, but those names are references, not attached content.

## Where Delivery begins

| Dimension | Mission | Delivery |
|---|---|---|
| Question | Did the owner-defined outcome meet its criteria? | What artifact and revision is being proposed for review or integration? |
| Unit | Mission ID, assignment generation and evidence scope | Delivery ID, source revision and target; a separate queue entry for integration |
| State | Ready, in progress, needs owner, in review, paused, completed or cancelled; prompt receipt is separate | Author's review request is intent; queue entries move through waiting, authorized, running and done/failed, or withdrawn |
| Authority | Agent reports; owner assigns, transfers and accepts the result | Delivery author declares and requests review; owner authorizes integration; the declared provider or local runner performs it |
| Idempotency | Mission `requestId` and version; agent writes also bind the assignment generation and native session | Delivery declaration retry key and version; queue request receipts replay an identical request |

The link is stored in one direction and read in both: Mission evidence may cite
an existing **Delivery ID** (`kind: "delivery"`) in the same repository and
candidate revision, and the Delivery read derives the missions that cite a
delivery, so a row can name the objective it serves. Nothing in Delivery derives
authority from a Mission. Linking evidence never transfers Delivery author
authority, approves integration or places an entry in its queue. **Request review** also has two
meanings: in Missions it asks the owner to check the current criteria and
evidence before owner acceptance; in Delivery it is the agent author's
declaration that the artifact is ready for the owner to inspect (ADR-0171),
never an approval.

ADR-0186 makes Delivery's integration queue the existing boundary for M5 to
consider. A project declares `provider` or `local`; without a declared mode,
nothing runs. The local mode uses the owner's declared runner and a
fast-forward-only target move. Provider mode is intended to enqueue into the
project's queue and observe its position and ejection, but that provider path
is not implemented yet. A separate M5 integration executor would duplicate
integration authority. If M5 includes integration, the owner must decide in a
separate ADR whether that step consumes Delivery's declared queue/provider;
stage scheduling remains a different question. Missions has no unattended
execution today.

## Review and retention

Evidence is a reporter's observation, a confined file reference (regular,
at most 4 MiB, hashed through `os.Root`), or an existing Delivery artifact ID
in the same repository and candidate revision. A link never transfers Delivery
author authority. Evidence capture in a Git repository requires a clean
committed candidate.
Initializing Git in a previously plain folder binds the new repository identity
and invalidates evidence from the earlier scope. Every criterion needs
current-scope passing evidence. Pending review
checks scope, clean HEAD and current file/Delivery ID references again at accept.
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
inform Delivery artifact references; Paseo's explicit workspace boundary informs the
transfer preview. See the benchmark references in the approved plan. The
browser workspace overview reads a bounded summary of active missions and
links to the existing detail view; it creates no second mission editor or
shared presentation component between apps.
