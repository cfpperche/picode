### Added

- **Omp Models pane.** `#/clis/omp/models` lists every model Omp reports it can
  reach in the workspace, grouped by provider, with kind chips, a filter,
  context size and price. **Allowed** edits `enabledModels` (Omp then uses only
  those, and the pane warns when none of them is reachable in the folder);
  **Hide provider** / **Show** edit `disabledProviders`. Both layers; a toggle
  writes the layer's complete list, because omp's arrays replace the parent's
  rather than extend it (ADR-0181).
- The pane says when a layer leaves Omp with no usable model (an allowed list
  naming only models the folder cannot reach) and offers **Allow every model
  here**; a layer that hides every provider says so and offers **Show all**.
- The model catalog answer is kept while the config files it depends on are
  unchanged, so reopening the pane is instant; **Refresh** asks Omp again.
