### Fixed
- **Opening a link no longer flashes a stray page.** A browser tab created
  before its pane reported its size painted at a placeholder rectangle over
  the pane's own empty state until the real bounds arrived. Such a tab is now
  born hidden and appears in place — the moment its rect is known.
