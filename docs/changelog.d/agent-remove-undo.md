### Added

- **Desktop/Web:** removing an agent now offers **Undo** in the confirmation
  toast. Undo re-creates the agent in the same workspace with the same name,
  CLI and config, re-attaches its session history, and opens its tab again.
  The old id is gone for good (automations pointing at it stay broken), and
  sessions only come back if the dialog's "Also delete sessions" was not
  picked. Per-agent package selection is the one setting Undo cannot carry
  over.
