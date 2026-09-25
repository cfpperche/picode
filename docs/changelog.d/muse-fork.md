### Added

- **Fork agent…** now works for Muse Code agents: PiCode asks Muse's own session server to copy the conversation, opens the copy with `muse resume`, and sends the task as soon as Muse is ready (a task that cannot be delivered lands in the Inbox with its text).

### Fixed

- Resuming a Muse Code conversation (Resume last session, Continue in… to Muse) uses `muse resume <id>`; Muse Code 1.3.0 refuses the older `--resume` flag.
