# ADR-0159: CLI terminals as managed principals

- **Status**: superseded by [ADR-0160](0160-cli-runtimes-are-agents.md) (workspace CLI instances are agents; this table is transitional)
- **Date**: 2026-09-19
- **Boundary**: persistence — a workspace may bind an Agent CLI terminal as a
  durable principal; security model — that binding is the same `grant.Key`
  identity browser and computer already use (`term:<id>`). No agent protocol.
- **Amends**: ADR-0056 (this is the deferred tier-2 ADR, without a composer),
  ADR-0069 (guests remain not Agent records for chat/RPC/Pi packages),
  ADR-0091 (still no ACP/app-server/SDK client; a guest is not a protocol
  agent). ADR-0006 is untouched: one live `pi` process per *agent* row.

## Context

PiCode already launches, sensors, pins, hands off, and talks to guest CLIs
through terminals (ADRs 0069, 0056, 0079, 0084, 0088, 0089, 0107, 0143).
Managed *agents* are still Pi rows in `agents` with `Runtime.Start` and a
composer. The owner asked to reuse the Agent CLIs stack inside the managed
surfaces (fleet, Inbox, automations, Inspector) and to keep structured chat
as the last concern. ADR-0091 refuses an agent-protocol client until the
market converges; ADR-0056 left observe-only guest agents waiting for their
own ADR.

Putting guests into `agents` would make `Runtime.Start` try to spawn `pi`.
Waiting for ACP would reopen 0091. Doing nothing leaves every surface
special-casing `agentId` vs `termId`.

## Decision

A **managed principal** is `grant.Principal`: `{kind: agent, id}` or
`{kind: terminal, id}`. `grant.Key` stays the house spelling (`agent id` or
`term:<id>`). Pi agents stay rows in `agents`. A guest is never an `agents`
row.

A workspace may **bind** one Agent CLI terminal as a managed CLI
(`managed_clis`: workspace, catalog CLI, terminal, name). The terminal
record, launch settings, tmux session and vendor files stay the Agent CLIs
implementation. Binding does not start a process, does not call
`Runtime.Start`, and does not open a composer. Unbinding deletes only the
row; the terminal and `~/.claude` (etc.) remain. Deleting the terminal
cascades the binding.

`POST /api/workspaces/{id}/principals` creates or binds. `cli=pi` is
allowed as a TUI principal (the 0056 manual-Pi case) and still does not
create an agent. Structured chat, JSON-RPC, ACP and `SendTurn` for guests
stay out of this ADR; later slices rekey Inbox, fleet, automations and
`picode mcp` onto `Principal` without changing this table.

## Consequences

Easier: fleet, Inbox, grants and automations can take one identity type;
Fatia 3+ can "create a Claude in the workspace" without a second runtime.
Harder: two ids must not describe one process — the unique `terminal_id`
and the rule "never insert guests into `agents`" are the invariant.
If we are wrong, the failure is an extra row in the sidebar, not a
second writer on a Pi session.

## Decision table

| Conditions | Action |
|---|---|
| Workspace exists, launchable catalog CLI, no `terminalId` | Create a workspace terminal, set its launch CLI, insert `managed_clis`. No agent row. No process. |
| `terminalId` in that workspace, launch CLI matches (or none yet) | Bind that terminal |
| Terminal already bound | 409, nothing written |
| Terminal in another workspace | 400 |
| Unknown or non-launchable CLI | 400 |
| `ws_free` | 400 (binding is a workspace act) |
| Unbind | Delete `managed_clis` only |
| Delete terminal | Cascade binding; announce `managed_cli.removed` then `terminal.deleted` |
| Any path above | `Runtime.Start` is not called |

Coverage: `TestPrincipalIdentity`, `TestAddManagedCLIDecisionTable`,
`TestEveryMutationAppendsAnEvent` (`AddManagedCLI`, `RemoveManagedCLI`),
`TestManagedCLIHTTPDoesNotCreateAgent`.

## Alternatives considered

- **Guest rows in `agents` with a kind column.** Lost: ADR-0006 and
  every `GetAgent` → `Runtime.Start` caller become a trap.
- **Adopt ACP / app-server now (0091).** Lost: unconverged protocols;
  replaces the TUI the owner uses.
- **Observe-only agent entity (0056 tier 2 as written).** Too small —
  the owner wants delivery through doors that already exist (0089, 0107,
  0154), not a second fleet strip. This ADR is the identity seam those
  doors attach to; it does not itself paste into a TUI.
- **No table — treat every CLI terminal as managed.** Lost: ordinary
  project shells would appear as agents.
