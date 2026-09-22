### Fixed
- **Muse Code's Connectors pane writes a file Muse accepts again.** The driver wrote the MCP block under `mcp_servers`; Muse reads `mcpServers`, so a server PiCode added never started — and a file carrying both keys made Muse drop the block entirely. Measured against the installed CLI (`muse mcp --help` names `mcpServers`) after a vendor-session sweep found the mismatch.
