### Fixed

- **On macOS, a running dev server answered "stopped" and every hidden row was
  forgotten.** PiCode decided whether a process was still alive by reading
  `/proc`, and treated a file it could not read as a process that had ended —
  correct on Linux, where `/proc` is always there, and wrong on macOS, where it
  never is. Stop reported success over a server that kept running, and the
  Servers panel's hidden list emptied itself on the next read. Liveness is now
  asked the portable way off Linux.
- **A preview of a project reached through a symlink served 404 for its own
  files.** The containment check resolved the file but compared it against the
  folder as written, so a project under a linked path — every path on macOS,
  where `/var` is `/private/var`, and any linked mount or home elsewhere —
  looked like it was escaping itself. Both sides are resolved now; a symlink
  that really does leave the folder is still refused.
