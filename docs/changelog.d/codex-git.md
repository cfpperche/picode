### Added

- **Codex's Memory pane shows its version history.** Codex keeps that folder
  under git, so the pane now says how many revisions it holds, when the last
  one was written, and whether any file has changed since — and each row's
  Modified date comes from the revision that actually changed the file instead
  of the filesystem's timestamp, which moves every time Codex rewrites a file
  with the same contents. Any memory store the CLI keeps under version control
  gets the same line.

### Changed

- The Memory pane no longer sits flush against the tab bar, and its toolbar is
  one cluster on the left: the store switcher and the filter sit together
  instead of the filter floating alone against the right edge. The row is gone
  entirely when there is nothing to switch and nothing to filter.
- A read-only CLI now names its clear command inside the sentence, with a plain
  **Copy** button — the button used to read "Copy grok memory clear".
