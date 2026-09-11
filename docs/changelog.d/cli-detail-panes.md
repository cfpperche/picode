### Removed
- **Agent CLIs: the general Sessions tab is gone.** Sessions of a CLI are
  read in that CLI's **Sessions** pane, so the tab strip no longer carries a
  machine-wide list.

### Changed
- A CLI's page hosts **Launch**, **Terminals** and **Sessions** as inner
  tabs. The catalog is the CLI picker; switching CLI keeps the pane.
  `#/clis/<cli>/sessions` lists every folder; `#/clis/<cli>/sessions/<id>`
  lists one. Old `#/clis/sessions*`, `?cli=` and `#/sessions*` links rewrite
  onto those addresses. Dashboard top-session rows open that CLI's pane.
