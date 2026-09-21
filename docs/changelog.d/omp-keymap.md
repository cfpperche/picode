### Added

- **Omp's keyboard map is editable in PiCode.** Agent CLIs → Omp → Keyboard
  lists its 70 actions with the descriptions Omp itself gives them, and lets you
  add, replace or hand back a binding. PiCode writes Omp's own file — its
  `keybindings.yml`, or the `.yaml` or legacy `.json` file it is already using —
  and leaves every row it does not know about exactly as it was. A row the file
  holds in a shape PiCode will not rewrite is reported and never touched, a file
  that changed since the pane read it is refused instead of overwritten, and the
  tab says when Omp picks a change up (restart it).

### Changed

- The Keyboard tab of every agent CLI says what its map is: Pi and Omp are
  editable there; Hermes Agent points at the three keys its Settings tab carries;
  Grok and Muse Code link their own key list; and Claude Code, Codex, OpenCode
  and Antigravity say their editor has not shipped yet.
