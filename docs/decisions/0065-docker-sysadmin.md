# ADR-0065: Docker operations shared by an App and a Pi sysadmin package

- **Status**: accepted (owner authorized this implementation on 2026-09-04)
- **Date**: 2026-09-04
- **Extends**: ADR-0036 (Apps), ADR-0010 (Pi packages), ADR-0048 (events)

## Context

The owner wants PiCode to manage Docker through its interface and a sysadmin
agent. Both callers need the same operations, observable results, and history.
The existing Apps host already supplies lists, details, actions, and responsive
confirmation dialogs. It lacks plain-text logs and pending-action feedback.

The [Cursor/t3code/paseo study](../benchmarks/2026-08-24-adopt-t3code-paseo-cursor.md)
informs the adaptation: compact activity rows, explicit pending states, and
structured runtime data. Docker supplies an HTTP API; terminal output and
model prose are not the source of container state.

## Decision

Ship a first-party Docker App and an optional `pi-sysadmin` package, both using
one Go service. The first delivery supports local Unix-socket Docker Engines:
inventory, selected-container details, resource samples, bounded recent logs,
start, stop, restart, and durable operation history. Compose deployment,
container creation/deletion, image/volume cleanup, and remote engines are later
deliveries, not permanent refusals.

- Reuse Docker Engine API v1.44 through Go's standard HTTP client over a Unix
  socket. Verify server version compatibility. Discover the endpoint from
  `PICODE_DOCKER_HOST`, then Docker's context/environment rules; use the Docker
  CLI only to resolve a context. Never invoke a shell, elevate privileges, or
  change Docker configuration. Each operation pins its endpoint and full ID.
- The existing paired-device/install-token gate and operating-system user
  permissions apply. This adds no sandbox or per-agent security boundary.
  Agent identity in the audit log is provenance, not authentication. Raw
  Docker access under the same OS account can bypass this interface.
- Accept only named operations and full container IDs. Validate current state
  before recording a job. Stop/restart are confirmed in the UI; the Pi package
  uses Pi's confirmation dialog and refuses disruptive actions when no dialog
  is available. A confirmation is interaction policy, not an API privilege.
- Record jobs and their events in the same SQLite transaction. Idempotency
  keys prevent duplicate requests; one running job per endpoint/container
  prevents conflicting actions. Jobs survive browser disconnects, have a time
  limit, and verify the resulting Docker state. Interrupted or indeterminate
  outcomes are recorded as unknown and are never automatically replayed.
- Forward Docker container events to the existing change feed while Docker
  has been used. Reconnect with backoff. Resource values and logs are explicitly
  timestamped samples refreshed on request, not a new browser polling loop.
- Extend existing primitive fields for plain text, list emptiness, and busy
  state. Keep the four block types and API version 1. The embedded client
  ships with the server; no third-party code or iframe is introduced.
- The package is opt-in through the existing Packages surface. It uses the
  authenticated Docker API, returns structured data, and documents how to
  configure an agent as a sysadmin. Installing tools never starts an agent.

## Decision table

| Conditions | Action / result | Coverage |
|---|---|---|
| Endpoint absent, inaccessible, remote, or API incompatible | One blocked state with a setup/retry action; no mutation | Client/app tests |
| Engine reachable, no containers | Empty inventory with refresh/setup action | App tests + visual QA |
| Invalid ID, verb, key, or unknown agent | Reject before issuing a Docker mutation | Service/handler tests |
| Previously accepted key, same request | Return the recorded job; no second execution | Service tests |
| Same key, different request | Conflict; preserve the first job | Store/service tests |
| Another job is running for the same endpoint/container | Conflict; different containers remain independent | Store/service tests |
| Start on created/exited, stop/restart on running | Record running, execute, verify, record result | Service tests + disposable-container QA |
| Action incompatible with current state | Reject; leave Docker untouched | Service tests |
| Daemon returns a definite error | Record failure and expose the reason | Service tests + visual QA |
| Timeout, shutdown, or unverified postcondition | Record unknown; refresh Docker before retrying | Service/recovery tests |
| Pi confirmation declined or unavailable | No disruptive API request | Package tests |
| Logs include control characters, markup, or exceed the bound | Plain-text rendering, bounded output, truncation indication | Client/JS tests + visual QA |

## Consequences

Users and agents gain the same inspectable operations without operating an
additional management server. Docker remains authoritative; SQLite holds only
PiCode operation history. No new Go or browser runtime dependency is required.

We own API compatibility, job lifecycle, tests, and UI behavior. Completion
means the requested container state was observed, not application health or a
guaranteed rollback. Logs can contain application secrets and are visible only
through the existing authenticated surface; they are not copied into history.

## Alternatives considered

- **Portainer integration:** useful for an existing Portainer installation,
  but adds another service and management model for this first local feature.
- **Dockge integration:** useful for Compose-oriented deployments; the first
  delivery needs individual-container inspection and operations as well.
- **Docker Go SDK:** remains an option as coverage grows. The small, pinned
  API subset currently needs no external dependency.
- **Shell commands generated by an agent:** cannot provide the shared operation
  contract, idempotency, or durable history required here.

Sources: [Docker API](https://docs.docker.com/reference/api/engine/version/v1.44/),
[context precedence](https://docs.docker.com/reference/cli/docker/),
[daemon security](https://docs.docker.com/engine/security/).
