### Fixed
- **Agent CLIs: no more one-request "Not found" flicker while a CLI updates itself.** A vendor updater rewrites its launcher in place; the catalog now retries once before reporting a CLI missing, so lifecycle menus stop flickering off during updates.
