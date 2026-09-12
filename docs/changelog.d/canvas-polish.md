### Changed

- **The canvas switcher is a name until there is something to switch to.**
  With one canvas the top-left control opened a list holding the canvas you
  were already on. It is now simply that canvas's name; the moment a second
  canvas exists it becomes a picker again. **New canvas** is in the `⋯` menu
  either way, and the controls do not shift when the second one appears.
- **An empty canvas now offers its next step in the middle of the plane** —
  one line saying what a panel is, and **Add panel** — instead of a blank
  plane with the controls in the corner. The plane, its background and the
  minimap stay where they are, and the line goes the instant the first panel
  lands.
- **The plane no longer shows the React Flow credit.** The drawing library
  behind the canvas is MIT-licensed and unchanged; only its badge is hidden.
- **What's New opens on a fresh install too.** A released build used to wait
  until you had made a workspace, an agent or a terminal, which meant a brand
  new install saw nothing and looked broken. It now opens once on the first
  load, still closes for good when you dismiss it, and still waits while a
  dialog, a reconnect, a waiting agent, an Inbox item or a create/share flow
  needs you first.
