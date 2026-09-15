### Added
- **Muse Code launch is editable.** Its Launch tab has the real defaults
  editor (executable, arguments, PATH, environment), profiles, the
  Customize checkbox and the row menu's **Launch settings** — the gaps in
  the terminal `···` menu are closed for Muse. Sessions and resume are
  unchanged.

### Changed
- **No Activity reporting for Muse Code, stated plainly.** Its build
  offers no hook surface, so the toggle is replaced by a one-line note
  ("its terminals stay Open") and the server refuses `integration: true`
  with 400. The startup seed that switches Activity on for empty configs
  skips hookless CLIs.
