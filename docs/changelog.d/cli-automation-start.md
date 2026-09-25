### Added
- **Automations can start runs on Claude Code, Codex, Grok or Hermes.** In an automation's *Start a new run*, pick the **CLI**. PiCode opens the automation's agent, types the prompt once the CLI is ready, waits for the CLI to finish, reads the cost from its session and closes the terminal. It never approves anything for the CLI: a run that stops at a question waits for you, and the Inbox says where.

### Fixed
- **Sending to Claude Code 2.1.282 and Codex 0.157.** Their composers changed (a word-by-word suggestion, a new effort row, a shortcuts row), so PiCode could not confirm a message reached them; it reads the new screens now.
