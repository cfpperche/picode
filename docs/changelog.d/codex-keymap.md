### Added

- **Codex's keyboard map is editable in PiCode.** Agent CLIs → Codex → Keyboard
  lists its 149 keymap keys in the twelve contexts Codex puts them in — global, chat,
  composer, editor, the four vim modes, pager, list, agents and approval — with
  the description each key carries in Codex's own schema and the chords it
  binds out of the box. PiCode writes `[tui.keymap.<context>]` tables in
  `~/.codex/config.toml` and leaves every key it does not manage exactly as it
  was, including everything the Settings tab edits. Chords are shown and written
  in Codex's own spelling (`ctrl-alt-m`, `page-down`); a chord Codex cannot
  express is refused by name rather than written, because Codex validates its
  keymap when it starts and will not start on one it cannot parse.

### Changed

- The Keyboard tab of an agent CLI whose map PiCode writes now says when that CLI
  reads a change, in the CLI's own terms: Codex applies one when it starts, and
  only its own in-session editor is live.

### Fixed

- A keyboard row on a phone keeps its label and its keycaps on one line with the
  note under them, instead of squeezing the keycaps into whatever the desktop
  grid left over — a CLI whose labels are long (Codex's are full sentences) had
  its chips drawn over the label. The narrow-window layout on the desktop app
  gets the same treatment.
