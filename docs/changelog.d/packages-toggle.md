### Fixed
- Omp: Enable/Disable on one of the CLI's own extension rows now writes the
  setting Omp actually reads (`disabledExtensions`) instead of running
  `omp plugin disable`, which does not know a configured extension. A workspace
  entry is edited in `.omp/settings.json`; a user-level one goes through
  `omp config set`.
