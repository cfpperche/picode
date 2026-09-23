# Delivery declarations (ADR-0171)

`cmd/picode/delivery.go` and `internal/mcptool/delivery.go` use the same request
adapter. It replaces argument identities with the inherited launch identity and
posts to `internal/server/delivery.go`. The existing daemon auth gate applies.
This is same-user launch attribution, not a sandbox or native-session credential.

Three faces reach that one route, and every one of them offers the same actions
and parameters. The principal is the **agent** (ADR-0160): a workspace instance
of any launchable CLI is an agent, `launchIdentityEnv` puts `PICODE_AGENT_ID`
into the terminal it runs in, and `grant.FromIDs` resolves agent-wins — a bare
`PICODE_TERM_ID` names only a terminal that is not an agent. What differs
between agents is how the tool arrives, not who they are:

- `picode delivery <action>`, in the agent's terminal;
- `picode mcp delivery`, injected as the family `picode-delivery` when the
  agent's CLI takes MCP servers at launch — Claude Code, Codex, OpenCode
  (`internal/server/cli_tools.go`, ADR-0154) — or reached through that CLI's own
  Connectors scope (ADR-0150). The launch writes the identity the server
  resolves the principal from into that CLI's own server config where the CLI
  does not pass its environment on ([picode-mcp](picode-mcp.md));
- the `delivery` tool from `packages/pi-delivery` when the agent's CLI is Pi
  (Pi packages stay Pi-only, ADR-0091), which derives a mutation's retry key
  from the session, the action and the payload so a repeated call replays.

`internal/mcptool/delivery_test.go` holds the pi package against the MCP schema,
and `internal/server/cli_tools_test.go` holds the form's list
(`PICODE_TOOL_FAMILIES`) against `mcptool.FamilyNames()`. The catalog carried
delivery from ADR-0171 while the form offered four families, so a person could
not switch it on for an agent whose launch settings had been customized; a
launch that never touched them already received every family.

The daemon resolves the registered agent's launch folder (or terminal's stored
folder), then `gitgraph.Key` identifies the shared repository across worktrees.
The shell's current directory cannot redirect a request. Both supplied identities
must agree; a bound terminal resolves to its agent. The handler uses fixed Git
argument lists with deadlines, never a shell or project command.

| Action | Result |
|---|---|
| capabilities | Schema version, launch identity, supported actions; queue/deploy false |
| register | Check local branch/revision/target; create stable ID at version 1 |
| update | Author + expected version; replace declaration and clear review |
| request-review | Author + expected version + current source check; record intent |
| withdraw-review | Author + expected version; clear intent even if Git source moved |
| show | Declaration plus timestamped source observation; other evidence unknown |
| list | Up to 100 declarations; stable creation cursor; evidence not evaluated |

`internal/store/delivery.go` owns both tables from migration 063. Mutations
serialize, check optimistic versions and append `delivery.changed` in the same
transaction as the row and request receipt. Exact retries return the original
receipt, even after later updates; call show to read current state. Errors do not
create receipts. Retry keys are scoped by repository and principal.

Readers in the same repository may inspect declarations from other principals;
only the original principal can mutate them. Removed principals leave history
but cannot use the endpoint. Caps and the absence of pruning/reassignment are
specified in ADR-0171. Git movement between check and commit remains possible;
review requested never means the current branch was approved or validated.

Tests: store delivery decision table, concurrent writers/rollback/pagination;
server principal and refusal tables; command-through-daemon test; shared MCP
identity and transport failure tests; mutation event invariant. Vendor processes
and real deployments are outside this automated evidence.

D1a introduced declarations only. D1b adds the observation and desktop/mobile
presentation described below; publication remains the next separate slice in
[the delivery plan](../plans/delivery-flow.md).

## The integration queue (ADR-0182)

`internal/store/delivery_queue.go` owns one table from migration 066, built with
the mechanics the declarations proved: the row's truth lives in its JSON body,
`delivery_queue_requests` receipts compare byte-for-byte so a retry replays
instead of writing twice, the row and the `delivery.changed` event commit in the
same transaction, and the delivery mutex serializes both stores because they feed
one view. States are `waiting → authorized → running → done | failed`, plus
`withdrawn`; `order` and `authorize` are refused for every actor but
`store.OwnerActor`, so the owner's authority holds even where a door forgets to
check it, and an agent withdraws only its own entry. One active entry per
delivery is an invariant, and the entry names a declaration of its repository.

Eligibility is deliberately **not** stored: the branch still pointing at the
reviewed revision, the target not having moved and the evidence still covering
that revision are derived from Git and receipts when an entry is read or run, so
a stale approval can never ride along inside the row.

