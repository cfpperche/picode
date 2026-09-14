### Added
- **Agents can read the page open in the work browser.** A new `browser` tool
  (`snapshot`, `screenshot`, `events`) reads the tab on screen in the desktop
  app and answers with its structure, a PNG path, or what the tab recorded.

### Security
- Read is the default and the ceiling: an agent with no grant reads the tab the
  human has on screen, and cannot click, type or navigate. Anything more needs
  an explicit per-agent grant (Settings ▸ Browser, next).
