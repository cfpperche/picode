### Added
- **Windows Hello and passkeys** (Settings ▸ Browser ▸ Password manager): a
  row that opens Windows' own Sign-in options. PiCode does not store or
  unlock passwords and passkeys — Windows does — so the row points at the
  screen that manages them instead of imitating it. It appears only in the
  desktop app, and it is the first target class besides `http(s)` that the
  shell will hand to the operating system.
