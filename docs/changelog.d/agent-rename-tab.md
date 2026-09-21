### Fixed

- **Renaming an agent left its editor tab on the old name.** A non-Pi agent
  runs as its bound terminal (ADR-0160), and the tab strip labels that
  `t:<id>` tab with the terminal's own name — a value copied from the agent
  once, at bind time. The rename route only patched the agent row, so the
  sidebar showed the new name while the tab kept the one from creation.
  `Store.UpdateAgent` now renames the bound terminal inside the same
  mutation and announces `terminal.updated`, so the strip, the dashboard
  fleet and every other `term.name` reader follow the rename; a same-name
  patch emits nothing.
