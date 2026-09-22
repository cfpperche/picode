### Added

- **Omp's own extensions are listed in the packages pane.** Omp loads these from
  its own settings rather than from its plugin store, so the pane showed the
  vendor's plugins and nothing else. Each configured entry now has its row — the
  CLI's own name for it, the path it resolves to, the layer that declared it
  ("This machine" or "This workspace") and whether it is switched off — and
  removing one takes it out of the layer that named it: PiCode edits the
  workspace's `.omp/settings.json` (every other key left exactly as it was), and
  the user layer goes through Omp's own `omp config set extensions`.
