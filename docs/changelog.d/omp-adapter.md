### Added
- **Agent CLIs: Omp is a full citizen.** Editable launch settings, the PATH
  wrapper with a presence lease (maintenance subcommands and protocol modes
  stay out of it), an **Activity reporting** toggle through omp's own
  extension API (it reports Ready and Working like pi), and **Check for
  updates** — `omp update` on native installs, npm on npm installs. omp
  refuses to start when launch arguments carry `--trusted-extension` next
  to PiCode's activity extension, so that combination is named in the
  launch preview instead of starting a broken terminal. Still needs
  Bun ≥ 1.3.14 on PATH.
