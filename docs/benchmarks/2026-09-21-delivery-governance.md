# Study: observing and governing agent delivery

- Date: 2026-09-21.
- Question: Is tracking agents waiting to land and deploy a general product need,
  and how do PiCode's benchmarks address it?
- Method: official public documentation and changelogs inspected on this date;
  no hands-on verification of competing products. Repository `main` documentation
  may describe features ahead of the distributed release. Absence in the sources
  below is not proof that a product lacks a capability.

## Findings and sources

| Benchmark | Documented capability | Adaptation and limit |
|---|---|---|
| [Cursor PR review](https://prod.cursor.com/changelog/05-07-26) and [Automations](https://cursor.com/docs/cloud-agent/automations) | PR review brings comments, commits, changes and reviewer context together. Automations respond to source-control and CI events. | Keep delivery context near agent work. These sources do not establish a unified land-and-deploy queue. |
| [t3code source control](https://github.com/pingdotgg/t3code/blob/main/docs/user/source-control.md) | PR page, linked threads, auto-merge pending checks, and GitHub stack operations respecting branch rules and merge queues. Threads can settle once all linked reviews are terminal. | Link changes to conversations, and reuse provider governance. A terminal review may be closed without being merged; settling a thread is not proof of deployment. This is documentation from main. |
| [Paseo worktrees](https://github.com/getpaseo/paseo/blob/main/public-docs/worktrees.md) and [changelog](https://paseo.sh/changelog) | Isolated workspaces support diff review, merge and archival. The September 2026 changelog records Ready to review and automatic PR-tab opening. | Review readiness deserves a visible state. These sources do not establish a project deployment queue. |
| [Linear Releases](https://linear.app/docs/releases) | CI/CD integration associates issues with releases and environments, with continuous and scheduled pipelines. | Done and delivered are distinct; deployment evidence comes from a pipeline. Available plan restrictions must be checked before any adoption. |
| [Vercel promotion](https://vercel.com/docs/deployments/promoting-a-deployment) and [Rolling Releases](https://vercel.com/docs/rolling-releases) | Deployments have identities, promotion and rollback; staged rollout exposes candidate traffic and comparative metrics. | Separate a successful build from production serving that build. Do not copy hosting-specific rollout mechanics into a generic ADE. |
| [GitHub merge-queue governance](https://docs.github.com/en/enterprise-cloud%40latest/repositories/configuring-branches-and-merges-in-your-repository/configuring-pull-request-merges/managing-a-merge-queue) and [queue visibility](https://docs.github.com/en/pull-requests/how-tos/merge-and-close-pull-requests/merging-a-pull-request-with-a-merge-queue) | Ordered integration validates changes with the current target and earlier queued changes; queue removals expose failure reasons. | A readiness list is not an execution queue. Provider eligibility and merge-group CI configuration constrain direct reuse. |

## Interpretation, not a prevalence claim

The sources establish recurring product investment in review readiness,
integration governance and release visibility. They do not measure how many
PiCode users need a formal queue. The working hypothesis is that the need grows
with concurrent changes to one repository, mandatory gates, dependencies and a
delay between merge and publication. One developer supervising several agents
can encounter the same bottleneck as a team.

PiCode's own development amplifies it: worktrees, fast-forward landing, changing
base revisions, explicit owner deployment and serialized restarts. Updating the
system that hosts the working agents also couples delivery to their availability.
Those are project-specific policies, not defaults to impose on every workspace.

## Local evidence and limits

The operating contract and `scripts/land.mjs` distinguish branch closure,
fast-forward integration and the full CI run on main. CI can fail after the
branch has already landed; the UI must preserve that fact. The Makefile serializes
deploy and desktop restarts through `/tmp/picode-mutate.lock`; a lock alone does
not provide an observable, durable delivery queue. Deploy records are appended
by `internal/install/install_unix.go` to the data directory's
`var/deploy-log.jsonl`. Availability and semantics must be rechecked during the
implementation inventory; a deploy record alone does not prove current health.

## Recommended adaptation

Track a change linked to its repository, branch/revision and agent sessions.
Show integration and deployment as separate lifecycles. A publication may include
several integrated changes. Start with trustworthy observations and blockers,
then add explicit queue governance. Unknown or stale evidence remains visible;
an agent's completion message is never sufficient evidence of delivery.

The execution baseline is [Delivery flow](../plans/delivery-flow.md). This study
adds a focused adaptation; it does not replace existing benchmark choices or
accept a new process ADR. Public release cadence remains a separate topic in
[the release-cadence study](2026-09-04-release-cadence.md).
