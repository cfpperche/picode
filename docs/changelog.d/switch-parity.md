### Changed

- **A setting that is on or off looks the same everywhere.** Every pane now
  uses one switch for a boolean instead of a switch in Pi's and a checkbox in
  the other eight. A switch cannot show "nobody set this", so a row nobody set
  is drawn at the CLI's own default and the line beneath it says whose default
  that is.

### Fixed

- **Hermes' Show reasoning row said Off when Hermes turns it on.** Every
  boolean now declares what its CLI does when the key is absent, checked
  against a recorded table, so a row left alone shows what will actually
  happen. Hermes' turn limit says 20 rather than a vague "default".
