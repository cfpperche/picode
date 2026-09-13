### Fixed
- **Terminal side padding is even.** xterm's fit floors the column count, so
  unused pixels collected on the right of every pane (desktop app, browser,
  and phone). The leftover is now split so both sides match the terminal
  padding preference.
