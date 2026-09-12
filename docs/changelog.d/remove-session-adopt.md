### Removed

- **From a Pi session** in the create dialog (desktop and mobile) and its
  API, `POST /api/clis/{cli}/sessions/adopt`. Agents are born only from
  new sessions (ADR-0126, superseding ADR-0021). Agents already created by
  adoption keep working; session files were never modified. The sessions
  surface keeps list, delete and auto-clean.
