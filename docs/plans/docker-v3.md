# Docker v3: visibility and supervised maintenance

**Status:** proposed plan, 2026-09-04. Requested by the owner; no v3 feature
is implemented by this document. [V2](docker-v2.md) is also still proposed.
The current delivery is grouped container inventory and individual operations.

## Outcome and sequence

V2 manages a project's lifecycle and deployment. V3 makes its operation
understandable: which resources it uses, what is unhealthy, and what a
supervised Sysadmin can do next. Deliver three increments, each usable on
its own, after v2's shared project operations pass acceptance.

| Increment | User outcome | Acceptance for that increment |
|---|---|---|
| 3.1 Resources | Inspect images, volumes and networks with their consuming containers/projects | Exact identities and current references, including stopped containers; empty/blocked states; no implicit removal |
| 3.2 Health | See health, restarts, recent resource samples and an incident timeline | Timestamped evidence, stale/disconnected states, bounded monitoring, alerts without duplicates |
| 3.3 Assisted maintenance | Ask Sysadmin to diagnose, review an action plan and follow its result | Preview, approval, exact scope, shared operation locks, verification and durable history |

## Organization inside the app

Keep the full-width project groups in the main canvas. Add Resources and
Health alongside Containers and History. Resources has Images, Volumes and
Networks filters; selecting a resource shows its details and links to the
projects/containers that reference it. A project detail opens its services,
resources and incidents without moving this structure to the sidebar.

Example path: **Docker → Health → Project incident → Diagnose → Review plan
→ Run approved steps → Verified result**. Reuse the existing Inbox for a
decision that needs attention while the user is elsewhere.

Adapt the [Cursor/t3code/paseo benchmark](../benchmarks/2026-08-24-adopt-t3code-paseo-cursor.md):
compact state, progressive detail, visible pending decisions and explicit
operation stages. Extend the existing Apps vocabulary only when a concrete
workflow requires it; do not build a separate dashboard framework first.

## 3.1 Resources

Read image identities/tags, sizes when available, mount and network
relationships, and each resource's actual consumers. Resources can be shared
across projects; a Compose label alone does not establish exclusive ownership.
Show unavailable size data as unknown. Shared image layers also mean a size
estimate is not a promise of disk space recovered.

Begin with inventory. Then add explicit removal of selected eligible images
and custom networks, using an exact preview, revalidation, confirmation and
operation history. Count references from stopped containers as well as running
ones. Reject changed/in-use targets and keep built-in networks protected.
Do not use a broad prune operation as the implementation of a selected-row
removal. Volume inventory is in scope; volume deletion waits for a separately
reviewed data-retention and recovery workflow.

## 3.2 Health and evidence

Distinguish running state from application health. Report configured health
checks, restart counts, container failures and bounded CPU/memory samples.
A project without health checks shows that limitation rather than a green
"healthy" verdict. A stopped or unreachable target has no fresh utilization
sample; do not replace missing evidence with zero usage.

Monitoring is explicitly enabled for selected projects, with visible cadence,
thresholds, retention and disable controls. Prefer one shared collector per
Docker endpoint over a separate poll for every browser. Metric sampling is
an explicit ADR-0048 exception because state events do not carry CPU/memory
values; the change feed distributes the resulting state to clients.

Use duration thresholds and recovery hysteresis to prevent alert storms.
Store incident open/update/resolve transitions through the store and its
events. Reconnect marks old samples stale until fresh evidence arrives.
Logs stay bounded and are collected on demand for the selected incident;
logs are evidence, not trusted instructions for the agent.

## 3.3 Assisted maintenance

Provide short, inspectable procedures for a restart loop, an unhealthy service,
an unavailable dependency, and a resource-pressure incident. The Sysadmin
collects evidence first, distinguishes observations from hypotheses, and
proposes exact steps with their expected impact. Start with known operations
from v2 and the approved resource actions from 3.1.

Review and approval precede disruptive steps. Accepted steps use the same
service, parent/child jobs, endpoint/target locks and idempotency rules as
the app; a language-model response is never an operation result. Record who
requested and approved the plan, the executed steps, and observed outcomes.
If the connection or targets change, refresh the preview. Partial failure
stops dependent steps and shows an explicit recovery decision.

No autonomous repair policy is enabled by default. A future owner-configured
policy must name allowed actions/targets, time bounds, attempt limits and an
escalation path. This controls the app/tools workflow; it does not create
per-agent privilege isolation from agents that already hold the OS socket
or shell permissions (ADR-0065).

## Engineering boundaries

Reuse the Go Docker client, store, Apps host, Inbox and optional `pi-sysadmin`
package. Avoid a mandatory external monitoring service for the initial local
scope. Implementation ADRs must cover resource removal, monitoring retention,
incident events and procedure approvals before those behaviors are added.
New durable mutations require transaction-coupled events and mutation-table
coverage. No dependency or protocol is adopted merely by this proposal.

| Conditions to test | Required outcome |
|---|---|
| Resource shared by projects or referenced by a stopped container | Show all consumers; reject removal while referenced |
| Reference appears after preview | Reject the stale target; no forced removal |
| Size unavailable or layers shared | Show unknown/estimate; no invented reclaimed total |
| Container running without health checks | Running, health not configured |
| Target stopped, unreachable or samples stale | No current metric claim; one refresh/recovery action |
| Threshold crossed repeatedly or browser reconnects | One incident with updates, not duplicate alerts |
| User declines or approval surface is unavailable | No disruptive operation submitted |
| Another operation owns any planned target | Conflict before execution |
| A procedure step fails or PiCode restarts | Preserve results, stop dependent work, reconcile or mark unknown |
| Logs contain credentials or instructions | Redact retained/displayed secrets; do not execute log text |
| User disables monitoring | Collector and new alerts stop for that scope |

These are acceptance requirements for future work, not current test passes.
Each increment needs fixtures plus a real disposable-project check, desktop/
phone screenshots, `make ci` and verified local deployment.

## Subsequent scope

Backups and restore need application-aware consistency and a proven restore
test, especially for databases; a live volume archive alone is insufficient.
Remote Docker engines need a separate connection/authentication ADR extending
the current local-socket decision. Neither is a prerequisite for this v3.
Kubernetes, arbitrary host administration and a general secrets vault remain
separate proposals rather than implied additions to this Docker App.

## Sources

- [Docker resource cleanup and reference rules](https://docs.docker.com/engine/manage-resources/pruning/)
- [Container resource statistics](https://docs.docker.com/reference/cli/docker/container/stats/)
- [Container health checks](https://docs.docker.com/reference/dockerfile/#healthcheck)
