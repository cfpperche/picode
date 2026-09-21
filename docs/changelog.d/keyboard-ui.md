### Added

- **Keyboard → Find by key**: press a chord and the map narrows to the actions
  that answer to it — the fastest way to answer "what is `Ctrl+T` doing?".
- **Keyboard → Reset all**: hands every key this pane changed back to pi's
  default in one write, and leaves keys PiCode does not know where they are.
- A **warning on the chords a browser keeps**. Five of pi's own defaults
  (`Ctrl+T`, `Ctrl+W`, `Ctrl+N`, `Ctrl+PgUp`/`PgDn`) never reach the terminal
  inside PiCode; the rows that use them now say so instead of silently doing
  nothing in a browser tab.
- `app.thinking.save` (`Ctrl+S`) is on the map: the pane listed 89 of pi's 90
  actions.

### Changed

- **Keyboard** is rebuilt (Agent CLIs → a CLI → **Keyboard**). One row per
  action at 32px, the chord drawn as a keycap (mono, 24px — it used to be a
  36px box the height of a form control), the row's own **Add key** and
  **Reset** in a fixed column instead of an Add button chasing the chips, and
  a toolbar with the filter, a **Key** filter and facets that count:
  **Changed**, **Shared**, **Off**. A key two actions share is reported, not
  called a conflict — 52 of pi's 90 actions share one on purpose.
- A row that differs from pi's default wears the accent bar the settings rows
  already use, and the pane names the file it writes
  (`~/.pi/agent/keybindings.json`).
- The key map's own chords read as keys (`Ctrl+B`, `Page ↑`) rather than the
  file's spelling.

### Fixed

- **The default shown was not always pi's default here.** Nine actions bind
  differently on Windows and WSL — `Ctrl+-` is `Alt+Z` on WSL, and `Suspend`
  has no binding at all on native Windows. The pane reads the host's platform
  and prints the binding that machine actually has, with the other platforms'
  rows on a muted line beneath.
- **The CLI pane tabs no longer run past the card.** At 1024px the nine-tab
  strip (Launch…Connectors) was painted over the page gutter with its last tab
  cut mid-word; it scrolls inside the card now, which is what the active-tab
  reveal was always written for.
- **A destructive button reads destructive on desktop.** `.btn-danger` only had
  a hover rule there, so every confirm drew Remove and Cancel as the same grey
  button; the phone app had the fill all along. It now has the rest state on
  both (`Reset all` is the first one you will notice).
