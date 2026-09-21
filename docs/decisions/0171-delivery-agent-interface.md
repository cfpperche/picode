# ADR-0171: Common delivery declarations for agent CLIs

- **Status**: accepted
- **Date**: 2026-09-21
- **Boundary**: protocol, persistence and process — a shared CLI/MCP declaration contract and durable revision-bound review requests, without integration or deployment authority.

## Context

The delivery-flow benchmark and D0 design distinguish agent activity from actual
integration and publication. Git can discover branches but cannot establish an
agent's intent to deliver a particular revision. The owner approved a common
agent interface on 2026-09-21. Requiring vendor-specific hooks for registration
would make that intent depend on the CLI instead of the project.

## Decision

Provide `picode delivery` and the optional `picode mcp delivery` family over one
`POST /api/delivery/tool` handler. Registration supplies title, local source
branch, full revision and local target branch. The daemon derives repository and
principal from the registered launch, checks Git and persists a stable delivery
ID in SQLite. The initial actions are capabilities, register, update,
request-review, withdraw-review, show and list. A review request is an agent
declaration, never a human approval, validation result or execution request.
Updates clear review requests. Observed source changes are reported separately.

Mutations require a request key and, after registration, the expected version.
Rows, original retry receipts and `delivery.changed` events commit together.
An exact retry returns the original result without reverting subsequent changes.
A changed payload under the same key conflicts. Reads are scoped to the launch
repository; writes additionally require the original principal. Repository
identity uses the canonical Git common directory, shared across worktrees.

The interface uses the existing daemon authentication and inherited
`PICODE_AGENT_ID`/`PICODE_TERM_ID` convention. These identify a PiCode launch,
not an attested vendor conversation. This is the existing same-user trust model,
not isolation against local processes with the install token. An unknown or
removed launch is refused; declarations survive launch removal as history.
Neither tool arguments nor the CLI accept an arbitrary repository or actor.

## Consequences

Any shell-capable CLI launched with PiCode's identity, binary and daemon access
can use the common command. Optional MCP exposure depends on existing launch
support and configuration; no universal vendor-native MCP integration is claimed.
Automated tests exercise all nine catalog identifiers, not authenticated vendor
sessions. Existing launches need access to the updated binary and daemon.

The registry is bounded at 1,000 declarations per repository and 10,000 receipts
per repository/principal. Capacity refuses new mutations; existing receipts
remain replayable. There is no pruning or reassignment in this first slice.
Moving a repository changes its key; history migration is future work. Git checks
are observations, not a lock against concurrent external Git changes. The source
can move after a successful request; consumers must inspect current evidence.

Queue and deployment capabilities are explicitly unavailable. No code invokes
project scripts, CI, merge, deploy or force/restart. ADR-0170 remains proposed for
the separate observation/receipt contract; ADR-0105 still owns delivery execution.
The first slice introduces no browser delivery screen or validation provider.

## Alternatives considered

- Git discovery alone: useful for observation, but cannot express explicit review intent.
- Vendor hooks as the required entry point: fragment the contract and exclude generic CLIs.
- Native conversation identity: stronger attribution, but requires separate vendor lifecycle integrations; retain explicit launch scope for this bounded declaration interface.
- Reuse review requests as approval or queue enrollment: grants authority that this declaration does not carry.
