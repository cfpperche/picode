### Added

- **The Snippets guide now covers the whole editor**: what the slug line means
  while you type (free, or in use with Save off), the placeholder table
  (default, optional, and the choices list that becomes a dropdown when you
  send), where snippets come from (starters, saving a selection, importing a
  prompt, duplicating), and that a half-written snippet no longer dies with the
  tab.

### Fixed

- **The guide was showing `&#123;&#123;name&#125;&#125;` instead of `{{name}}`.**
  Every placeholder on the public Snippets page was written as an HTML entity,
  which a code block renders as literal text. The braces are real now, on that
  page and everywhere the guide shows one.
