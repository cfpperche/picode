### Changed

- **A Canvas panel's edges and corners are now something you can actually
  grab.** Every resize target is the same size to the hand however far in
  or out the plane is zoomed — a 20-pixel square at each corner, a
  12-pixel band along each edge — instead of shrinking with the camera
  until there was nothing left to aim at. What you *see* is the grid's old
  language back: a short bar at the middle of each edge and a corner mark,
  appearing when you hover, focus or select the panel.
- **The line between two linked panels is a curve**, and the line you drag
  while making the link is the same curve you end up with. Two panels
  sitting in the same row still join with a straight segment — that is what
  a symmetric curve between two aligned points is.
- **A link is easier to hit when zoomed out.** The invisible band along the
  line no longer thins with the camera.
- **The Canvas background texture can be seen.** Dots, Grid and Cross were
  drawn in the colour of a hairline — on the light theme the grid was at
  1.16:1 against the plane, which is to say invisible. All three now use a
  colour chosen for a texture, and the dot is two pixels rather than one.
  The spacing is unchanged, so nothing on a canvas moves.
- **A stopped agent's panel offers Run as the accented button** the agent
  tab uses for the same thing, instead of a grey chip that read as
  disabled. The same rule now runs through every panel placeholder: the
  action that *starts work* is accented, the action that only opens a tab
  or drops a dead binding is not.

### Fixed

- **The Inspector's branch chip is readable again.** A branch name and a
  worktree name were sharing one chip's width, each with its own ellipsis,
  and together they read as `· fe… ·…`. The row shows the branch, which is
  what it is for; the worktree is in the tooltip with the rest, and on a
  narrow rail the *unpublished* / *detached* word steps aside for the name
  instead of the other way round.
