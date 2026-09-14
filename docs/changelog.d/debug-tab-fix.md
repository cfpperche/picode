### Fixed
- Desktop: clicking a work-browser tab in the tab strip now selects it (it
  silently did nothing — `openTab` had no web branch and fell through to the
  agent lookup), and a selected browser tab no longer snaps back to the
  previous tab on the next route reconcile. Work-browser tabs own a
  `#/web/<id>` address the router writes and honors.
