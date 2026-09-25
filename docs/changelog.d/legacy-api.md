### Removed

- **Old API routes and fields.** `GET /api/packages` (Pi's list in its old
  JSON) and `/api/pi-keys` are gone; read Pi's packages with
  `GET /api/packages/report?cli=pi` and its key map with
  `/api/cli-keys?cli=pi`, the routes the app already used. Workspaces no
  longer carry a single `agent` field; read `agents`.

### Fixed

- **Removing a workspace closes every agent's terminal pane**, not only the
  first agent's.
