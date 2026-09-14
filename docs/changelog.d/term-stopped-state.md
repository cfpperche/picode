### Changed

- **A stopped CLI terminal is a real empty state, not a sentence in the
  corner.** Its pane now shows a mark, what stopped it (a plain stop, a
  restart that ended it, or a failed last launch), the conversation
  **Resume last session** would reopen — CLI, name and age, with the opening
  words on hover — and the two actions. A terminal with nothing pinned says
  *no session to resume* instead of offering no button and no reason.
- The pane's message is inked by the **terminal's** theme, not the app's, so
  a light app with the default dark terminal no longer draws dark text on a
  black pane. Narrow panes (down to the Canvas panel minimum of 256×224)
  drop the mark and stack the actions instead of clipping them.

### Fixed

- Resuming a stopped CLI terminal that fails now says why on the pane
  ("Codex was not found. Check its executable or PATH.") instead of nothing
  changing when the button is pressed.
