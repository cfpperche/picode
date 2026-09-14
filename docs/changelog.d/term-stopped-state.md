### Changed

- **A stopped CLI terminal is a real empty state, not a sentence in the
  corner.** Its pane now shows a mark, what stopped it (a plain stop, a
  restart that ended it, or a failed last launch), the conversation
  **Resume last session** would reopen — CLI, name and age, with the opening
  words on hover — and the two actions. A terminal with nothing pinned says
  *no session to resume* instead of offering no button and no reason.
- **The empty terminal pane follows the app's theme** (owner call): a light
  app no longer shows a black well where a terminal used to be — the ground
  is the app's own and the message reads in the app's inks, in both themes.

### Fixed

- Resuming a stopped CLI terminal that fails now says why on the pane
  ("Codex was not found. Check its executable or PATH.") instead of nothing
  changing when the button is pressed.
