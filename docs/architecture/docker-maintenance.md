# Docker maintenance and health (ADRs 0067/0068)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

Containers, Resources, Health and History stay within the App. List blocks can
include their own action footer; `ListItem.Wrap` preserves complete identifiers
and observations. Native controls and Zod validate App forms. Desktop/mobile
App links support `#/app/<id>/<path>`, including Inbox maintenance review links.

`GET /api/docker/resources` reads images, volumes, networks and references from
all containers. Stopped consumers prevent removal too. Image sizes are reported
sizes, not reclaimable totals; volume/network sizes are unknown. Only selected
unreferenced images and custom local networks support removal. Volumes, built-in
networks, ingress and non-local networks remain inspection-only.

`POST /api/docker/plans` persists a five-minute preview; `POST /api/docker/jobs`
requires confirmation and revalidates endpoint, full IDs, membership and stable
state fingerprints. Every target is reserved atomically in `docker_locks`, shared
with v1's individual operations. Jobs contain requester/approver provenance and
step outcomes. Project actions operate on existing containers in ID order with
unchanged members skipped; they do not implement Compose dependency order.
Independent failures yield partial results; procedures stop after a failed or
unknown step. Jobs last at most five minutes with 45-second step bounds. Startup
marks interrupted results unknown and never replays host mutations.

`GET /api/docker/plans/{id}` and `GET /api/docker/jobs[/{id}]` expose review and
results. `POST /api/docker/plans/{id}/review` atomically files one Inbox link per
plan. Inbox triage never executes a job; the owner confirms in Docker. Recognized
secret values and common credential patterns are masked from bounded literal
logs. Environment values stay transient; no raw logs enter persisted history.

`GET /api/docker/health?endpoint=…&project=…` reads the latest saved snapshot.
`POST /api/docker/health/check` samples on demand. `GET/POST /api/docker/monitors`
reads/updates explicit per-project settings with revision checks. A service-owned
collector observes at most four projects concurrently (32 enabled projects,
128 containers per project), at 30/60/300-second cadence. Disabling cancels its
in-flight sample; stale revisions cannot write. No background work runs for
unconfigured projects. Endpoint changes do not silently redirect a monitor.

Store keeps the latest snapshot and incident lifecycle, not a metrics time
series. Consecutive bad observations open one incident per signal; two good
observations resolve it. Unknown evidence and reconnect gaps break streaks and
never falsely resolve an incident. Retention is 7/30 days for closed/pending
records; unresolved incidents remain. `docker.monitor`, `docker.health`,
`docker.plan` and `docker.job` events are coupled to their Store transactions.

`POST /api/docker/diagnosis` separates observations and possible causes, with
named procedures for repeated restarts, failed health checks, stopped services
and high memory. No dependency topology or root cause is inferred from names.
Restart-loop previews allow advancing counters/timestamps while preserving the
exact identity and restarting precondition. Memory verification uses the reviewed
threshold. No automatic repair, backup/restore, blanket prune or remote Engine
access is enabled. API/package details: [Docker guide](../../docs-site/guide/docker.md).
