# 2026-09-20 — pi-rows-table: Pi's rows stop being hand-written JSX

Why this branch exists: the owner asked for Pi's six original rows to move onto the same table the eight guest CLIs use. It closes the loop on the complaint that started the day — Pi persisted about forty settings keys and its pane showed eight, and the reason was mechanical: the guests declare rows in a table, Pi's were JSX, so adding one meant editing a component and nobody did.
What landed: all eleven rows are declared in `web/shared/domain/piRows.js` with their label, kind, group, order and the keys each reads, and the pane renders whatever the table says. Three of Pi's rows are not scalars and are declared as their own kinds rather than flattened into one — `model` is the three coupled selects the catalog feeds, `patterns` is the free list of scoped models, `tools` is the grid over pi's fixed tool set. Each keeps the control it always had; what changed is that the table decides they exist, where they sit and in what order.
The grouping came with it, so Pi's pane reads like the guests' now: Session, Model, Approvals, Interface, same provenance line, same "Use inherited" on a row this layer set. A row marked `machine` is offered on the This machine layer only, which the table enforces and the server already refused by name.
One thing the refactor exposed: the layer wrapper was itself a `settings-section`, so the groups nested the class inside itself and put the `+ .settings-section` sibling margin in reach of rows it does not describe. The wrapper is a plain element now and the groups are the only sections — four of them, eleven rows, confirmed in the live DOM rather than inferred.
Verified live on a scratch instance with a copy of the owner's real `~/.pi/agent/settings.json`: four group headers, eleven rows, Theme reading `dark`, the overlay audit clean, no horizontal overflow at 390px, and the mobile quick sheet (`agentOnly`) still present — that component has diverged from its desktop twin before and a straight copy destroyed it earlier today.
State: `make close` green, `main` can fast-forward.

## Next up

- `web/shared/domain/resolveLayer.js` is still a named list, so a new Pi row is two lines rather than one, and a key added to only one of them renders empty. Deriving it from `piRows.js` would finish the job.
