### Added
- **Packages: Configure now works for descriptor-declared packages.**
  pi-roles was the only package with a Configure button; any extension whose
  configuration is described (PiCode's catalog, or a `picode.config` manifest
  in the extension itself) now shows one, backed by a form that edits the
  package's own config file. First up: pi-web-search — the model that backs
  native web search (`~/.pi/agent/web-search.json`) can be viewed, edited and
  cleared from the Packages view. The files stay the packages' own source of
  truth: unknown keys survive saves, and a broken file is reported and never
  silently replaced.
