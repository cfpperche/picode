### Fixed

- Agent CLIs: the page no longer sticks on its loading skeletons when the terminal list is slow — `GET /api/terminals` now serves a shared snapshot (singleflight + 1s TTL) computed by a bounded worker pool, and the page keeps the last result that landed instead of discarding every response older than the newest request.
- Agent CLIs: web-tab ids (`w:<n>`) no longer reach `/api/agents/{id}/slash` and `/role-state` as agent ids (console 404s).
