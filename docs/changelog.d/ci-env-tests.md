### Fixed

- **`picode inbox notify` diagnosed a missing flag as a missing daemon.**
  Leaving out `--title` answered "PiCode is not running" and exited 1, because
  the command looked for the daemon before reading what you typed. It now
  validates first, the way `picode inbox ask` already did, and says which flag
  is missing with the usage exit code.