The owner's doors are `POST /api/{workspaces|agents|terminals}/{id}/delivery/queue`
(act as `store.OwnerActor`; the store refuses `order` and `authorize` for anyone
else) and `GET|PUT|DELETE /api/delivery/integration`, the **declaration** of how a
project integrates — `ffOnly` plus up to eight single-line commands, written per
workspace with the machine as the fallback layer and a built-in default
(`ffOnly`, no checks) when neither declares. A PUT with `expectedVersion`
answers 409 when the layer moved since it was read; DELETE drops a
*workspace's* layer (it must name one) so it inherits again; removing the
workspace drops it too. The owner edits it in the workspace card's Settings…,
and the machine layer in Preferences → Landing work, which reads every layer
at once through `GET /api/delivery/integrations` (`machine`, null when
undeclared, and `workspaces` keyed by id) to list who follows what. The Delivery read carries both: its
payload gains `queue` and the already-resolved `integration`, so a surface never
has to repeat the fallback. The agent's half rides the delivery tool contract: `request-integration` asks for
a place for the launch's **own** delivery, naming the revision and target the
delivery declares — the store checks both, so a drifted or unreviewed revision
cannot be queued — and `withdraw-integration` takes it back by the entry's own id
and version, which `show` prints. All three faces (the `picode delivery` command,
the MCP family and the pi package) offer the same actions and the same fields,
and a test compares the enum of one with the action list of the other. The
serialized executor and the Delivery-view lane are the slice's remaining steps
(`docs/plans/delivery-flow.md`, D3).

### Modes and vocabulary (ADR-0186)

An integration is performed by **the project's provider when it has one**. The
declaration names its mode, `provider` or `local`, and a project that declares
nothing is not executed at all — absence stays a named blocker. Under `provider`,
PiCode enqueues through the provider's own queue: the repository's GitHub merge
queue, or a reviewer's `r+` where the project runs bors. It then observes that
queue — the entry is present or not, its position, and its **ejection with the
provider's reason** — and never merges by itself; the provider's CI validates the
merge result. `order` belongs to the local mode, and under `provider` it is
refused with that reason, because the provider owns the order. A provider that
cannot be reached is unknown, never inferred.

The `local` mode is the fallback for a repository with no hosting provider, and
is the runner described below: the declared single-line commands, then a
fast-forward-only move.

The declaration's `mode` is implemented end to end for what does not need a
vendor: an undeclared mode runs nothing (the runner records `failed` with
`not run: the project declares no integration mode…`), `local` runs the runner,
and `provider` is refused by the runner with `not run: this project integrates
through its own provider…` while the owner's `order` is refused with "the
provider owns the queue's order". Provider mode carries no commands — its
provider runs those checks — and the declaration refuses the combination. What
is still to build is the provider path itself: enqueueing through the project's
queue and reading its position and ejections.

The surface speaks the community's vocabulary — **merge queue**, entry,
position, *approved*, *integrating*, *integrated*, *ejected with a reason* — so
anyone who knows GitHub or bors can read the screen without a manual. Internal
store field names keep their own words; the mapping is deliberate.

### The runner (ADR-0182, the local mode)

Authorization is execution authority: when the owner authorizes an entry, that
repository drains — the entry just authorized, then whatever else is authorized
behind it, in the order the owner set. One repository runs **one operation at a
time**: the store refuses `start` while another entry of the same repository is
running, and the daemon holds a live lock per repository as well, so the promise
does not depend on a single layer.

`internal/delivery.RunEntry` re-reads what the entry was authorized against
before claiming it, and every way an authorization can go stale is a **named
blocker** recorded on the entry: the project declares no integration rules, the
declaration does not ask for fast-forward-only integration, the branch no longer
points at the reviewed revision, the target is gone or has moved past a
fast-forward, or the recorded evidence says the checks failed. A blocker is
recorded as `failed` with `not run: <reason>` — it never moves a branch.

The checks are the **declared** commands, run through `/bin/sh` in the entry's
repository with a 30-minute bound each; a project's declaration is the owner's
own text, so an agent cannot put a command there. Then the target moves to the
reviewed revision fast-forward only: a branch a worktree holds is fast-forwarded
in that worktree (the merge refuses a dirty tree and refuses anything that is
not a fast-forward by itself), and a branch no worktree holds moves by
compare-and-swap, so a target that changed under the run is a refusal instead of
a lost commit. The outcome lands on the entry as `done` with what it did, or
`failed` with the command that failed and its last line.

A daemon that stops while an entry runs leaves it `running` with a note saying
the outcome is unknown; nothing is retried and no success is inferred. The
owner's withdraw is the way out of that state (and of any running entry), which
is what clears a repository whose operation died with the process.

## The Mission link (ADR-0199)

