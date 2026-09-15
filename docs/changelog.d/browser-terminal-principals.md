### Added
- **Terminal agents identify themselves to the built-in browser** (ADR-0143):
  the `browser` tool now sends the house identity tuple — managed agent id
  first, then the PiCode terminal id — and the daemon resolves a terminal's
  own grant (`browser.policy.term:<id>`) beside the existing per-agent key.
  A caller with neither stays read-only on the tab on screen.
