### Fixed
- Annotations reach the strip even when the page cannot post to the app:
  the annotate strip now asks the host for the page's state on a slow tick
  (a path that needs no page-side bridge), so the count and Send light up
  and the batch carries the real payloads.
- The work-browser page is created with web messages enabled, instead of
  enabling them only when annotate mode is armed — which left the already
  loaded document without the channel.

### Added
- Annotate mode mirrors the page: pressing Esc in the page turns the strip
  off too (the pull says "off"), and a page with no script yet (a navigation
  in flight) is told apart from an off page.
