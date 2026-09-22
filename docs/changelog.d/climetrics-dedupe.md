### Fixed
- **Dashboard: Claude Code, Codex and Grok tokens are counted once.** Claude Code repeats a response's usage on every block it writes, Codex forks copy their parent's history, and Codex and Grok count cached tokens inside input; the dashboard summed all of it and showed roughly twice the real token totals for Claude Code and Codex.
- **Dashboard: Claude Code subagents are counted.** Their transcripts live one folder deeper than the dashboard looked, so their tokens and tool calls were missing.
- **Dashboard: Grok usage comes from its update stream.** Sessions that never wrote `usage.json` now report tokens and cost.
