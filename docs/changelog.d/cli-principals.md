### Added

- A workspace can bind an Agent CLI terminal as a **managed principal**
  (`term:<id>`): the same identity Computer and Browser grants already use,
  without creating a Pi agent or opening a chat. Create with
  `POST /api/workspaces/{id}/principals`. Removing the binding leaves the
  terminal and the vendor's files alone.
