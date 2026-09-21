# 2026-09-21 — feat/apps-grid-dnd: drag tiles in the Apps grid

The Apps tab uses the same sortable lift and dashed slot as the sidebar, with `rectSortingStrategy` so a tile can change column. Order is `settings["apps.grid.order"]` via `GET/PUT /api/apps/order` (`setting.updated`). No saved order stays alphabetical. Search filters and does not write the order. The phone grid is still alphabetical.

## Next up

- The phone Apps list does not drag and does not read the saved order yet.
