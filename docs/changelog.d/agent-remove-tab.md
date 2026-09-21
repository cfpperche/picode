### Fixed

- **Desktop/Web:** removing the agent whose tab you are on no longer lands on a
  dead surface. Selection now moves to the neighbouring tab (right, else left),
  and to the dashboard when no tabs remain. Closing a tab picks the same
  neighbour instead of jumping to the last tab. A CLI agent's bound terminal
  tab and a removed workspace's tabs follow the same rule.
