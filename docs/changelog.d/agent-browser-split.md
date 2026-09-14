### Added
- **Agent browser split (ADR-0135)**: a terminal that hosts an agent — the
  agent's TUI or a CLI launch — can open the work browser beside it
  (right-click → Open browser), bound to that agent so the `browser` Pi tool
  acts on the pane on screen. The split resizes (drag), maximizes, and closes
  from the pane's own controls.
- **The split survives a relaunch**: the layout (which tabs host a pane, the
  dragged width, the maximized flag) and each pane's last url persist in
  localStorage. After the desktop app restarts, the panes come back at the
  pages they were on — the first time a host tab is shown, its webview is
  recreated where the pane is. Panes whose agent or terminal is gone are
  pruned on boot instead of reopening to nowhere.
