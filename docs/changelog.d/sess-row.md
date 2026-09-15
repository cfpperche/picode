### Fixed
- **A long session preview no longer breaks the row.** In the Sessions tabs the
  second line is the CLI's own prompt text — a paragraph, or a quoted Windows
  path. It sized the line to its whole text and pushed **Open in terminal** and
  the row menu outside the card; at the same time the model name (a secondary,
  single-line attribute) refused to shrink, squeezed the session name to
  nothing and painted over the age/size/count. A session row is now two
  deterministic lines: identity + facts, then the preview with its buttons,
  each clamped with `…`. Seen on Muse Code and Grok; every CLI's Sessions tab
  uses the same row.
