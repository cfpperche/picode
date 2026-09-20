### Added

- **Settings for every agent CLI.** Claude Code, Codex, Grok, Hermes Agent,
  OpenCode, Muse Code, Antigravity and Omp now have a real Settings pane at
  `#/clis/<cli>/settings` that edits the CLI's own config file — model,
  approvals, sandbox, reasoning effort, memory switches and more, each row
  named with the vendor's own key. A CLI with one config file shows one layer;
  one with a workspace file shows both, with the file it writes named under the
  switcher and provenance on every row (ADR-0163).
- **A Memory pane for every agent CLI** at `#/clis/<cli>/memory`, showing what
  the CLI has remembered between sessions. Claude Code, Hermes, Muse Code and
  Omp are read, edited and pruned from the browser wherever they keep plain
  markdown; Grok and Codex are read here and cleared with their own command,
  because they generate those files; Pi and OpenCode say in one line that they
  keep no memory, and Antigravity says PiCode cannot confirm one (ADR-0163).
  Omp writes nothing until its memory backend is turned on, and PiCode reads
  its folder from two paths the vendor does not document, so that pane is
  usually empty.

### Changed

- Saving a guest CLI's setting keeps the rest of the file byte-for-byte:
  comments, key order and every key PiCode does not know survive, and a file
  that changed on disk since it was read is refused instead of overwritten.

### Security

- Memory files are masked on read for credential-shaped values (provider keys,
  tokens, private-key blocks). The file keeps what the CLI wrote; the browser
  does not echo a secret back, and memory text never reaches the change feed.
