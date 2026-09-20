### Added

- **Inspector Changes follows dirty worktrees.** When the anchored folder is
  clean but a linked worktree of the same repository has uncommitted
  changes, the rail shows that checkout behind a Following pill (Back
  returns to the anchor); several dirty checkouts render one branch-headed
  group each. The branch chip, Git menu and commit dialog address the
  followed checkout, and its files open in the center through the same
  file tab. Works for agents, terminals and workspaces, with any CLI.
