### Added
- **Installed plugins group by where they came from.** A list that mixes origins
  (Hermes' bundled plugins beside yours, Claude Code's claude.ai-managed ones
  beside the machine's) now carries a header per group — `Installed by you`,
  `Ships with the CLI`, `From a marketplace`, `Managed by claude.ai` — keyed on
  the vendor's own provenance word. A single-origin list gets no header.
- **The plugin filter says what matched.** The run of text a filter matched is
  highlighted in the name and in the description, instead of the card merely
  surviving the filter.
- **A marketplace chip filters the catalog.** Clicking a marketplace name — on a
  catalog card or in the sources row — narrows the list to that marketplace, and
  clicking it again clears the filter.
- **An empty plugin list points at the CLI's own catalog.** Where the CLI can
  install and its catalog was read, the empty state offers up to three real
  entries with Install, instead of only a link to the Marketplace.
- **The toolbar rides along.** The filter and the update check stay pinned while
  a catalog of hundreds of rows scrolls under them.

### Fixed
- **Claude Code's installed plugins no longer disappear.** The pane reads the
  machine scope by leaving `scope` out of the query, and the roster read passed
  that empty scope through: Claude Code's rows name their scope (`user`), so
  every one of them was filtered away and the pane showed an empty list for a CLI
  that has plugins. The read now resolves the scope once — empty is the machine
  scope — for the list, the catalog and the update check, which is also the key
  they share in the cache. Regression test: `TestTheDefaultScopeReadsTheMachineScope`.
