### Changed
- **Connectors get a Marketplace, like Packages.** The pane now has **Installed | Marketplace** tabs: search a curated catalog, pick **Save to**, press **Add** — the dialog is gone. PiCode's own connectors stay pinned at the top, and every agent CLI's pane works the same way.
- **The catalog goes beyond the hand-picked few.** Marketplace search covers a curated slice of the official MCP Registry (verified, remote servers), synced in the background and cached — it answers even when the registry is unreachable.

### Removed
- **Connector file import and "use from another app" are gone.** The Pi adapter already imports, and each CLI's Connectors pane writes that CLI's own config directly — adding the same service in its marketplace replaces copying it from another app. **Custom server…** stays.
