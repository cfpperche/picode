# 2026-09-21 — feat/sidebar-order: stored sidebar order (ADR-0173)

Migration 064 adds `position` on workspaces, agents and terminals and ranks existing rows the way the sidebar showed them (name, created_at, free agents by name, terminals by name). New rows append. `ReorderWorkspaces`, `ReorderAgents` and `ReorderTerminals` rewrite one container when `ids` is a permutation, same transaction as `workspace.reordered` / `agent.reordered` / `terminal.reordered`; the same order emits nothing; any other list is a 400 and writes nothing. A terminal bound to an agent is not a terminal row. `PUT /api/workspaces/order`, `PUT /api/workspaces/{id}/agents/order`, `PUT /api/workspaces/{id}/terminals/order`. The fleet reducer appends a new workspace and applies a permutation or refetches. Browser and phone stop re-sorting terminals; the Agents tab stops re-sorting free agents.

Verified: `go test ./internal/store/` pass; reorder HTTP test pass; `node --test` on feedReducers and both termGroups tests pass. OpenAPI regenerated.

Drag: `@dnd-kit/core` 6.3.1 pointer sensor (6px) on the workspace name and on agent/terminal rows. Separate lists, so an agent cannot drop onto a terminal. Move up / Move down in each row menu; Space and Enter still open the row. Optimistic fleet patch, PUT, rollback if that container is still the failed order. Phone shows the order and has no drag.

visual-review: PASS (sidebar-order-menu.png, sidebar-order-list.png, sidebar-order-narrow-open.png; overlayAudit ok; card 5/5). Scratch `sidebar-order`. Drag Bravo to the top and Nova above Atlas persisted through GET /api/workspaces. Move down on Bravo's menu put QA first.

## Next up

- The phone shows the stored order and has no drag gesture.
