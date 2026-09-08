# Docker and sysadmin tools

Open **Apps → Docker** on desktop or **More → Apps → Docker** on a phone.
The app uses the Docker connection available to the PiCode server's user.

## Connect Docker

Install and start [Docker Engine](https://docs.docker.com/engine/install/)
or [Docker Desktop](https://docs.docker.com/desktop/). PiCode supports local
Unix sockets and Docker API 1.44. Docker 25 or later is suitable when its
supported API range includes 1.44.

PiCode checks `PICODE_DOCKER_HOST` first, then the selected Docker context and
environment. For example, `PICODE_DOCKER_HOST=unix:///var/run/docker.sock`
selects that socket explicitly. Use the socket for your Docker installation;
Desktop and rootless Docker can use different paths. For an installed service,
set this through `picode install --env PICODE_DOCKER_HOST=unix:///path/to/docker.sock`.
See [Docker context rules](https://docs.docker.com/reference/cli/docker/).

PiCode never starts Docker or changes socket permissions. If the app reports
an unavailable connection, check Docker and the server user's access, then
choose **Check again**. Remote Docker engines are not supported in this version.

## Inspect and operate

Containers appear in project groups inside the app. Select a heading to
expand or collapse it; its count and state summary remain visible. Groups
start closed and remember your choice in this browser. Containers without a
Compose project appear under **Standalone containers**.

Filter by project name, container name, image or state. Matching groups open
while searching; clear the filter to restore your saved view. Project names
come from Compose labels reported by Docker, so similar names alone do not
put containers in the same group.

| Control | Result |
|---|---|
| Container row | Current state, image, Compose labels, resource sample and recent logs |
| Refresh | Read current state and a new sample |
| Start | Start a created or stopped container |
| Stop / Restart | Confirm the interruption, then run the operation |
| History | Recent operations, who requested them, and their recorded outcome |

Logs are a snapshot of up to 200 lines, limited to 64 KiB. Resource samples
carry their capture time. Docker events refresh state when containers change;
there is no continuous resource or log stream.
Known sensitive environment values and common credential patterns are masked
from logs. Unlabeled secrets may still appear; avoid sharing raw output.

An accepted operation first appears as **running**. **Succeeded** means the
requested container state was observed; application health needs a separate
check. **Failed** reports a Docker error. **Unknown** means PiCode could not
verify the outcome, for example after a connection loss or restart. Refresh
the container before deciding whether to retry. PiCode never replays an
interrupted operation automatically.

## Operate a project

Expand a project and choose **Manage project**. Review start, stop or restart
to see the exact containers and any members that will remain unchanged.
Confirm the plan, then follow each step in **History**. A preview expires after
five minutes; a changed connection, membership or state requires a fresh review.

Project operations affect existing containers. They do not deploy Compose
files or apply Compose dependency order. **Partial** means some steps succeeded
and others failed. Each container is reserved against conflicting PiCode actions
until the job ends. External Docker clients can still change its state.

## Inspect resources and clean up

Open **Resources**, then expand Images, Volumes or Networks. Each resource lists
its consuming containers and projects, including stopped services. **Project
resources** shows the references for one project.

An image or custom local network with no consumers offers **Review removal**. The
plan names its full identifier and impact. Confirmation checks references again
and removes only that resource. Image size is the reported size, not a promise
of reclaimed space: layers can be shared. Volume and network size stays unknown.

Volumes are available for inspection only. Built-in, ingress and non-local
networks are protected. There is no blanket prune or volume deletion action.

## Check health and enable monitoring

Open **Health**, select a project and choose **Check health**. The sample shows
container state, configured health checks, restarts, CPU and memory. A running
container without a health check is labeled accordingly. Stopped containers,
missing metrics, an unreachable connection and stale samples are distinct states.

**Configure monitoring** enables observations while the browser is closed:

| Setting | Choices |
|---|---|
| Cadence | 30, 60 or 300 seconds |
| CPU threshold | 80%, 90% or 200% of one core |
| Memory threshold | 80%, 85% or 95% of the reported limit |
| Bad samples to open an incident | 2, 3 or 5 consecutive observations |
| Closed incident retention | 7 or 30 days |

Two good observations resolve an incident. Unknown evidence and connection gaps
break the streak; they do not prove recovery. Repeated breaches update the same
open incident. Unresolved incidents remain in history. Disabling monitoring
cancels collection. There are limits of 32 monitored projects, four concurrent
project samples and 128 containers per project.

Monitoring is off until you enable it and never starts a repair. PiCode keeps
the latest project sample and incident history, not a metrics chart history.

## Review a supervised procedure

Choose **Diagnose** in project health. The result separates observed conditions
from possible causes. Available procedures can stop repeated restarts, restart
an unhealthy service, start a stopped service or restart to reduce memory usage.

**Review procedure** shows its exact container, interruption and verification
steps. Confirm before execution. A failed or uncertain step stops the remaining
procedure. A restart can mitigate a symptom without fixing its cause. Starting
a stopped service does not prove it was a dependency or that other services
have recovered.

## Give a Pi agent sysadmin tools

`pi-sysadmin` is an optional Pi package. The Docker App works without it.
In **Packages**, select the target agent and install the local package path
`packages/pi-sysadmin` from your PiCode checkout (use its absolute path when
the agent works elsewhere). Restart that agent to load the tools. The package
is bundled in the repository; it is not published to npm.

Name the agent **Sysadmin** and give it a concrete task, such as “Inspect my
containers and explain why the web service is stopped.” The package also
provides the `docker-sysadmin` skill.

| Tool | Purpose |
|---|---|
| `docker_containers` | Find containers and Compose projects |
| `docker_container` | Read state, resources and recent logs |
| `docker_manage` | Request start, stop or restart; returns an operation ID |
| `docker_history` | Check the operation's result or recent history |
| `docker_resources` | Inspect images, volumes, networks and their consumers |
| `docker_plan` | Preview a project action, removal or named procedure |
| `docker_execute_plan` | Ask for confirmation, or file an Inbox review link |
| `docker_jobs` | Read maintenance jobs and every step result |
| `docker_health` | Read or request a timestamped project sample |
| `docker_diagnose` | Inspect conditions and suggest supported procedures |
| `docker_monitors` | Read saved monitoring settings |

Stop/restart use Pi's confirmation dialog. When a dialog is unavailable,
perform that operation from the Docker App. Declining a confirmation sends
no operation. The agent must check history before reporting success.
For a maintenance plan, missing confirmation UI creates an Inbox review link.
Open it to review and confirm in Docker. Marking the Inbox item done never
executes the plan. Monitoring settings are chosen in the App's Health tab.

| Pi surface | Compatibility |
|---|---|
| PiCode chat and Pi TUI | Same tools, with confirmation in Pi |
| A terminal Pi outside PiCode | Same tools, without a managed agent identity |
| PiCode Docker App | Independent of the package; uses the same service |

Connection settings follow the existing PiCode package convention:
`PICODE_DATA` or `~/.picode` for discovery, with `PICODE_URL` and
`PICODE_TOKEN` for an explicit PiCode server. Remote PiCode connections require
trusted HTTPS. The Docker engine itself must still be local to that server.
Canonical: [Pi packages](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/packages.md).

## Access and scope

The existing PiCode device/token authentication and OS user permissions apply.
Agent identity in history is provenance, not a separate security boundary.
Docker access can grant extensive control of the host; this package does not
isolate agents that already have access to the same socket. See
[Docker security](https://docs.docker.com/engine/security/).

Compose deployment, container creation/deletion, volume cleanup, backups and
remote Docker engines are later work. The current procedures never run a shell
command generated from logs or model output.
