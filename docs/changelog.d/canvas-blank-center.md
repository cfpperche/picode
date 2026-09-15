### Fixed
- **Canvas: the plane fills the pane again, and *No canvas yet* sits in the
  middle of it.** The page-frame change of 2026-09-14 dropped the surface's
  own flex column, so the stage collapsed to its content: an open canvas
  measured 0px tall (an invisible plane) and the empty state hugged the top
  of the tab. A native surface (Canvas, the QA demo) keeps that column now.
