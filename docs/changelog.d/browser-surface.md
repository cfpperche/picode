### Added

- Browser surface: a watch-only live view of the agent's browser, docked as the agent tab's third view (Chat | TUI | Browser). Frames stream through a new authenticated daemon proxy (`/ws/browser`) of the browser engine's local stream; the view shows the page URL, a Watch-only chip, and honest empty/ended states. Interactive control arrives in a later phase. (ADR-0114)
