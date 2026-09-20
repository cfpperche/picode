### Fixed

- **Session Changes no longer corrupts fleet pills.** Linked-worktree
  watcher events are path-only, so the workspace and agent pills keep
  describing the anchor folder while followed groups still update live.
- A file tab whose worktree checkout is gone now reads "That file is
  gone." instead of the raw server message.
