### Added

- **One credential vault for every agent CLI.** Claude Code, Codex, OpenCode,
  Grok, Hermes, Muse, Antigravity and Omp now have a Providers pane of their
  own: see which accounts the machine holds, import the login a CLI already
  has, add an API key, verify it, pause it or sign out — one account per
  provider or ten.

### Changed

- Pi's provider accounts moved into an encrypted vault
  (`credentials.json` beside its key, both readable only by you). Existing
  accounts are carried over the first time PiCode starts after the update;
  the old file is left where it was.
- A backup with secrets carries the vault but never the key that opens it: a
  snapshot restored on this machine keeps working, and a copy taken to another
  machine cannot be decrypted. That is the point of storing the keys
  encrypted, and it is the one thing to know before moving a backup around.

### Fixed

- Verify on a saved key spends exactly one listing call to the provider, named
  on the button, and stores the answer with its age on the account's row.
  Nothing else in the app talks to a provider about credentials.

### Security

- Credential values never leave the server: the roster shows a masked hint
  (`sk-ant-…f2a`) at most, and a vault that cannot be read (missing key,
  tampering) is reported in words instead of being silently replaced.
