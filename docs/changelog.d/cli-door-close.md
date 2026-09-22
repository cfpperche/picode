### Changed

- A CLI's sign-in terminal no longer shows up in the sidebar or among the
  CLI's terminals; it lives on the Providers card that opened it. PiCode
  closes it once the account is saved, when the login exits, after 15
  minutes, or when you press **Cancel** — and the card says when it closed.

### Removed

- `POST /api/clis/{cli}/terminals`. Starting a CLI goes through
  `POST /api/agents` or `POST /api/workspaces/{id}/agents` with `overrides`.
