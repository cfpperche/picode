# ADR-0173: Sidebar order is a stored position

- **Status**: accepted
- **Date**: 2026-09-21
- **Boundary**: persistence and protocol — `workspaces`, `agents` and `terminals` gain a `position`, and the change feed gains `workspace.reordered`, `agent.reordered` and `terminal.reordered`

## Context

The sidebar showed workspaces by name, agents in a workspace by creation time, free agents by display name (sorted again in the browser), and terminals by name (sorted in SQL and again on the client). A newly added workspace was inserted back into alphabetical order by the fleet reducer. The phone reads the same fleet and calls that sequence "sidebar order", so a private browser sort cannot be the order.

Dragging a row is a later gesture. The order itself has to exist first, and it has to be the same on every client. `automation_schedules.position` is the existing shape for a small ordered list.

## Decision

Each workspace, agent and terminal has an integer `position`. Lists the sidebar shows are `ORDER BY position, id` within their container: non-free workspaces; agents of one workspace (the free workspace included); terminals of one workspace. A new row takes `max(position) + 1` in that container. Deleting a row leaves the others where they are.

`ReorderWorkspaces`, `ReorderAgents` and `ReorderTerminals` rewrite positions `0..n-1` for one container when `ids` is a permutation of that container, in the same transaction as the event. The same order writes nothing and emits nothing. Any other list is refused and writes nothing. A terminal bound to an agent is not part of the terminal container: the agent row is the sidebar row for it.

The migration ranks existing rows the way the sidebar showed them that day: workspaces by name, agents by `created_at`, free agents by name, terminals by name. `DefaultAgent` stays the oldest agent, not the first in the sidebar.

Clients append a newly added workspace and do not sort these lists again. A reorder event carries the ids; a client that cannot apply them refetches.

## Consequences

New workspaces and new terminals appear at the bottom of their list instead of sliding into alphabetical order. Every browser, the desktop shell and the phone show one order once a reorder is stored.

A wrong `ids` list cannot partially apply. Two creates in the same container can theoretically take the same position; the next reorder rewrites a dense sequence, and `id` breaks ties until then.

The drag gesture, move-between-workspaces, and mixing agents with terminals into one sequence are not this decision.

## Alternatives considered

- **Keep sorting and store the order in localStorage.** The phone and a second browser would disagree, and a refetch would undo it.
- **Lexicographic rank (one row touched per drop).** The lists are dozens of rows. Rewriting `0..n-1` is one transaction and a trivial feed payload.
- **One mixed sequence of agents and terminals per workspace.** That needs a union table and changes the collapsed group, which shows agents first and terminals after. Reorder stays inside each block.

## Amendment — presentation-only views (2026-09-23)

The rule "clients append a newly added workspace and do not sort these lists again" governs the stored order: a client sort that fed positions back or survived a refetch would undo a reorder and split the clients. It does not forbid a presentation-only view. The sidebar's **Working first** reorders rows on one device, per container, as buckets of `agentRowStatus` (needs-you, working, interactive, open, ready, stopped — a status outside the list still renders, at the end); inside a bucket the stored position rules; it writes no position, emits no event, and lives in the client's local preferences. While the view is on, the drag handle and Move up / Move down hide: a drop or menu move under a sorted view would write positions the user is not looking at. **Sort agents by name** is not a view — it is a one-shot reorder through the same PUT and event as a drop, and every client keeps one order.

**Amendment — park-time ranking inside the view (2026-09-24).** The owner asked the view to rank every agent that is not working by how long it has been parked in its status, shortest park first. Inside the working bucket the stored position still rules; every other bucket sorts by the row's truthful status stamp — the same `agentStatusStamp` the row's age pill shows, so bucket order and pill age cannot disagree — and rows with no stamp keep the stored order after the stamped ones. The view still writes no position and emits no event; without the view on, the stored order rules unchanged.
