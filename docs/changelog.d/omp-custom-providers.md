### Added

- Providers: Omp now has the same custom-provider surface as Pi — **Custom
  provider** on Omp's pane adds a gateway (base URL, API type, models) by
  merging into Omp's own `~/.omp/agent/models.yml`, leaving hand-edited
  providers, unknown fields and comments untouched (ADR-0175). Omp's API key
  lives inside the definition (the only place a custom Omp provider can carry
  one); it is never shown back, only spent on the gateway — Verify and Load
  models read it from the file. The form offers Omp's accepted options only,
  and definitions appear as rows on the pane with Edit / Verify / Remove.

### Changed

- Providers: the custom-provider form's URL hints and help lines no longer
  name Pi specifically; each CLI's Advanced section carries only what that
  CLI reads (Pi keeps the full set).
