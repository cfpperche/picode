### Changed

- **Agent CLIs → Connectors is a roster, not a catalog grid.** The pane opens
  with one primary action (**Add connector**), then the configured services:
  each row carries its state, the layer it lives in (title: the config file),
  the target, a switch and an overflow menu (**Sign out**, **Remove**, whose
  confirmation names the file). The catalog, the file picker and the target
  pills left the pane and became one dialog with a search field, a labelled
  **Save to** control, and two quiet secondary entries (*Custom server…*,
  *Import a file…*).
- The redundant workspace line under the Connectors tab bar is gone (it
  repeated the scope already shown per row), and every control in the pane is
  the 36 px control height instead of a 28 px pill beside a 36 px button.
- Live status is only claimed while an agent runs: with the agent stopped the
  pane says so once, instead of showing `Idle` on every row. A server that
  needs a login shows **Sign in** on its row even with the agent stopped.
- The install target is part of the route (`?scope=`), so it survives a
  reload, and it is stated where the write happens instead of above the list.

### Fixed

- Mobile rendered the Connectors pane without its stylesheet (the CSS was only
  imported by the webhooks view, which mobile loads lazily). The pane imports
  its own styles now.
- Removing the unreachable second "MCPs" view (`#/mcps` already rewrote to the
  Connectors pane) leaves one rendering path for both apps.
