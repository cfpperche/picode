### Changed
- **The Fleet tile counts agent CLI terminals, not just managed agents.**
  The sidebar's Claude Code, Codex, Grok, Hermes and OpenCode terminals were
  invisible to the dashboard, which could read `1 / 1 running` beside seven
  live terminals. The tile now shows `N live` (agents + agent CLIs), states
  `working` / `need you` / `idle` / `no signal`, names each one, opens its tab
  on click, and counts plain shells apart — `no signal` means a CLI is running
  but has not reported activity, never that it is idle.
