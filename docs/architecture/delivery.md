# Delivery declarations (ADR-0171)

`cmd/picode/delivery.go` and `internal/mcptool/delivery.go` use the same request
adapter. It replaces argument identities with the inherited launch identity and
posts to `internal/server/delivery.go`. The existing daemon auth gate applies.
This is same-user launch attribution, not a sandbox or native-session credential.

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
