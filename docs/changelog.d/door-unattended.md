### Fixed
- **Automations and the Chrome extension no longer paste into a CLI's login or menu screen.** When the terminal of a Claude Code, Codex or other agent is open but not at a prompt PiCode recognizes, the run is skipped as **unrecognized** and nothing is typed; before, it was pasted blind and recorded as done.
- **The Chrome extension sends to agents of every CLI.** A non-Pi agent gets the page link and your message at its prompt while its terminal is open; screenshots and Act on this page still need a Pi agent.
- **Provider usage works without Pi.** The usage summary and its background refresh read the signed-in accounts from the vault instead of Pi's model list, so the Providers roster of every CLI keeps its usage on a machine without Pi.
- PiCode no longer checks Pi packages against npm every 30 minutes on a machine where Pi was never used.
