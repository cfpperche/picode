### Fixed
- **Removing a workspace no longer takes the server down.** The git watcher
  killed the whole daemon when it noticed a watched folder was gone: that path
  had left the watch set, and the pass still read the group it no longer had.
  A removal — a workspace, its last agent, or a folder that stopped being a
  repository — is now a quiet pass; the removal event is what reconciles the
  sidebar, as it always did.
