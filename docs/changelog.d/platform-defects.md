### Fixed
- **Windows builds again.** `internal/server` used Unix-only process calls
  (`Setpgid`, `syscall.Kill`) in the GUI sign-in paths, so it had not compiled
  for Windows since they landed — the leg that checks that only runs on tags
  and manual runs. The calls live behind per-platform files now, the way the
  agent-stop and zombie helpers already do.
- **A folder that is gone still counts as its workspace's own.** The dashboard
  attributes a session's recorded folder to a workspace by comparing canonical
  paths; a folder that no longer exists could not be canonicalised at all, so on
  macOS — where `/var` symlinks to `/private/var` — it matched nothing. The
  deepest existing part is resolved and the rest re-appended.
