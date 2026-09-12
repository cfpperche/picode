### Changed
- Agent CLIs strip is **CLIs | Messages**. Settings, Packages and Connectors
  (MCP) are panes of the selected CLI, next to Launch / Terminals / Sessions /
  Providers. Webhooks stay a PiCode page. Old Settings, Packages, Integrations
  connectors and `#/mcps` addresses rewrite.
- Providers pane dropped the “This machine” lecture; empty llama.cpp is one
  line plus Set up.

### Fixed
- Connectors opened from the composer, user menu, palette or More keep the
  selected agent and workspace in the URL, so “This agent” and reload after
  Add still work. Free agents resolve the same way as Packages.
- `#/packages` and `#/settings` without a query adopt the selected pane
  again once the fleet is ready. `#/clis/connectors` rewrites to Pi.
  Extra path on Settings is blocked instead of opening the editor.
