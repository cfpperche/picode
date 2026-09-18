### Changed
- **Browser tool: a screenshot reaches the agent as an image.** `browser
  screenshot` now hands the capture straight to the model instead of writing
  a PNG to a temp path and returning the path — one call to see the page, no
  `read` after it.
