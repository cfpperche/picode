### Changed

- A guest agent's launch now carries `PICODE_AGENT_ID`, so `picode mcp`
  and the computer/browser tools resolve the agent as the caller — grants
  given to the agent in Settings match without per-terminal setup.
- Inbox "needs you" items for a guest are filed for the agent (not the
  terminal), close when the agent or its terminal is deleted, and their
  Open terminal action follows the agent's bound terminal.
