### Fixed
- The Windows installer no longer hides a failure: the launcher waits for the
  elevated run and reports its exit code, the last error line is written to
  `%ProgramData%\PiCode Desktop\install.log`, and the elevated window pauses
  for Enter instead of closing with the evidence.
- `install --user <name>` now aims the picode binary and pi at that account
  (and `--user root` keeps the system-wide install); stage scripts no longer
  rely on `$` surviving the `wsl.exe` argument boundary.
