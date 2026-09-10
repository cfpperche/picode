### Added

- Workspace Communication on desktop and mobile: select managed Pi agents and native terminals, apply their connections, and follow preparation and activity in one view.
- Run a connection test through the participants' native tools. A test passes only after a message, correlated reply and both acknowledgments; pending and unconfirmed results remain visible.

### Changed

- Participant selection carries forward to future conversations in the selected workspace. Each conversation keeps a separate credential; clearing a selection revokes its connections atomically.
- Normal setup offers Apply and connect, Open and connect and repair actions, with per-conversation configuration under Advanced.

### Fixed

- Pi connection setup shares its receiver registration, preserves native input and rejects stale process or conversation identity.
- Communication preparation resumes the exact idle terminal conversation only after verifying the previous writer has exited; an unverified shutdown blocks replacement.
- Managed Pi reconnect checks pending delivery, native commands and approvals before stopping. A stale stop cannot remove its replacement.
