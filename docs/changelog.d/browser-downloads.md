### Added
- **Downloads** (Settings ▸ Browser): Location shows where the built-in
  browser saves files (the system Downloads folder until changed), with a
  Change dialog; "Ask where to save downloads" decides between a save
  prompt and writing straight to the folder; Download history lists every
  file with its size, time and outcome, searchable, with Open file /
  Show in folder / Copy path / Remove per row and a two-step Clear all.
- The shell reports each download (start and outcome) through
  `btab://download`; the list lives in the daemon store
  (`GET/POST /api/browser/downloads`, `POST /api/browser/downloads/status`,
  `DELETE /api/browser/downloads/{id}`, `POST /api/browser/downloads/clear`).
