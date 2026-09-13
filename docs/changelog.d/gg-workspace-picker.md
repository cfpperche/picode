### Added
- **Git graph: switch workspace from the toolbar.** The graph's first item is
  no longer the repository's name but the workspace its history is read
  through — a dropdown of your other workspaces, each with the branch that
  folder is on. Picking one retargets the reading folder without leaving the
  tab: sibling worktrees of one repository swap in place (the HEAD dot and the
  `this worktree` row move), and a pick into another repository moves the tab
  to it, so one tab per repository still holds. Folders that are not
  repositories are never offered, and a pick that cannot resolve leaves the
  graph exactly as it was.
