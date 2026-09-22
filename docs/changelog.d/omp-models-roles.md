### Changed

- **Omp's model roles moved to the Models pane.** Roles, the quick-switch cycle,
  fallback chains and the retry settings now sit above **All models** on
  `#/clis/omp/models`, the way Omp's own model hub keeps them on one screen. A
  role whose model is not reachable in the folder, or not in the allowed list,
  says so on its own row. Settings keeps the thinking level, memory and
  interface rows.
- A fallback chain reads as what it is for — "When default fails", "When any
  openai model fails" — and the Fallbacks help says what `@smol` means.

### Fixed

- A list row's **Use inherited** sat alone at the right edge, in Settings as
  in Models; it now sits under the list.
