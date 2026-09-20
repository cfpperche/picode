### Fixed

- **A guest CLI's setting could be written to the wrong key.** In YAML, a path
  like `memory.memory_enabled` matched a block of the same name nested anywhere
  else, so the save landed on it and the key the pane showed never changed. The
  locator now anchors the first segment at the document's top level and keeps
  every next one inside the block its parent opened.
- **TOML writes could land inside a multi-line string.** A `"""…"""` body
  containing `key = value` or `[table]` lines was scanned as configuration,
  which rewrote the user's prose and could silently retarget a save. The
  scanner now tracks string state, and an array of tables is refused rather
  than edited.
- **JSON writes could shadow a value.** A key present twice was written on the
  copy every parser ignores, and a path whose parent was a string, an array or
  null appended a duplicate member at the root. Both are refused with a
  sentence naming the file.
- **Handing a key back could delete more than the key.** Emptying a YAML block
  swallowed the comments and blank lines that followed it, which emptied a file
  whose block was followed by commented-out configuration.
- A file with no final newline, and a JSONC file with a comment before its
  closing brace, can now take a new key instead of refusing every save.
- **Claude Code's Approvals row could not show the value in the file.** The
  options were missing `auto` and `dontAsk`, so a machine set to `auto` drew a
  blank control and any choice moved it off. OpenCode's `autoupdate` row is
  withdrawn: it is `true`, `false` or `notify`, and a switch would destroy the
  third. Grok's options now match what that CLI accepts.

### Security

- **A memory could be written outside its folder.** Containment resolved the
  target file, which does not exist when one is being created, so a write
  through a symlinked subdirectory landed outside. It now resolves the deepest
  existing ancestor, and listing skips anything that is not an ordinary file —
  a symlink named `notes.md` had its first line published, and a FIFO hung the
  pane.
- A relocated memory folder is bounded to the user's home directory, so the
  pane cannot be pointed at `/etc` or `/proc` from the Settings pane.
- Masking covers more credential shapes (cloud secrets, `password=` and
  `api_key:` assignments, passwords in URLs, Slack webhooks, PGP and truncated
  key blocks) and no longer destroys ordinary words: `risk-assessment-…` was
  being redacted.
- Reading a memory refuses anything that is not an ordinary file, so a
  pseudo-file reporting size 0 no longer walks past the size limit.
