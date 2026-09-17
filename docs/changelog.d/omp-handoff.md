### Added
- **Agent CLIs: "Continue in Omp…" works both ways.** Omp's own sessions
  move to any other CLI (it reads its on-disk transcript), and other CLIs'
  conversations arrive as native omp sessions you resume with
  `--resume <id>` — written at omp's own minimal session shape, verified
  live against omp 18.2.4 before shipping. Omp joins pi, Claude Code,
  Codex, Grok, Hermes Agent, OpenCode, Muse Code and Antigravity as a full
  handoff source and target.
