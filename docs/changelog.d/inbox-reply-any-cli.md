### Fixed

- Replying in the Inbox to a question from Claude Code, Codex, Grok or any other non-Pi agent no longer fails with "The terminal session could not be identified safely". An agent still waiting in `ask_human` gets the answer as the tool's result. If it stopped waiting, the reply is typed into its terminal. If it is not running, the answer stays on the item, and the item says the agent was not told.
- The mobile Inbox's Reply now reaches a Pi agent running in its terminal, as the desktop Inbox already did.

### Added

- Omp agents load Pi's Inbox reply receiver, so a reply arrives as the agent's own turn, confirmed against its session file, as it does for Pi.
