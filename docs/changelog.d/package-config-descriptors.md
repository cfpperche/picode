### Added
- **Packages: Configure now works for descriptor-declared packages.**
  pi-roles was the only package with a Configure button; any extension whose
  configuration is described (PiCode's catalog, or a `picode.config` manifest
  in the extension itself) now shows one, backed by a form that edits the
  package's own config file. Configurable today: pi-web-search (the model
  that backs native web search) and pi-compact (compaction policy — enabled,
  token/percent triggers, floor, cooldown, summarizer model, thinking and
  instructions). The files stay the packages' own source of truth: unknown
  keys survive saves, unset fields stay unset, and a broken file is reported
  and never silently replaced.
