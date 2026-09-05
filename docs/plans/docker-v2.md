# Docker v2: project operations and Compose deployment

**Status:** project-operation foundation implemented with Docker v3
(ADR-0067), 2026-09-04. Exact previews, shared reservations, parent/step
results and shared tools are present. Compose registration/deployment below
remains proposed. Grouped inventory is implemented in ADR-0066.

## Product outcome

Manage an application as a project inside the Docker App: inspect its
services, start/stop/restart the group, and deploy a registered Compose
configuration with visible progress and a durable result. Keep the current
container detail and History views. The sidebar remains an app launcher.

| Surface | Proposed behavior |
|---|---|
| Project heading | State summary, expand/collapse, project action menu |
| Project detail | Services, recent operation, registered configuration if present |
| Start / Stop / Restart project | Preview affected containers, then show each result |
| Register Compose project | Select workspace, existing file(s), project name and profiles; validate before saving |
| Deploy / Update | Preview configuration and affected services, apply, verify running/healthy state |
| History | One parent operation with child results and readable partial failures |

Use the existing Apps host and its list/detail/form/action vocabulary. Adapt
the [Cursor/t3code/paseo benchmark](../benchmarks/2026-08-24-adopt-t3code-paseo-cursor.md):
progressive disclosure, immediate pending feedback, real operation stages and
one clear recovery action. Broader Apps changes must be justified by a
concrete workflow, not by building a second UI framework first.

## Identity and ownership

A **detected group** is the exact endpoint/project-label pair seen in Docker.
It supports inspection and operations on its existing containers. A
**registered project** also names the owning PiCode workspace, base folder,
ordered Compose files, explicit project name, profiles and environment-file
references. Deployment requires registration; labels alone do not establish
that PiCode has the original or complete deployment configuration.

Registration previews the resolved service names and connection. Existing
project labels that conflict with another registration must be reconciled
explicitly. Nothing is imported, overwritten or adopted silently. Bind mounts
and build contexts are shown in the preview; resolved secret values are not.

## Delivery sequence

### 1. Project actions

Extend the shared Docker service with a preview and a parent operation over
an exact snapshot of existing container IDs. Reserve all target locks before
starting; use the same endpoint/container locks as v1 so group and individual
actions cannot race. Start applies only to stopped/created containers;
stop/restart apply to running containers. Unchanged members are reported as
skipped. This phase does not claim Compose dependency ordering or recreate
missing services.

The confirmation names the project, connection, action and affected members.
The server rejects stale previews instead of silently changing the target
set. The parent records queued/running/succeeded/partial/failed/unknown, while
child operations retain v1's observed outcomes. Store methods append events
in the same transaction; the existing change feed updates the app. A browser
disconnect does not cancel accepted work.

### 2. Register and deploy Compose

Use the installed official Docker Compose plugin, invoked with a fixed
argument vector through `exec.CommandContext` and an explicit working
directory/endpoint. Do not interpret shell command text or implement a
second Compose engine. Check plugin capabilities and validate configuration
before offering Deploy. A missing plugin is one line plus a setup action.

This extends ADR-0065's choice to use the Docker CLI only for connection
discovery. An implementation ADR must record the Compose subprocess boundary,
supported plugin capabilities, file/credential handling, time bounds and
recovery before this phase changes that behavior. Existing Engine operations
continue to use the Go client; no new Go dependency is planned.

Deploy pins the registration revision, configuration digest, target endpoint
and project name. Revalidate immediately before execution. Run Compose's
apply operation with a bounded wait for running/healthy services. Progress
shows actual preparation, application and verification stages; partial output
cannot be treated as success. Configurations without health checks can only
establish running state, which must be stated in the result.

Keep credentials in the existing environment/files or Docker credential
store; registrations hold references. Do not persist expanded environment
values, resolved configuration or raw Compose output in history without
redaction. Do not promise automatic rollback of databases or volumes. A
future redeploy of an earlier revision needs retained configuration/image
identities and an explicit review of its data compatibility.

### 3. Shared sysadmin tools

Expose project inventory, preview, operations and registration through the
same service/API used by the app. The optional `pi-sysadmin` package submits
the accepted preview identity and checks the final parent/child results.
Pi confirmations follow the existing v1 convention; unavailable confirmation
returns the operation to the Docker App. Existing user/token/OS permissions
remain the trust model, not a separate per-agent privilege boundary.

## Acceptance matrix for implementation

These are required tests for the corresponding phase, not current passes.

| Conditions | Required result |
|---|---|
| Mixed running/stopped project; start requested | Start eligible members; show unchanged members |
| Project or container operation already owns any target | Reject before any child starts |
| Endpoint, membership or configuration changes after preview | Reject stale preview and offer refresh |
| Some children fail | Preserve successes and show partial result with exact failures |
| Daemon disconnects or PiCode restarts mid-operation | Reconcile where provable; otherwise unknown, never automatic replay |
| Detected group without registration | Inspect and operate existing members; offer Register for deployment |
| Compose unavailable or configuration invalid | Show blocked/error state and one next action; no deployment |
| Conflicting registration/project ownership | Explain conflict; require explicit reconciliation |
| Deploy completes but a service is unhealthy or times out | Failure/unknown as appropriate; no success badge |
| Credentials appear in configuration/output | Redact before any saved/displayed operation record |
| No health checks configured | Verify running state and identify the limit |
| Browser disconnects or retries an accepted request | Same idempotent operation, no duplicate work |

Images, networks and volumes can follow as resource inventory linked to these
projects. Container removal, volume deletion, global cleanup and remote
engines need separate scope and decisions; they are not prerequisites for
this v2. Ordinary deployment preserves volumes and never adds `down -v` or
global prune as an implicit step.

The proposed next stage is [Docker v3: visibility and supervised maintenance](docker-v3.md).
V2 remains the prerequisite for project operations and deployment.

## Sources

- [Compose project identity](https://docs.docker.com/compose/how-tos/project-name/)
- [Compose project/service labels](https://docs.docker.com/reference/compose-file/services/#labels)
- [Compose up and bounded readiness wait](https://docs.docker.com/reference/cli/docker/compose/up/)
- [Compose down and volume behavior](https://docs.docker.com/reference/cli/docker/compose/down/)
