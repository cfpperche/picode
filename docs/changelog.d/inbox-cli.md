### Added
- **`picode inbox notify|ask` — the Inbox door for any CLI, no MCP config
  needed.** Any guest CLI (Claude Code, Codex, a plain script) can file an
  FYI into your Inbox with one command, or ask you a blocking question:
  `picode inbox ask --question "…" --wait` polls until you answer in the
  Inbox and prints your reply on stdout, hours if needed. Same daemon
  discovery as `picode mcp` (`--url`, `PICODE_URL`, or server.json), same
  items, same app.

### Fixed
- Answering a question filed by a CLI with no PiCode identity (a guest
  running outside PiCode) now records the answer on the item, where the
  asking CLI picks it up. The Inbox used to refuse with "no reply channel"
  and the item stayed open forever, leaving the asker waiting on a poll
  that could never end.
