### Fixed

- Settings ▸ Browser and Settings ▸ Computer now fill the full page card
  instead of a narrow 780px column, and all row controls share the 36px
  control height.
- Agent permission lists carry workspace provenance on every row (workspace
  chip resolved via `/api/workspaces`, plus managed/terminal kind), with a
  search + filter toolbar and paged rendering instead of one unbounded list.
- Recent steps (Computer) and Raw calls (Browser) are filterable by search
  and outcome, with expandable rows showing the full reason, actor and
  timestamp instead of a truncated single line.
