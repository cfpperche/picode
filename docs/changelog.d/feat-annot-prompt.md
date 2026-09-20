### Fixed
- Annotation Send delivers again: it posts to the prompt door
  (`{message, paths}` pasted into the agent TUI), not the file-drop door —
  which answered 400 while the toast blamed the terminal. A refused paste
  now names the server's reason.
