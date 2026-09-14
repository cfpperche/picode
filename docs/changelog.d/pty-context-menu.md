### Changed

- One right-click menu for every terminal pane (managed Pi in-terminal, Agent CLI, plain shell). Chat and the composer stay on the generic menu.
- **Send to terminal…** on an in-terminal Pi pastes into the TUI (palette Send snippet too); the sheet says “Sent to the terminal.”
- **Attach files…** / **Ask Pi about this** on an in-terminal Pi use `/api/agents/{id}/drop` and `/prompt`, not a terminal id.
- **Rename agent…** / **Remove agent…** (existing cleanup confirm, never “This stops the tmux session.”). **Terminal settings** from an agent pane opens global **Terminal defaults**.
- **Continue in…** on an in-terminal Pi uses that pane’s session file and asks to continue while the TUI is still writing.
- Close browser split actually appears when the split is open.
