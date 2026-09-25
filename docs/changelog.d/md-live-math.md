### Added

- **Math and diagrams in the Live view.** Formulas written with `$…$` and
  `$$…$$` and Mermaid diagrams now appear drawn while you edit a markdown file
  in Live; click one (or move onto it with the arrow keys) to edit its source.
  Amounts like "$5 and $10" stay plain text.

### Fixed

- **Prices no longer turn into formulas in markdown previews.** Text like
  "it costs $5 and $10" is shown as written instead of as math.
- **A broken Mermaid diagram no longer leaves its error drawing behind.** The
  error graphic could pile up at the bottom of the page and shift it.
