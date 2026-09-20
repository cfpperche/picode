### Added
- `docs/architecture/work-browser.md` — the work browser's own architecture
  file (who decides what, the command channel, tabs, the "a layer never paints
  over a WebView2" rule, permissions, annotations, and the traps that are
  architectural rather than dated).

### Fixed
- The stale `btab_layer` permission artifact is gone; the generated schemas no
  longer advertise a command that does not exist.
