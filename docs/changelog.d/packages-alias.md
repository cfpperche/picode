### Fixed

- The packages pane groups a CLI's plugins by the vendor's own provenance
  again (`From a marketplace`, `Ships with the CLI`, `Local plugin files`, …):
  a row carries that fact as `kind`, and the pane was reading the old alias's
  name for it.

### Removed

- The `/api/cli-packages*` API family (ADR-0176's one-release alias) is gone.
  Every CLI's package verbs — its catalog and its configured sources, install,
  remove, update, toggle, marketplace and inspect — are on `/api/packages*`,
  with the CLI named in the request (`cli`), and the answers are the unified
  report every pane already reads.
