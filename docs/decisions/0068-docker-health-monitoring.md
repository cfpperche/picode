# ADR-0068: Opt-in Docker health sampling and supervised diagnosis

- **Status**: accepted (owner authorized Docker v3 on 2026-09-04)
- **Date**: 2026-09-04
- **Extends**: ADR-0065, ADR-0067, ADR-0048

## Context

A running container is not proof that an application is healthy. Browser
refreshes cannot provide monitoring while the owner is away. Repeated threshold
crossings should update one incident instead of creating an alert flood.

## Decision

A service-owned collector runs only for explicitly enabled endpoint/project
configurations. Owners choose a bounded cadence, CPU/memory thresholds,
consecutive bad samples, and incident retention. Disabling cancels collection;
configuration revisions reject late writes. No repair is enabled by monitoring.

Store one latest snapshot per project and a deduplicated incident lifecycle,
not an unbounded metrics series. CPU is percent of one core; memory is usage
relative to the Engine's reported limit. Missing samples, unreachable engines,
stopped containers, and absent health checks remain distinct from healthy/zero.
Old timestamps remain visible. Consecutive bad samples open an incident; two
consecutive healthy observations resolve it. Unknown evidence breaks a streak
and never resolves an open incident. Retain unresolved incidents, and remove
closed/pending history older than the selected retention. All writes emit
events through Store; clients use the existing feed without an API timer.

On-demand diagnosis separates observed state from possible causes. Named
procedures cover restart loops, failed health checks, stopped services that may
be dependencies, and memory pressure. Every procedure is an ADR-0067 reviewable
plan with impact, exact targets, approval, and verification. A restart can
mitigate symptoms without establishing a root cause; report that limit.
Snapshots and incidents contain no raw logs or environment values.

## Decision table

| Conditions | Action / result | Required coverage |
|---|---|---|
| No monitor configured | No background sampling; manual check available | Service tests |
| Enabled monitor reaches its cadence | Bounded shared-service sample; store event | Collector tests |
| Monitor disabled while sample is in flight | Cancel and reject its stale revision | Store/collector tests |
| Running, no health check | Display no health check; do not assert application health | Health/app tests |
| Stopped, missing stats, unreachable, or stale | Distinct state; unavailable metrics are not zero | Health/app tests + visual QA |
| Repeated bad samples reach threshold | One open incident per signal | Store tests |
| Unknown observation or reconnect gap | Break pending streak; do not resolve open incident | Store tests |
| Two good observations | Resolve existing incident; subsequent regression starts a new incident | Store tests |
| Incident outside retention | Remove closed/pending history; retain unresolved incidents | Store tests |
| Diagnosis finds risky condition | Observations, hypotheses, optional reviewed procedure; no automatic mutation | Service/app tests |

## Consequences

Monitoring works with the browser closed and stays opt-in. Persisted snapshots
are intentionally small; historical charts, backups, remote engines, and
automatic repair policies are future decisions. Health checks are defined by
the application; PiCode does not invent them or infer dependency topology from
container names.

## Alternatives considered

- Browser polling: stops when the owner leaves and duplicates collectors.
- Metrics time-series service: adds operations and retention complexity before
  the app needs historical charts.
- Repair on first threshold breach: treats a symptom as a confirmed cause.
