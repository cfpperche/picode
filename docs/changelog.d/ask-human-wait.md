### Changed
- **`ask_human` over MCP waits for you.** The call now stays open until
  you answer in the Inbox (hours if needed), keeping the CLI's tool call
  alive with progress reports, instead of handing the wait back to the
  agent every 90 seconds.
