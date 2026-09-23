# 2026-09-22 — feat/deploy-lock-fixture

Deploys queued for hours behind a docs-fixture tmux server that was not a
deploy: the fixture (started by `make deploy` → docs-shots, from a PiCode
terminal) could not `kill-server` its private server — the session guard
refuses it on any socket — then deleted its directory, leaving a socketless
server that had inherited the flock descriptor. Freed live by the owner
(SIGUSR1 to recreate the sockets, exact-name kills); a second fixture was
running a real `claude` login from the pre-hardening sign-in regression.

Fix: `flock -x -o` for deploy and desktop-restart (verified: a surviving child
no longer holds the lock); `tmux.Binary()` skips PiCode intercept wrappers for
PiCode's own tmux calls, with `refuseUserServer` now also refusing the
instance socket under `~/.picode`; the fixture ends a previous run's server
before reusing its port directory and keeps the directory when a kill fails;
docs-shots waits for the fixture and warns if its socket survives. Verified
from a guarded PiCode terminal: SIGTERM leaves no server and no directory; a
SIGKILLed fixture restarted on the same port ends the old server first.
