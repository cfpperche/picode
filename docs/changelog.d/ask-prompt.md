### Added
- **Browser permissions: Ask.** A permission kind can be set to Ask in
  Settings ▸ Browser ▸ Site settings. When a site asks, the prompt appears in
  that browser tab with Allow, Block, and Always allow — the last remembers
  the answer for the site. An unanswered prompt denies itself after a minute
  instead of leaving the site waiting.

### Fixed
- Choosing “Platform default” in Site settings now clears the per-kind
  policy; the choice failed before.
- The Manage dialogs (Site settings, passwords, and the rest) scroll inside a
  short window instead of running past its bottom edge, where Close was
  unreachable.
- A work-browser tab no longer re-arms its page-metadata poll on every
  render, which pinned a CPU core while the tab was open.
