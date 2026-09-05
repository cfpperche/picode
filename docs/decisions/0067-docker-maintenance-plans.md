# ADR-0067: Reviewed Docker maintenance plans and project operations

- **Status**: accepted (owner authorized Docker v3 on 2026-09-04)
- **Date**: 2026-09-04
- **Extends**: ADR-0065, ADR-0066, ADR-0048

## Context

Docker v3 needs resource inspection, selected cleanup, and supervised repair.
The current service only locks one container. The project-operation foundation
from the v2 proposal is therefore part of this delivery. Compose file
registration and deployment remain a separate increment.

The [Cursor/t3code/paseo study](../benchmarks/2026-08-24-adopt-t3code-paseo-cursor.md)
informs compact project rows, a visible review state, and explicit per-step
outcomes. Everything stays inside the Docker App's main pane.

App list action footers, optional wrapped rows and parent-tab selection reuse
the four primitives. Shared App paths can open an exact review from Inbox on
desktop or phone; navigation and reload preserve that path.

## Decision

Use the same Engine API and service for the App and optional Pi tools. Add
images, volumes, and networks with references from **all** containers, including
stopped ones. Size is the Engine's reported image size, never an estimate of
reclaimable space. Unknown volume/network size stays unknown. Volumes are
inspection-only in this increment. Built-in, ingress, and non-local networks
cannot be removed. Image removal uses full IDs, no force, and no parent prune.

Persist immutable five-minute previews with exact endpoint, resource IDs,
container membership, state fingerprints, impact, and requester provenance.
Execution requires explicit confirmation, revalidates the preview, reserves
every target atomically, and returns a durable parent job with step results.
Individual operations share the same lock table. Revalidate each target again
before its mutation; Docker's non-force conflict checks remain authoritative
against changes made outside PiCode. Confirmation is interaction policy under
the existing authentication model, not a per-agent security boundary.

Restart-loop procedures preserve exact identity and the restarting precondition
while allowing the counter/start timestamp to advance during review. Otherwise
an active loop would make every review stale. Other lifecycle actions retain
strict state fingerprints. Memory verification uses the reviewed threshold.

Project start/stop/restart operates on existing containers, reports unchanged
members as skipped, and makes no claim about Compose dependency order.
Independent project steps may finish after a sibling fails. Supervised
procedures stop after a failed or indeterminate step and verify their result.
Timeout/shutdown outcomes remain unknown and are never replayed automatically.
Persist requester and approver separately; no logs or environment values enter
plans, jobs, incidents, or Inbox. When Pi cannot show confirmation, it can file
a deduplicated Inbox review link. Opening that link shows the saved preview;
marking the Inbox item done never executes a Docker action.

Logs remain bounded, literal, and untrusted. Redact known sensitive environment
values and common labeled credentials, bearer tokens, and URL credentials
before returning them. This is best-effort masking, not a claim that arbitrary
unlabeled secrets can be recognized. Environment values never leave inspect.

## Decision table

| Conditions | Action / result | Required coverage |
|---|---|---|
| Image/volume/network shared, or referenced only by a stopped container | Show all consumers; reject removal | Resource tests |
| Built-in/ingress/non-local network, or any volume | Inspect only | Resource tests |
| Eligible image/custom network selected | Preview exact ID, confirm, recheck, non-force delete, verify absence | Client/service + visual QA |
| Reference appears after preview | Reject before delete | Service tests |
| Restart loop advances counter/time but retains exact identity and restarting state | Permit reviewed stop; recheck loop precondition | Procedure tests |
| Preview expired, endpoint/member/state changed | Reject; create a fresh preview | Service tests |
| Any target reserved by individual/project/procedure operation | Reject whole job; no partial reservation or execution | Store/service tests |
| Same request key and plan | Return same job; no replay | Store/service tests |
| Key reused for another plan, or plan already executed | Conflict | Store tests |
| Project member already in requested state | Skipped child, visible in preview/result | Service tests |
| Independent step fails | Record failure and partial parent result | Service tests |
| Procedure step fails or shutdown interrupts work | Stop remaining steps; record failure/unknown; no replay | Service/recovery tests |
| Approval declined or unavailable | No mutation; optional Inbox review link | Package/service tests |
| Secret or instruction-shaped text in logs | Mask recognized secrets; render literal bounded evidence | Client/package/visual tests |

## Consequences

Users and agents share reviewed maintenance without a new management server,
shell execution, dependency, blanket prune, or automatic repair. We own the
preview/lock/job contract. External Docker clients can still change the engine
between observations; uncertain outcomes must stay visible.

## Alternatives considered

- Broad prune: does not name the affected resources in the owner's review.
- Generated shell runbooks: cannot preserve exact targets and shared locks.
- Full Compose deployment first: independent of inspecting and maintaining
  existing containers; remains in the v2 deployment proposal.

API source: [Docker Engine API v1.44](https://docs.docker.com/reference/api/engine/version/v1.44/).
