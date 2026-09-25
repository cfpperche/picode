### Removed

- **Old Pi-era Sessions and Providers links.** `#/sessions*`,
  `#/clis/sessions*`, `#/providers*`, mobile `#/more/providers*` and
  `#/clis/providers[/<cli>]` no longer open Pi's panes. A CLI's panes live at
  `#/clis/<cli>/sessions` and `#/clis/<cli>/providers`. `#/providers/llama`
  still opens llama.cpp.

### Fixed

- **A malformed Add provider link is refused.** `#/clis/<cli>/providers/new/<extra>`
  opened the Add provider dialog; it now reads as an invalid link, like any
  other path after the pane.
