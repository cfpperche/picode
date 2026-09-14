### Added
- **The shell enforces the browser grant at navigation time**: once an
  act-tier command drives a tab, agent-caused loads outside the agent's
  domains — script redirects included, not just the `navigate` verb the
  daemon pre-checks — are cancelled. User-initiated navigation is
  untouched; navigating the pane from its own toolbar disarms the gate
  until the agent acts again.
