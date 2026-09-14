### Added
- **A grant can let an agent act.** With the `act` tier, two verbs join the
  browser tool: `evaluate` (run an expression in the page) and `navigate` (go
  to a URL). Without a grant both are refused by name, with the way to grant
  them.

### Fixed
- The `events` verb now honours `since`: the sequence number the agent last saw
  travelled at the top level of the request, where the channel never read it,
  so every poll replayed the whole ring.
