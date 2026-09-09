# Docker App and sysadmin tools (ADR-0065)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

The first-party Docker App and optional `packages/pi-sysadmin` tools call the
same `internal/docker.Service`. The service uses the Docker Engine API v1.44
over a local Unix socket through Go's HTTP client. Endpoint resolution follows
`PICODE_DOCKER_HOST`, then `DOCKER_CONTEXT`, `DOCKER_HOST`, and the selected
Docker context. Without the CLI, the default is `/var/run/docker.sock`.
No Docker credentials, daemon settings, or OS privileges are changed.

`GET /api/docker/containers`, `GET /api/docker/containers/{id}`, and
`GET /api/docker/operations[/{id}]` expose inventory, a resource/log sample,
and history. `POST /api/docker/operations` accepts start/stop/restart, a full
container ID, and an idempotency key. App actions call the same service.
The existing authentication gate applies. Reported agent identity provides
provenance only; agents keep the existing user's permissions, not a new sandbox.

The store owns `docker_operations` and appends `docker.operation` events in
the same transaction. One job per endpoint/container can run at a time. Jobs
have a 45-second bound, survive browser disconnects, verify their postcondition,
and record `succeeded`, `failed`, or `unknown`. Startup marks interrupted jobs
unknown without replay. Docker container events flow into the existing SSE
feed as ephemeral `docker.changed`. The browser never polls the Docker API.

The Apps vocabulary retains its four block types. Optional `Block.Text`
renders plain text, `Block.Empty` names an empty list, and busy metadata adds
motion to pending jobs. The host prevents repeated clicks while submitting.
The phone's More → Apps grid opens the shared AppSurface at `#/app/<id>`;
Inbox keeps its specialized route. Public instructions live in the
[Docker guide](../docs-site/guide/docker.md).

Docker inventory groups containers by their exact Compose project label
(ADR-0066); unlabeled containers appear last under Standalone containers.
Named list blocks opt into native disclosure with stable `id` and
`collapsible` fields. Group cards fill the available app canvas width,
respecting its padding on desktop and phone. Counts summarize running/stopped
and other actual states. The host stores each app/endpoint/project fold in
browser preferences; new groups start closed. Search reveals matching groups and clearing it
restores the saved folds. Presentation changes do not mutate the store or
Docker. The project-operation foundation from the [v2 plan](plans/docker-v2.md)
is implemented with v3; Compose registration/deployment remains separate work.
