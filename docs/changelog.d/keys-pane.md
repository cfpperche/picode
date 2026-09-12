### Changed

- **The keyboard map is a pane of its own.** It was a *Keys* sub-tab under the
  Settings pane, which put two tab rows inside one view; it is now **Keyboard**,
  sitting between Settings and Packages in the Agent CLIs pane bar
  (`#/clis/pi/keyboard`). The Settings pane no longer has a sub-tab row, and the
  link keeps the agent and layer you came from, so going back lands where you
  left. A `?tab=keys` bookmark still opens it.

### Fixed

- Clicking the keyboard map no longer silently returns to Settings: the pane
  rewrote the route to carry the selected agent and dropped the sub-tab with
  it (the new pane cannot lose a tab it does not have, and the layer survives
  the same rewrite).
