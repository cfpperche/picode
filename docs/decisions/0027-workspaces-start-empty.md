# ADR-0027: Workspaces start empty

- **Status**: accepted (amends ADR-0011 item 4; its "owns zero or more agents" clause now holds literally)
- **Date**: 2026-08-31

## Context

ADR-0011 declared a workspace "owns zero or more agents" but its item 4 kept
"adding a workspace still creates a first agent", and the New-workspace form
carried Provider/Model/Thinking plus a session-adopt shortcut — creating a
workspace and creating an agent were fused. The owner wants them separate:
a workspace is a project folder; it can sit with no agents and no terminals
for as long as the user likes (the sidebar's workspace card was already
built for that — "Empty — add an agent or a terminal.").

## Decision

`POST /api/workspaces` registers the folder and nothing else. The response
carries `agents: []` and **no `agent` key**; `workspaceView.Agent` became a
pointer with `omitempty` because a zero-value object read as a truthy agent
with an empty id in every client. An idempotent re-add of a registered path
returns the workspace with whatever agents it really has — it no longer
resurrects a deleted default agent. The New-workspace form asks for name
and folder only. Agents come from `POST /api/workspaces/{id}/agents` (or
the free-agent form); the legacy JSON-registry import still creates its
default agent, since pre-ADR-0027 registries expect a usable workspace.

Workspace-scoped endpoints that need an agent (open, close, sessions,
status) answer **409 "workspace has no agents — add one first"** on an
empty workspace — previously a 500 or a misleading 404 "workspace not
found".

## Consequences

- **Easier**: registering a project carries no model/provider decisions;
  empty workspaces are honest instead of hiding a synthetic "default".
- **Breaking**: clients that assumed `agent` always present in workspace
  JSON; the shape is unchanged whenever at least one agent exists.
- **Accepted cost**: a workspace with no agents cannot be "selected" — its
  card only collapses/expands; the empty state's actions are the way in.
- **If wrong**: recreate the default-agent call in `AddWorkspace`; the
  store helper (`ensureDefaultAgentTx`) still exists for the importer.

## Alternatives considered

| Alternative | Why not |
|---|---|
| Keep creating the agent when provider/model come in the body | Keeps the fusion the owner asked to break; two creation contracts |
| `agent: null` instead of omitting | Same effect for JS; omitempty is idiomatic Go and smaller |
| 404 for empty-workspace open/sessions | The workspace exists; 404 says it doesn't |
