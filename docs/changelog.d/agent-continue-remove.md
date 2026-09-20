### Changed

- A CLI agent row offers Continue in… (ADR-0088): a conversation pinned on
  its terminal continues in another CLI, the same submenu terminal rows
  have.

### Fixed

- Removing an agent no longer leaves the row behind when the delete
  response is lost: "not found" is treated as removed and the fleet is
  refetched, so a second remove cannot fail with "agent not found".
