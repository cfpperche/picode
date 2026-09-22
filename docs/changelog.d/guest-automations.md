### Fixed
- **Automations reach agents of other CLIs.** A scheduled or webhook message to a Claude Code, Codex, Omp or other non-Pi agent is now pasted into that agent's terminal while it is open; before, it was always skipped as "agent in terminal" or "closed". A closed terminal now skips as **terminal closed**, with the reason in the Inbox.
- **Settings, Packages, Providers and Connectors open the right CLI** from the user menu and the Ctrl+K palette when the selected tab is a non-Pi agent's terminal; they fell back to Pi.
- **Undo after removing a free agent** recreates it with its CLI, provider and model instead of as a Pi agent.
