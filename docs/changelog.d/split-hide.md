### Fixed
- **The agent split's browser pane stayed painted over other tabs**: the
  native WebView2 kept its last bounds when the hosting tab was switched
  away (the pane mounts only for the active tab, and unmount never told the
  shell to hide). Switching tabs — or closing the pane — now parks the
  view; its page state survives and remounting brings it back.
