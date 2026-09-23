### Added
- **Removing an agent asks how it went.** The remove dialog, on the desktop and the phone, asks an optional question — Resolved, Partly, Didn't resolve, Just trying — and, when it did not go well, what got in the way. It never blocks Remove, it skips agents that never worked or lived under a minute, and **Stop asking** turns it off.
- **Outcomes page.** The user menu ▸ Outcomes lists every removed agent with your answer, its setup and what PiCode saw (how long it lived, turns, how often it asked for you), with filters, a later answer, record deletion and a JSON-lines export. Records stay on this machine; environment variables are kept by name only.
- **Agent outcomes on the dashboard**: removed agents, the share resolved and the reason most in the way for the chosen range.

### Changed
- Removing a workspace, or an agent's terminal, now keeps a record of each agent it ends.
