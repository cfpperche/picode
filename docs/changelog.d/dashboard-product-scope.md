### Changed

- Web: the dashboard no longer offers a folder-scope filter — the "This machine | PiCode" chips are gone and `GET /api/sessions/stats` measures the whole machine only (ADR-0127). `Spend by workspace` keeps ranking every folder, naming claimed workspaces and leaving the rest as their own folder; the payload's `scope` field and the `?scope=` parameter are removed.
