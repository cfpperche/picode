### Removed
- **Agent CLIs: the general Providers tab is gone.** Provider accounts of a
  CLI are read in that CLI's **Providers** pane, so the tab strip no longer
  carries a second CLI picker.

### Changed
- A CLI's page hosts **Launch**, **Terminals**, **Sessions** and
  **Providers** as inner tabs. The catalog is the CLI picker. Pi shows the
  account roster; other CLIs explain that Providers is unavailable and offer
  Open Pi. `#/clis/<cli>/providers` is the list;
  `#/clis/<cli>/providers/new` opens Add provider. Old `#/clis/providers*`,
  `#/providers*` and `#/more/providers*` rewrite onto those addresses.
  OAuth still returns to the list in the same app.
