### Added
- **Work browser: the shell now speaks CDP itself.** The desktop app carries
  a command bridge over the page host API, so a page's DevTools protocol is
  reachable without any open debug port. Read-tier page events (navigation,
  console, network, log) are recorded per tab for whoever polls them.

### Security
- **The desktop debug port is off by default.** It was opened on loopback in
  every launch; it is now an explicit `PICODE_CDP_PORT` opt-in for external
  tooling. The agent access tiers (`read` / `act` / `full`) are enforced
  against a named command catalog, and an unnamed command is refused at every
  tier.

### Fixed
- `make desktop-shell` actually builds the Windows shell again — the target
  was shadowed by the `desktop-shell/` directory and reported "up to date".
