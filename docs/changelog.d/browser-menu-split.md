### Fixed
- **Work-browser options menu in the agent split.** The menu's settings
  links (Browser settings, History, Downloads, Clear browsing data,
  Passwords) did nothing in the split pane — that surface never received
  the navigation callback. They now land on Settings ▸ Browser, like the
  tab version does.
- **Menu still.** An empty page capture used to hide the live page behind
  nothing (uniform gray). The tab now only hides the page behind a capture
  that decoded to real pixels; a failed capture keeps the menu usable over
  the host background.
- **Settings views under a live page.** Opening any settings view with a
  work tab selected left the page painted over it. The tab is now parked
  while the route is elsewhere and restored on return.
