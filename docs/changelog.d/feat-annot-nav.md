### Fixed
- Annotate mode survives a full page load: the shell re-injects the in-page
  script when a navigation completes, so the overlay no longer vanishes
  while the strip still says it is armed.

### Added
- The annotate harness doubles as the navigation check: after a real
  navigation the script is gone, and one re-injection arms the new document
  again and re-reports its (empty) state.
