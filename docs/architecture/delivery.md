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

Next: observation and desktop/mobile presentation follow
[the delivery plan](../plans/delivery-flow.md). No new polling or UI is introduced.
