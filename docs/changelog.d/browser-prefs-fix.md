### Fixed
- **The download switch (and every browser preference) survives a save
  now.** The page read the daemon's preferences twice — on load and after
  each save — with two copies of the same mapping, and the save path's
  copy had lost `askDownload`: the switch turned on, the answer came
  back, and the row snapped off, never to turn on again. One reader,
  `web/browser/src/lib/browserPrefs.js`, with the regression covered by
  tests.