A mission cites a delivery as evidence — `kind: "delivery"`, the delivery id —
and that is the only direction the reference exists in: the mission owns it, and
**a link never transfers integration authority**. The Delivery surfaces read it
backwards so a change can say which objective it serves: `MissionsByDelivery`
inverts the evidence link per workspace (each mission named once, however many
of its criteria point at the same change, paged to a bound above the store's own
mission cap and reporting `truncated` instead of passing a short answer off as a
complete one), the Delivery read carries the map as `missions`, and a row or a
detail shows the mission's own title, its own state label and a route back to it.
A change nobody cites says nothing — not "no mission" — because absence here is
simply absence of a link.

The two axes stay separate: a mission is an objective with criteria, evidence and
the owner's acceptance; a delivery is an artifact with a revision, receipts and
its place in the merge queue. `review` means different things in each — in
Missions it is the owner accepting a result, in Delivery a *review requested* is
an agent's declaration that a human should look (ADR-0171). Missions states the
same boundary in its own words (`docs/architecture/missions.md`, "Where Delivery
begins"), including that its assignment receipt is a receipt and never a
"delivery"; the rule stands for anything written later.

## Integration observation (ADR-0170, D1b)

`internal/delivery` reads local refs, explicit target ancestry, checkout state and
bounded producer receipts. Owner-scoped `GET /api/{workspaces|agents|terminals}/{id}/delivery`
uses the same independently resolved root precondition as Git/files. The optional
`target` names a local branch; only an existing main is the default. `root` is an
equality condition, never a path lookup. The command's `show` also consumes this
observer; publication remains unknown. Registration never grants review approval.

The server coalesces reads and caches them for five seconds per repo/root/target,
with at most 64 active keys. Manual `fresh=1` bypasses completed cache entries.
Collection has a ten-second deadline, 32 checkout status probes, 1,000 changes and
1,000 receipt files; truncation/errors are explicit issues, not empty success.
Refs are compared before/after collection, with one retry then unknown facts.
Current occupancy supplies associated agent names, not authorship/session links.

`ci.sh`, `ci-scoped.sh` and `land.mjs` use `scripts/delivery-receipts.mjs` to
atomically publish started/finished facts under the common Git directory. Failures
to write warn without changing the original command result. A land followed by
failed CI stays integrated. A start without a terminal record is unknown, never
assumed active or retried. Receipt parsers reject unsupported versions, malformed
or oversized data, unsafe POSIX permissions, symlinks, repository mismatch and invalid
times. Git status probes disable fsmonitor hooks and optional index locking.

No records are deleted automatically. Existing logs and gate stamps remain intact.
Historic attempts without receipts stay unknown. Covered-content reuse mirrors
ADR-0124 but is labeled scoped historical evidence; it never asserts current
main CI or approval. Dirty or changed command inputs cannot establish a pass.
Local file evidence is not an authorization or tamper-proof audit boundary.

The browser adds History/Delivery within the existing Git tab; mobile owns its
Delivery section and detail navigation. Both share pure labels and a headless
observer, not UI components. The browser's Delivery view draws a **page frame**
inside that tab — `.settings-wrap` + `.settings-head` + `.settings-card`, the
Agent CLIs geometry of ADR-0103 (owner, 2026-09-21; the rule lives in
`docs/benchmarks.md` § One page width, the guard in
`web/tools/delivery-surface.test.mjs`) — because a list of changes with a
detail is read top to bottom like a route. History stays a canvas, and the
tab's own strip (workspace picker plus the History/Delivery toggle) stays above
both and outside the card, so the graph never moves. The view is the page's
only scroll container. Entry, focus, feed events and visible 15-second
reconciliation refresh facts because Git/receipt writers do not always emit
feed events. Hidden views stop polling; errors retain the last observation, roots
pin after the first read, and disposed requests cannot change the next project.
Views identify stale observations after 30 seconds. No deployment lane is shown
until D2 exists, and no land/deploy button is introduced.

On Windows, native ACL inheritance applies; POSIX mode-bit checks are skipped.
The Linux scratch checks do not establish Windows ACL or physical UI acceptance.

### Shared Git workspace selection

The desktop Git surface owns one workspace selector before the History and
Delivery tabs. Both views use that same owner and eligible repository list;
History's inner toolbar contains only its own filters and actions. Selecting a
workspace preserves the current History/Delivery view while the existing Git
picker resolves the destination repository and tab (ADR-0022).

| Workspace pick | Result | Verification |
|---|---|---|
| Sibling worktree, new repository or already-open repository | Existing tab resolution applies; retain History/Delivery | `workspacePicker.test.js` resolution table; scratch browser checks for both views |
| Failed lookup or no repository key | Keep the current owner and view; show the error | `workspacePicker.test.js` failure rows; scratch failed-lookup check |

Visual checks cover the shared selector in both views, its open and no-results
menu, and Delivery's empty and missing-target states.
