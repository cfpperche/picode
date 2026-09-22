### Fixed

- The packages pane on every agent CLI now names the context you are bound
  to in its scope switch — Global, <workspace>, <agent> — the way Pi's
  always did, instead of a generic "This workspace" that could be any of
  them. A pane bound to no workspace stops offering the workspace scope
  rather than promising a scope that cannot install.
