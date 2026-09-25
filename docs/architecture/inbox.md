# Inbox (ADR-0037, amended by ADR-0208)

Inbox is PiCode's durable human mailbox. `internal/store/inbox.go` owns items, states and events; `internal/server/inbox.go` owns creation, list, answer and triage HTTP routes. Agent and terminal delivery paths use the same item identity and preserve visible failure when a reply cannot reach its source.

## Read and action surfaces

| Concern | Owner |
|---|---|
| Primitive-tree views and action mapping | `internal/inboxview` |
| Browser and mobile rendering | Each app's `AppSurface`, pointed at `/api/inbox` |
| Badge | `GET /api/inbox/badge`, with blocking count and a dot for other open activity |
| Navigation | Desktop sidebar button → `#/inbox[/<id>]`, with Back in the page header → `#/`; mobile Inbox tab uses the same hash shape |
| Compatibility | Retired 2026-09-25: `#/app/inbox[/<path>]` no longer redirects and `/api/apps/inbox/{view,action}` is no longer an alias |

Inbox is not in `GET /api/apps`. The desktop Apps grid and its badge count only registered apps. The desktop Inbox button fetches its badge independently and follows `inbox.*` feed events. The phone derives its numeric badge from its blocking Inbox read. The common primitives renderer is a presentation dependency, not app registration.

No persistence migration occurs: all old rows, item IDs, answer channels and event types remain valid. A saved `x:inbox` app tab is removed during desktop tab restoration; direct item hashes still open the item. An item with no reply channel remains open or records the answer according to the existing delivery contract in ADR-0037 and ADR-0154.
