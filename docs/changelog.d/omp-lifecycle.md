### Added
- **Agent CLIs: Omp installs and updates per install method.** A missing
  Omp now offers **Install** through npm (like pi, Codex and Claude Code).
  Update checks follow the install: npm installs read the npm registry,
  native installs run omp's own `omp update --check` and parse its answer.
  A bun-global Omp is honestly reported as unmanaged — measured, npm
  would miss the bun copy and omp's updater resolves by PATH — so no
  update or uninstall buttons propose an action that would hit the wrong
  installation.
