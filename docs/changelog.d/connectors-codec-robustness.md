### Added
- **Connectors tolerate vendor JSONC configs on read.** Trailing commas and
  comments that OpenCode and friends hand-write into their JSON configs no
  longer break the Connectors pane; every save still lands strict JSON. A
  blocked config file shows one line naming it, with **Open** and **Retry**;
  other config layers keep listing.

### Fixed
- A malformed connector config file (JSON, TOML or YAML) no longer fails
  the Connectors pane with an error — the pane names the file and carries on.
