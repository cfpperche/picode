### Fixed
- **Muse Code's own plugins are listed.** `muse plugins list` nests each
  plugin under `record` and `plugin`, which PiCode read as a list with no
  names — so a machine that has Muse plugins saw an error where the list
  belongs. Install, remove and enable/disable already worked and are unchanged.
- **The Muse plugin surface is per machine, and the pane says so.** Muse turns
  its plugin commands off through its own cached feature configuration; when
  that is the case the CLI answers *"plugins are not available in this build"*
  and PiCode shows the CLI's own sentence instead of an empty list.
